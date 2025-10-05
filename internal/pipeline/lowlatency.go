package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"prompter-live-go/internal/gemini"
	"prompter-live-go/internal/types"
	// YouTube クライアントをインポート
	"prompter-live-go/internal/youtube"
)

// LowLatencyPipeline は低遅延処理の中核を担い、音声入力と AI 応答のストリームを管理します。
type LowLatencyPipeline struct {
	liveClient gemini.LiveClient
	// YouTube クライアント構造体を追加
	youtubeClient *youtube.Client

	config types.LiveAPIConfig
}

// NewLowLatencyPipeline は新しいパイプラインインスタンスを作成します。
// youtubeClient を引数に追加
func NewLowLatencyPipeline(client gemini.LiveClient, youtubeClient *youtube.Client, config types.LiveAPIConfig) *LowLatencyPipeline {
	return &LowLatencyPipeline{
		liveClient:    client,
		youtubeClient: youtubeClient, // 構造体に格納
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

	go p.handleReceive(session, responseChan, errorChan)

	// TODO: 実際のオーディオ入力は、このダミー関数を置き換える必要があります。
	go p.handleSend(session, errorChan)

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

				// 【コアロジック】AI応答をYouTubeに投稿する
				if p.youtubeClient != nil {
					// 非同期でコメント投稿を実行し、パイプラインの遅延を防ぎます。
					go func(text string) {
						// コメント投稿ロジック
						if err := p.youtubeClient.PostComment(ctx, text); err != nil {
							// 投稿エラーはパイプライン全体を停止させず、ログに出力
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

// handleSend は LiveSession に入力データを継続的に送信します。
func (p *LowLatencyPipeline) handleSend(session gemini.LiveSession, errorChan chan error) {
	// ダミーのデータ送信ループ
	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			log.Println("Dummy input stream finished after 5 seconds.")
			// 入力側を終了し、AIからの応答を待ちます。
			return

		default:
			// ダミーのオーディオデータを送信
			dummyData := types.LiveStreamData{
				MimeType: "audio/pcm;rate=16000",
				Data:     make([]byte, 8000), // 0.5秒分の16kHzモノラルPCMデータ
			}
			if err := session.Send(dummyData); err != nil {
				errorChan <- fmt.Errorf("error sending data: %w", err)
				return
			}
			// リアルタイムをシミュレートするために少し待機
			time.Sleep(500 * time.Millisecond)
		}
	}
}
