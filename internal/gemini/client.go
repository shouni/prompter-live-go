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

	// 2. システム指示を設定
	// 💡 修正: genai.WithSystemInstructionが未定義のため、モデルの再生成ロジックを削除し、
	// ビルドエラーを回避します。システム指示の適用は、newGeminiLiveSession (internal/gemini/live.go)
	// の内部で処理される必要があります。

	// システム指示の適用が有効になるまで、ログに出力
	if c.systemInstruction != "" {
		log.Printf("Warning: System instruction ('%s') is currently not applied due to SDK build constraints.", c.systemInstruction)
		// ⚠️ 今後、live.goを修正して systemInstruction を適用する必要があります。
	}

	// 3. 内部セッション (newGeminiLiveSession) を作成
	// newGeminiLiveSessionのシグネチャに合わせ、modelとconfigのみを渡す
	session := newGeminiLiveSession(model, config)

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
