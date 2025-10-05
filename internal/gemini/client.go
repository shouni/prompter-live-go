package gemini

import (
	"context"
	"fmt"
	"log"

	"prompter-live-go/internal/types"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// Session は Gemini Live API との単一の会話セッションが満たすべきインターフェースです。
type Session interface {
	Send(ctx context.Context, data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close()
}

// Client は Gemini API との接続を管理するエクスポートされた構造体です。
type Client struct {
	baseClient *genai.Client
	modelName  string
	// システム指示をClientレベルで保持
	systemInstruction string
}

// NewClient は新しい Gemini Client インスタンスを作成します。
func NewClient(ctx context.Context, apiKey string, modelName string, systemInstruction string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is required")
	}

	// 1. genai.Client の初期化
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
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
	// 1. モデルを取得。
	model := c.baseClient.GenerativeModel(c.modelName)

	// 2. システム指示の設定ロジックは newGeminiLiveSession に移譲されたため、
	// ここではモデルの設定を省略し、単にモデルと設定、システム指示を渡します。

	// 3. 内部セッション (newGeminiLiveSession) を作成
	// 💡 修正: c.systemInstruction を第3引数として渡す
	session := newGeminiLiveSession(model, config, c.systemInstruction)

	log.Printf("New Gemini Session started for model: %s", c.modelName)

	// 4. Sessionインターフェースとして返す
	return session, nil
}

// Close は基盤となる genai.Client 接続を閉じます。
func (c *Client) Close() {
	if c.baseClient != nil {
		c.baseClient.Close()
	}
}
