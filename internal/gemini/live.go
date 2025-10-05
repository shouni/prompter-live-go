package gemini

import (
	"context"
	"errors"
	"fmt"
	"io"

	"prompter-live-go/internal/types"

	genai "google.golang.org/genai"
)

// LiveSession は確立された Live API の WebSocket セッションを抽象化します。
type LiveSession interface {
	Send(data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close() error
}

type LiveClient interface {
	Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error)
}

type liveClient struct {
	client *genai.Client
}

// SDKの Live.Connect が返す、ストリーム操作に必要なメソッドを持つインターフェースを定義
type sdkLiveStream interface {
	Send(*genai.Part) error
	Recv() (*genai.Content, error)
	Close() error
}

type liveSessionWrapper struct {
	session sdkLiveStream // ストリーム操作を担うSDKのオブジェクト
}

func NewLiveClient(ctx context.Context, apiKey string) (LiveClient, error) {
	// genai.ClientConfig を使用
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	return &liveClient{client: client}, nil
}

func (c *liveClient) Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error) {

	// 1. types.ResponseModalities ([]string) を SDK の型である []genai.Modality に変換します。
	sdkModalities := make([]genai.Modality, len(config.ResponseModalities))
	for i, mod := range config.ResponseModalities {
		// genai.Modality は単なる string のエイリアスであるため、直接キャストします。
		sdkModalities[i] = genai.Modality(mod)
	}

	// LiveConnectConfig を構築
	sessionConfig := &genai.LiveConnectConfig{
		// SystemInstruction は *genai.Content 型へ変換
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: config.SystemInstruction}}},

		// 2. 正しい型である sdkModalities を設定します。
		ResponseModalities: sdkModalities,
	}

	// Connect メソッドは (ctx, modelName, config) のシグネチャを期待
	// 戻り値の型がインターフェースではないエラーを回避するため、戻り値を interface{} で受け取る
	var sdkSession interface{}
	sdkSession, err := c.client.Live.Connect(ctx, config.Model, sessionConfig)
	if err != nil {
		return nil, err
	}

	// interface{} で受け取った sdkSession が、期待する sdkLiveStream インターフェースを満たしているか確認
	stream, ok := sdkSession.(sdkLiveStream)
	if !ok {
		// もし型が一致しない場合は、エラーを返す
		return nil, fmt.Errorf("Live.Connect returned unexpected type %T; expected an object with Send/Recv/Close methods", sdkSession)
	}

	return &liveSessionWrapper{session: stream}, nil
}

// --- liveSessionWrapper のメソッド実装 ---

func (s *liveSessionWrapper) Send(data types.LiveStreamData) error {
	if s.session == nil {
		return errors.New("live session is not initialized")
	}
	part := &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: data.MimeType,
			Data:     data.Data,
		},
	}
	return s.session.Send(part)
}

func (s *liveSessionWrapper) RecvResponse() (*types.LowLatencyResponse, error) {
	if s.session == nil {
		return nil, errors.New("live session is not initialized")
	}
	content, err := s.session.Recv()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return &types.LowLatencyResponse{Done: true}, nil
		}
		return nil, err
	}
	text := ""
	if len(content.Parts) > 0 && content.Parts[0].Text != "" {
		text = content.Parts[0].Text
	}
	return &types.LowLatencyResponse{Text: text, Done: false}, nil
}

func (s *liveSessionWrapper) Close() error {
	if s.session == nil {
		return nil
	}
	return s.session.Close()
}
