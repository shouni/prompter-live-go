package gemini

import (
	"context"
	"fmt"

	"prompter-live-go/internal/types"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// LiveSession は Gemini Live API の対話セッションが満たすべきインターフェースです。
// これはパッケージ gemini 内のすべての Live API 実装が準拠すべき規約です。
type LiveSession interface {
	Send(ctx context.Context, data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close() // 💡 修正: live.go の実装に合わせて戻り値なしの Close() に統一
}

// NewClient は Gemini クライアントを作成し、LiveSession を初期化します。
// 外部からの呼び出しには、この関数を使用します。
func NewClient(ctx context.Context, apiKey, modelName, systemInstruction string) (LiveSession, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is empty")
	}

	// 1. Google GenAI クライアントの初期化
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create genai client: %w", err)
	}

	// 2. モデルの選択
	model := client.GenerativeModel(modelName)

	// 3. 設定の構造化
	config := types.LiveAPIConfig{
		ModelName:         modelName,
		SystemInstruction: systemInstruction,
	}

	// 4. LiveSession (ChatSessionラッパー) の作成
	// newGeminiLiveSession は internal/gemini/live.go で定義されています。
	// 💡 修正: 正しい型の引数 (model, config) を渡すことで、型不一致エラーを解消
	return newGeminiLiveSession(model, config), nil
}
