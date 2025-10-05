package gemini

import (
	"context"
	"io"

	// 公式 SDK をインポート
	genai "google.golang.org/genai"
	geminiTypes "google.golang.org/genai/types"

	// 共通の型をインポート
	"prompter-live-go/internal/types"
)

// LiveSession は確立された Live API の WebSocket セッションを抽象化します。
type LiveSession interface {
	Send(part *genai.Part) error
	Recv() (*geminiTypes.Content, error)
	Close() error
}

// LiveClient は Live API との接続を管理するインターフェースです。
type LiveClient interface {
	Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error)
}

// liveClient は LiveClient インターフェースの実装です。
type liveClient struct {
	client *genai.Client // Go SDK のクライアントインスタンス
}

// NewLiveClient は LiveClient の新しいインスタンスを作成します。
func NewLiveClient(apiKey string) (LiveClient, error) {
	// Contextは一時的にnilを使いますが、適切なContextを渡す必要があります
	client, err := genai.NewClient(context.Background(), genai.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &liveClient{client: client}, nil
}

// Connect は Gemini Live API への WebSocket 接続を確立します。
func (c *liveClient) Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error) {

	// Gemini SDKのセッション設定を構築
	sessionConfig := geminiTypes.LiveConnectConfig{
		Model:            config.Model,
		GenerationConfig: &geminiTypes.GenerationConfig{
			// ... ここで maxOutputTokens, Temperature などの設定を反映
		},
		SystemInstruction:  config.SystemInstruction,
		ResponseModalities: config.ResponseModalities,
		// ... ツール(Function Calling)の設定もここで行う
	}

	// Live API セッションを開始
	live, err := c.client.Live.Connect(ctx, sessionConfig)
	if err != nil {
		return nil, err
	}

	// SDKの LiveClient は既に LiveSession インターフェースを満たしているため、そのまま返却
	return live, nil
}

// --- LiveSession インターフェースの実装 (SDKのLiveClientを使用するため、Go SDKの型をそのまま利用) ---

// Send はデータをAIにストリーム送信します。
func (s *LiveSession) Send(data types.LiveStreamData) error {
	// types.LiveStreamData を genai.Part に変換するロジックが必要です
	part := &genai.Part{
		InlineData: &genai.Blob{
			MIMEType: data.MimeType,
			Data:     data.Data,
		},
	}
	return s.Send(part)
}

// Recv はAIからの応答ストリームを受信します。
func (s *LiveSession) Recv() (*types.LowLatencyResponse, error) {
	content, err := s.Recv()
	if err != nil {
		// EOF (ストリーム終了) や他のエラーを処理
		if err == io.EOF {
			return &types.LowLatencyResponse{Done: true}, nil
		}
		return nil, err
	}

	// SDKのContentから、コメント投稿に使うテキストを抽出
	text := content.Parts[0].Text
	return &types.LowLatencyResponse{Text: text, Done: false}, nil
}
