package pipeline

import (
	"context"
	"fmt"
	"log"
	"regexp"  // 正規表現を使用
	"strings" // 文字列処理に使用
	"time"
	"unicode/utf8"

	"prompter-live-go/internal/gemini"
	"prompter-live-go/internal/types"
	"prompter-live-go/internal/youtube"
)

// Config はパイプライン動作のための設定を保持します。
type Config = types.PipelineConfig

// YouTubeのライブチャットコメントの最大文字数 (500文字)
const youtubeMaxCommentLength = 500

// LowLatencyPipeline は低遅延処理の中核を担い、入力と AI 応答のストリームを管理します。
type LowLatencyPipeline struct {
	liveClient    gemini.LiveClient
	youtubeClient *youtube.Client

	geminiConfig   types.LiveAPIConfig
	pipelineConfig Config
}

// NewLowLatencyPipeline は新しいパイプラインインスタンスを作成します。
func NewLowLatencyPipeline(client gemini.LiveClient, youtubeClient *youtube.Client, geminiConfig types.LiveAPIConfig, pipelineConfig Config) *LowLatencyPipeline {
	return &LowLatencyPipeline{
		liveClient:     client,
		youtubeClient:  youtubeClient,
		geminiConfig:   geminiConfig,
		pipelineConfig: pipelineConfig,
	}
}

// Run は Live API への接続を確立し、入力ストリームと出力ストリームの処理を開始します。
func (p *LowLatencyPipeline) Run(ctx context.Context) error {
	log.Println("Starting Live API connection...")

	session, err := p.liveClient.Connect(ctx, p.geminiConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to Live API: %w", err)
	}
	defer session.Close()

	responseChan := make(chan *types.LowLatencyResponse)
	errorChan := make(chan error, 1)

	go p.handleReceive(session, responseChan, errorChan)

	go p.handleLiveChatPollingAndInput(ctx, session, errorChan)

	for {
		select {
		case resp := <-responseChan:
			// リアルタイム応答の処理
			if resp.Done {
				log.Println("AI response stream finished.")
				return nil
			}

			if resp.Text != "" {
				// 応答テキストをYouTubeの文字数制限に合わせてサニタイズ
				safeText := sanitizeMessage(resp.Text)
				log.Printf("Received AI Text (Sanitized to %d chars): %s", utf8.RuneCountInString(safeText), safeText)

				// AI応答をYouTubeに投稿する (非同期で実行)
				if p.youtubeClient != nil {
					go func(text string) {
						if err := p.youtubeClient.PostComment(ctx, text); err != nil {
							log.Printf("Error posting comment to YouTube: %v", err)
						}
					}(safeText)
				}
			}

		case err := <-errorChan:
			log.Printf("Pipeline error: %v", err)
			return err

		case <-ctx.Done():
			log.Println("Pipeline shutting down due to context cancellation.")
			return ctx.Err()
		}
	}
}

