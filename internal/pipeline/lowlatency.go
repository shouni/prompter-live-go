package live_processor

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"prompter-live-go/internal/apis"
)

// Processor はライブコメントのポーリング、AI応答生成、投稿の主要なロジックを保持します。
type Processor struct {
	ytClient     *apis.YouTubeClient
	geminiClient *apis.GeminiClient
	dryRun       bool
}

// NewProcessor は新しい Processor インスタンスを作成します。
func NewProcessor(ytClient *apis.YouTubeClient, geminiClient *apis.GeminiClient, dryRun bool) *Processor {
	return &Processor{
		ytClient:     ytClient,
		geminiClient: geminiClient,
		dryRun:       dryRun,
	}
}

// ProcessNextBatch は、新しいコメントを取得し、AI応答を生成・投稿する一連の処理を実行します。
// これは cmd/run.go からポーリングループ内で呼び出されます。
func (p *Processor) ProcessNextBatch(ctx context.Context) error {
	newComments, err := p.ytClient.FetchLiveChatMessages(ctx)
	if err != nil {
		return fmt.Errorf("コメント取得エラー: %w", err)
	}

	if len(newComments) == 0 {
		slog.Info("新しいコメントはありませんでした。")
		return nil
	}

	slog.Info("新しいコメントを検出しました。", "count", len(newComments), "live_chat_id", p.ytClient.GetLiveChatID())

	// コメントごとにAI応答を生成
	for _, comment := range newComments {
		// ここで AI 自身のコメントをスキップするロジックを挿入することも可能ですが、
		// チャンネル名が頻繁に変わる可能性があるため、プロンプトでの制御を推奨します。

		slog.Info("コメントを処理中", "author", comment.Author, "message", comment.Message)

		// 1. Geminiで応答を生成
		response, err := p.geminiClient.GenerateResponse(ctx, comment.Author, comment.Message)
		if err != nil {
			slog.Error("Gemini応答生成エラー", "error", err, "author", comment.Author)
			continue
		}

		// 2. 応答を整形
		sanitizedResponse := sanitizeMessage(response)

		if sanitizedResponse == "" {
			slog.Info("AIが空の応答を生成しました (スパムまたは無効なコメントと判断)")
			continue
		}

		// 3. コメントの投稿
		if p.dryRun {
			slog.Warn("ドライラン: コメントは投稿されません。", "応答", sanitizedResponse)
		} else {
			if err := p.ytClient.PostComment(ctx, sanitizedResponse); err != nil {
				slog.Error("コメント投稿失敗", "error", err, "response", sanitizedResponse)
			}
		}
	}

	return nil
}

// sanitizeMessage は Gemini からの応答を YouTube の制約に合わせて整形します。
func sanitizeMessage(message string) string {
	// 1. マークダウンのコードブロックを削除 (例: ```json)
	re := regexp.MustCompile("(?s)```.*?```")
	message = re.ReplaceAllString(message, "")

	// 2. 過剰な改行を削除し、文字列の先頭と末尾の空白を削除
	message = strings.TrimSpace(message)
	message = strings.ReplaceAll(message, "\n", " ")
	// 3. YouTubeのコメント文字数制限 (最大500文字) を考慮
	const youtubeCommentLimit = 500
	if len(message) > youtubeCommentLimit {
		slog.Warn("メッセージが長すぎるため切り詰めます。", "original_len", len(message), "limit", youtubeCommentLimit)
		message = message[:youtubeCommentLimit]
	}

	return message
}
