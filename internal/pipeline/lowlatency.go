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

// LowLatencyPipeline は低遅延処理の中核を担い、音声入力と AI 応答のストリームを管理します。
type LowLatencyPipeline struct {
	liveClient    gemini.LiveClient
	youtubeClient *youtube.Client

	config types.LiveAPIConfig
}

// NewLowLatencyPipeline は新しいパイプラインインスタンスを作成します。
func NewLowLatencyPipeline(client gemini.LiveClient, youtubeClient *youtube.Client, config types.LiveAPIConfig) *LowLatencyPipeline {
	return &LowLatencyPipeline{
		liveClient:    client,
		youtubeClient: youtubeClient,
		config:        config,
	}
}

// Run は Live API への接続を確立し、入力ストリームと出力ストリームの処理を開始します。
func (p *LowLatencyPipeline) Run(ctx context.Context) error {
	log.Println("Starting Live API connection...")

	session, err := p.liveClient.Connect(ctx, p.config)
	if err != nil {
		return fmt.Errorf("failed to connect to Live API: %w", err)
	}
	defer session.Close()

	responseChan := make(chan *types.LowLatencyResponse)
	errorChan := make(chan error, 1)

	// 1. レスポンス受信ハンドラを開始
	go p.handleReceive(session, responseChan, errorChan)

	// 2. ライブチャットのポーリングとGeminiへの入力ハンドラを開始 (ダミーオーディオを置き換え)
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

				// AI応答をYouTubeに投稿する
				if p.youtubeClient != nil {
					// 非同期でコメント投稿を実行
					go func(text string) {
						if err := p.youtubeClient.PostComment(ctx, text); err != nil {
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
	// YouTube Live Chat APIの推奨ポーリング間隔に合わせて設定
	// Note: この間隔がクォータ消費に直結します。
	const pollingInterval = 5 * time.Second
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	log.Printf("Starting YouTube Live Chat polling every %s...", pollingInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Input handler shutting down.")
			return
		case <-ticker.C:
			// ライブチャットメッセージを取得
			comments, err := p.youtubeClient.FetchLiveChatMessages(ctx)
			if err != nil {
				// ライブ配信終了などのエラーは致命的ではないため、ログに記録してポーリングを継続
				log.Printf("Error fetching live chat messages: %v", err)
				continue
			}

			if len(comments) > 0 {
				log.Printf("Fetched %d new comments. Sending to Gemini Live API...", len(comments))

				// 各コメントをGemini Live APIセッションに送信
				for _, comment := range comments {
					// LiveStreamDataを使用してテキストとして送信します。
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
					// ログに誰のコメントを送信したかを含める
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
