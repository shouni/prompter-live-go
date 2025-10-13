package gemini

import (
	"context"
	"fmt"
	"log"

	"prompter-live-go/internal/types"

	"google.golang.org/genai"
)

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
	// c.baseClient は *genai.Client 型
	session := newGeminiLiveSession(c.baseClient, c.modelName, config, c.systemInstruction)

	log.Printf("New Gemini Session started for model: %s", c.modelName)

	return session, nil
}

// Close は基盤となる genai.Client 接続を閉じます。
// （現在の genai.Client には明示的な Close() メソッドがないため、ロギングのみですが、
// 将来的なSDK更新によりリソースクリーンアップが必要になる可能性に注意してください。）
func (c *Client) Close() {
	log.Println("Gemini Client connection closed (Placeholder for potential future cleanup).")
	// 実際には、Clientのクリーンアップロジックをここに追加します。
	// 例: if c.baseClient != nil { /* SDKがClose()を提供した場合の呼び出し */ }
}
