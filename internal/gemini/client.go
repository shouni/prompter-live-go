package gemini

import (
	"context"
	"fmt"
	"log"

	"prompter-live-go/internal/types"

	"google.golang.org/genai"
)

// Session は Gemini Live API との単一の会話セッションが満たすべきインターフェースです。
// これは live.go で実装されます。
type Session interface {
	Send(ctx context.Context, data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close()
}

// Client は Gemini API との接続を管理するエクスポートされた構造体です。
type Client struct {
	baseClient        *genai.Client
	modelName         string
	systemInstruction string
}

// NewClient は新しい Gemini Client インスタンスを作成します。
func NewClient(ctx context.Context, apiKey string, modelName string, systemInstruction string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	// 1. genai.Client の初期化
	clientConfig := &genai.ClientConfig{
		APIKey: apiKey,
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	log.Printf("Gemini Client initialized with model: %s", modelName)

	// 2. Client構造体を作成
	return &Client{
		baseClient:        client,
		modelName:         modelName,
		systemInstruction: systemInstruction,
	}, nil
}

// StartSession は新しい会話セッションを開始し、その Session インターフェースを返します。
func (c *Client) StartSession(ctx context.Context, config types.LiveAPIConfig) (Session, error) {

	// 内部セッション (newGeminiLiveSession) を呼び出してセッションを作成
	session := newGeminiLiveSession(c.baseClient, c.modelName, config, c.systemInstruction)

	log.Printf("New Gemini Session started for model: %s", c.modelName)

	return session, nil
}

// Close は基盤となる genai.Client 接続を閉じます。（ここでは genai.Client に Close() がないため、ロギングのみ）
func (c *Client) Close() {
	log.Println("Gemini Client connection closed (Placeholder).")
	// 実際には、Clientのクリーンアップロジックをここに追加します。
}
