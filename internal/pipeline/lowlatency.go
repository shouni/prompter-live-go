package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"prompter-live-go/internal/gemini"
	"prompter-live-go/internal/types"
	"prompter-live-go/internal/youtube"
)

// Config はパイプライン動作のための設定を保持します。
type Config = types.PipelineConfig

// LowLatencyPipeline は低遅延処理の中核を担い、入力と AI 応答のストリームを管理します。
type LowLatencyPipeline struct {
	liveClient    gemini.LiveClient
	youtubeClient *youtube.Client

	geminiConfig   types.LiveAPIConfig
	pipelineConfig Config // PipelineConfig (ポーリング間隔など)
}

// NewLowLatencyPipeline は新しいパイプラインインスタンスを作成します。
func NewLowLatencyPipeline(client gemini.LiveClient, youtubeClient *youtube.Client, geminiConfig types.LiveAPIConfig, pipelineConfig Config) *LowLatencyPipeline {
	return &LowLatencyPipeline{
		liveClient:     client,
		youtubeClient:  youtubeClient,
		geminiConfig:   geminiConfig,
		pipelineConfig: pipelineConfig, // 設定を格納
	}
}

// Run は Live API への接続を確立し、入力ストリームと出力ストリームの処理を開始します。
func (p *LowLatencyPipeline) Run(ctx context.Context) error {
	log.Println("Starting Live API connection...")

	// Gemini Live API に接続
	session, err := p.liveClient.Connect(ctx, p.geminiConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to Live API: %w", err)
	}
	defer session.Close()

	responseChan := make(chan *types.LowLatencyResponse)
	errorChan := make(chan error, 1)

	// レスポンス受信ハンドラを開始
	go p.handleReceive(session, responseChan, errorChan)

	// ライブチャットのポーリングとGeminiへの入力ハンドラを開始
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
				log.Printf("Received AI Text: %s", resp.Text)

				// AI応答をYouTubeに投稿する (非同期で実行)
				if p.youtubeClient != nil {
					go func(text string) {
						if err := p.youtubeClient.PostComment(ctx, text); err != nil {
							// 投稿エラーはログに出力し、パイプラインは継続
							log.Printf("Error posting comment to YouTube: %v", err)
						}
					}(resp.Text)
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

			// YouTube API呼び出しのリトライロジック
			maxRetries := 3
			initialDelay := 1 * time.Second

			var comments []youtube.Comment
			var err error

			for attempt := 0; attempt < maxRetries; attempt++ {
				comments, err = p.youtubeClient.FetchLiveChatMessages(ctx)

				if err == nil {
					// 成功した場合
					break
				}

				// エラーが発生した場合
				log.Printf("Error fetching live chat messages (Attempt %d/%d): %v", attempt+1, maxRetries, err)

				// 最後の試行でなければ待機 (指数バックオフ)
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
				// 最大試行回数を超えても復旧しない場合、致命的なエラーとしてパイプラインを停止
				errorChan <- fmt.Errorf("failed to fetch live chat messages after %d retries: %w", maxRetries, err)
				return
			}

			if len(comments) > 0 {
				log.Printf("Fetched %d new comments. Sending to Gemini Live API...", len(comments))

				// 各コメントをGemini Live APIセッションに送信
				for _, comment := range comments {
					inputData := types.LiveStreamData{
						// テキストデータであることを示すMIME Type
						MimeType: "text/plain",
						// メッセージの内容をバイトスライスに変換して送信
						Data: []byte(comment.Message),
					}

					if err := session.Send(inputData); err != nil {
						errorChan <- fmt.Errorf("error sending comment to Gemini Live API: %w", err)
						return // 送信エラーはパイプライン全体を停止
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