// sanitizeMessage はメッセージをYouTubeコメントとして最適な形式に整形します。
func sanitizeMessage(message string) string {
	// 1. Markdown記号の除去
	// コードブロック (```...```) やインラインコード (`...`) を除去
	reCodeBlock := regexp.MustCompile("```[^`]*```")
	message = reCodeBlock.ReplaceAllString(message, "")
	reInlineCode := regexp.MustCompile("`([^`]+)`")
	message = reInlineCode.ReplaceAllString(message, "$1") // バッククォートのみ除去し、中身は残す

	// 強調記号 (**text**, *text*, __text__, _text_) の除去
	reEmphasis := regexp.MustCompile(`(\*\*|__)(.*?)\1`)
	message = reEmphasis.ReplaceAllString(message, "$2")
	reSingleEmphasis := regexp.MustCompile(`(\*|_)(.*?)\1`)
	message = reSingleEmphasis.ReplaceAllString(message, "$2")

	// ヘッダー (#) や引用 (>) の記号を除去
	reHeaders := regexp.MustCompile(`^[#]+[\s]?`)
	message = reHeaders.ReplaceAllString(message, "")
	message = strings.ReplaceAll(message, ">", "")

	// リスト記号 (*, -, 数字.) の除去（行頭のみ）
	reList := regexp.MustCompile(`^[\s]*[*-] `)
	message = reList.ReplaceAllString(message, "")
	reNumberedList := regexp.MustCompile(`^[\s]*\d+\. `)
	message = reNumberedList.ReplaceAllString(message, "")

	// 2. 連続する改行を統一
	reMultipleNewlines := regexp.MustCompile(`\n{2,}`)
	message = reMultipleNewlines.ReplaceAllString(message, "\n")

	// 3. 前後の余分な空白・改行を除去
	message = strings.TrimSpace(message)

	// 4. 文字数制限による切り詰め (前回実装したロジック)
	if utf8.RuneCountInString(message) <= youtubeMaxCommentLength {
		return message
	}

	runes := []rune(message)
	trimmedRunes := runes[:youtubeMaxCommentLength]

	suffix := "..."
	if utf8.RuneCountInString(string(trimmedRunes))+utf8.RuneCountInString(suffix) > youtubeMaxCommentLength {
		trimmedRunes = runes[:youtubeMaxCommentLength-utf8.RuneCountInString(suffix)]
	}

	log.Printf("Warning: AI response exceeds %d characters. Trimming message.", youtubeMaxCommentLength)
	return string(trimmedRunes) + suffix
}

// handleLiveChatPollingAndInput は YouTube Live Chat を定期的にポーリングし、新しいコメントを
// Gemini Live API セッションにテキストデータとして送信します。
func (p *LowLatencyPipeline) handleLiveChatPollingAndInput(ctx context.Context, session gemini.LiveSession, errorChan chan error) {
	// 設定されたポーリング間隔を使用
	pollingInterval := p.pipelineConfig.PollingInterval
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	log.Printf("Starting YouTube Live Chat polling every %s...", pollingInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Input handler shutting down.")
			return
		case <-ticker.C:

			// YouTube API呼び出しのリトライロジック (指数バックオフ)
			maxRetries := 3
			initialDelay := 1 * time.Second

			var comments []youtube.Comment
			var err error

			for attempt := 0; attempt < maxRetries; attempt++ {
				comments, err = p.youtubeClient.FetchLiveChatMessages(ctx)

				if err == nil {
					break
				}

				log.Printf("Error fetching live chat messages (Attempt %d/%d): %v", attempt+1, maxRetries, err)

				if attempt < maxRetries-1 {
					delay := initialDelay * time.Duration(1<<attempt)
					log.Printf("Retrying in %v...", delay)
					select {
					case <-ctx.Done():
						return
					case <-time.After(delay):
						// 再試行
					}
				}
			}

			if err != nil {
				errorChan <- fmt.Errorf("failed to fetch live chat messages after %d retries: %w", maxRetries, err)
				return
			}

			if len(comments) > 0 {
				log.Printf("Fetched %d new comments. Sending to Gemini Live API...", len(comments))

				for _, comment := range comments {
					inputData := types.LiveStreamData{
						MimeType: "text/plain",
						Data:     []byte(comment.Message),
					}

					if err := session.Send(inputData); err != nil {
						errorChan <- fmt.Errorf("error sending comment to Gemini Live API: %w", err)
						return
					}
					log.Printf("Sent to AI: '%s' (by %s)", comment.Message, comment.Author)
				}
			}
		}
	}
}

// handleReceive は LiveSession からの応答を継続的に受け取ります。
func (p *LowLatencyPipeline) handleReceive(session gemini.LiveSession, responseChan chan *types.LowLatencyResponse, errorChan chan error) {
	for {
		resp, err := session.RecvResponse()
		if err != nil {
			errorChan <- fmt.Errorf("error receiving response: %w", err)
			return
		}

		responseChan <- resp

		if resp.Done {
			return
		}
	}
}
