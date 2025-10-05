package apis

import (
	"context"
	"fmt"
	"os"

	"prompter-live-go/internal/util"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient は Google Gemini API と連携するためのクライアントです。
type GeminiClient struct {
	client *genai.Client
	// キャラクター設定を含むプロンプトのベース
	promptBase string
}

// NewGeminiClient は新しい GeminiClient のインスタンスを作成し、APIクライアントを初期化します。
// promptFilePath から設定ファイルを読み込み、APIキーは環境変数から取得します。
func NewGeminiClient(ctx context.Context, promptFilePath string) (*GeminiClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("環境変数 GEMINI_API_KEY が設定されていません")
	}

	// プロンプトファイルを読み込む
	promptBase, err := util.LoadPromptFile(promptFilePath)
	if err != nil {
		return nil, fmt.Errorf("プロンプトファイルの読み込みに失敗: %w", err)
	}

	// クライアントを初期化
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("Gemini クライアントの初期化に失敗: %w", err)
	}

	fmt.Println("Gemini API クライアントが正常に初期化されました。")

	return &GeminiClient{
		client:     client,
		promptBase: promptBase,
	}, nil
}

// GenerateResponse は、新しいコメントに基づいて AI の応答を生成します。
// 応答の生成には、キャラクター設定（promptBase）と、新しいユーザーコメントを使用します。
func (c *GeminiClient) GenerateResponse(ctx context.Context, newComment, author string) (string, error) {
	// 応答速度とコストのバランスが良いモデルを使用
	model := c.client.GenerativeModel("gemini-2.5-flash")

	// 厳密なキャラクター設定を守るため、通常は低い値またはデフォルトを使用します。
	model.SetTemperature(0.5)

	// システム指示(プロンプトベース)とユーザーのコメントを組み合わせたリクエストを構築
	fullPrompt := fmt.Sprintf("%s\n\n--- ユーザーのコメント ---\nユーザー名: %s\nコメント: %s\n\n--- AI応答（必ず日本語で記述）---",
		c.promptBase,
		author,
		newComment,
	)

	// API呼び出しの実行
	resp, err := model.GenerateContent(ctx, genai.Text(fullPrompt))
	if err != nil {
		return "", fmt.Errorf("Gemini API でのコンテンツ生成に失敗: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("Gemini API は応答を返しませんでした (候補なし)")
	}

	// --- ここを修正 ---
	// genai.Part を直接 string に変換するため、fmt.Sprint() を使用します
	responseText := fmt.Sprint(resp.Candidates[0].Content.Parts[0])

	return responseText, nil
}
