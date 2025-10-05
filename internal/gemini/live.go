package gemini

import (
	"context"
	"fmt"
	"io"
	"log"

	"prompter-live-go/internal/types"

	"google.golang.org/api/option"

	"github.com/google/generative-ai-go/genai"
)

// LiveSession はパイプラインが利用する、Gemini Liveセッションのインターフェースです。
type LiveSession interface {
	Send(data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close() error
}

// LiveClient は Gemini Live API への接続を管理するためのクライアント構造体です。
type LiveClient struct {
	client *genai.Client
}

// --- LiveSession の具体的な実装 (GenerateContentStream ベース) ---

// geminiLiveSession は LiveSession インターフェースを満たします。
type geminiLiveSession struct {
	model *genai.GenerativeModel

	// 💡 修正: ストリームからチャンクを非同期で受け取るためのチャネル
	streamChan chan *genai.GenerateContentResponse

	// 💡 修正: ストリーム終了を通知するためのチャネル
	doneChan chan error

	// 履歴をContent配列として保持
	history []*genai.Content

	// 応答の再構築用バッファ
	currentResponse *genai.Content
}

// newGeminiLiveSession は新しいセッションを作成します。
func newGeminiLiveSession(client *genai.Client, config types.LiveAPIConfig) *geminiLiveSession {
	model := client.GenerativeModel(config.Model)

	if config.SystemInstruction != "" {
		content := &genai.Content{
			Parts: []genai.Part{genai.Text(config.SystemInstruction)},
		}
		model.SystemInstruction = content
	}

	return &geminiLiveSession{
		model:           model,
		history:         []*genai.Content{},
		currentResponse: &genai.Content{Role: "model", Parts: []genai.Part{}},
	}
}

// Send はメッセージをモデルに送信し、ストリーミングを開始します。
func (s *geminiLiveSession) Send(data types.LiveStreamData) error {
	log.Printf("LiveSession: Sending input data (MimeType: %s, Data length: %d)", data.MimeType, len(data.Data))

	// 既存のチャネルがあればクローズ
	s.Close()
	s.currentResponse = &genai.Content{Role: "model", Parts: []genai.Part{}}

	var part genai.Part
	if data.MimeType == "text/plain" {
		part = genai.Text(string(data.Data))
	} else {
		part = &genai.Blob{
			MIMEType: data.MimeType,
			Data:     data.Data,
		}
	}

	userInput := &genai.Content{
		Role:  "user",
		Parts: []genai.Part{part},
	}

	// 履歴と新しい入力を組み合わせ
	contents := append(s.history, userInput) // Line 92: contents は履歴コミットに使われるため、未使用エラー解消

	// 新しいチャネルを作成し、ストリーム処理を開始
	s.streamChan = make(chan *genai.GenerateContentResponse)
	s.doneChan = make(chan error, 1) // バッファサイズ1

	// 💡 修正: GenerateContentStreamを非同期で実行し、Next()メソッドで処理
	go func() {
		stream := s.model.GenerateContentStream(context.Background(), part)

		// Next()メソッドを使ってストリームを処理（古いSDKの標準パターン）
		for {
			resp, err := stream.Next()
			if err == io.EOF {
				s.doneChan <- io.EOF
				close(s.streamChan)
				return
			}
			if err != nil {
				s.doneChan <- err
				close(s.streamChan)
				return
			}
			s.streamChan <- resp
		}
	}()

	// ユーザー入力をセッション履歴にコミット
	s.history = contents

	return nil
}

// RecvResponse はSDKからの応答を取得し、パイプラインの型に変換します。
func (s *geminiLiveSession) RecvResponse() (*types.LowLatencyResponse, error) {
	if s.streamChan == nil {
		return nil, fmt.Errorf("stream not initialized. Call Send() first")
	}

	select {
	case resp, ok := <-s.streamChan:
		if !ok {
			// ストリーム終了のチャネルが閉じている場合
			select {
			case err := <-s.doneChan:
				if err == io.EOF {
					return &types.LowLatencyResponse{Done: true}, nil
				}
				return nil, fmt.Errorf("gemini stream error: %w", err)
			default:
				return &types.LowLatencyResponse{Done: true}, nil
			}
		}

		// チャンク処理
		// 🚨 修正: genai.Part からテキストを安全に抽出
		text := "" // Line 145: 使用されるため、未使用エラー解消
		if resp.Candidates != nil && len(resp.Candidates) > 0 {
			if len(resp.Candidates[0].Content.Parts) > 0 {

				part := resp.Candidates[0].Content.Parts[0]

				// genai.Part を genai.Text に型キャストしてテキストを抽出
				if textChunk, ok := part.(genai.Text); ok { // Line 148: .Text メソッドがないエラーを解決
					text = string(textChunk)

					// 履歴用の応答バッファを更新
					if len(s.currentResponse.Parts) == 0 {
						s.currentResponse.Parts = append(s.currentResponse.Parts, genai.Text(text))
					} else {
						existingText := s.currentResponse.Parts[0].(genai.Text)
						s.currentResponse.Parts[0] = existingText + genai.Text(text)
					}

					return &types.LowLatencyResponse{
						Text: text,
						Done: false,
					}, nil
				}
			}
		}
		return &types.LowLatencyResponse{Text: "", Done: false}, nil

	case err := <-s.doneChan:
		if err == io.EOF {
			return &types.LowLatencyResponse{Done: true}, nil
		}
		return nil, fmt.Errorf("gemini stream error: %w", err)
	}
}

// Close はセッションを閉じます。
func (s *geminiLiveSession) Close() error {
	log.Println("Closing Gemini Live session.")
	if s.doneChan != nil {
		// 応答全体が完了したら、モデルの応答を履歴に追加
		s.history = append(s.history, s.currentResponse)
		s.doneChan = nil
	}
	// ストリームのチャネルは go routine 側でクローズされるため、ここで nil にするのみ
	s.streamChan = nil
	return nil
}

// --- LiveClientの実装 ---

// NewLiveClient は LiveClient の新しいインスタンスを作成します。
func NewLiveClient(ctx context.Context, apiKey string) (*LiveClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is empty")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("error creating gemini client: %w", err)
	}

	log.Println("Gemini Live Client initialized successfully.")
	return &LiveClient{client: client}, nil
}

// Connect は Gemini Live API への新しいセッションを確立します。
func (c *LiveClient) Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error) {
	if c.client == nil {
		return nil, fmt.Errorf("cannot connect: gemini client is not initialized")
	}

	log.Printf("Connecting to Live API with model: %s, Instruction: %s...", config.Model, config.SystemInstruction)

	return newGeminiLiveSession(c.client, config), nil
}
