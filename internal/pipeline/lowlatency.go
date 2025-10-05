package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	// 必須: Gemini Live クライアント
	"prompter-live-go/internal/gemini"
	// 必須: 共通の型定義
	"prompter-live-go/internal/types"
)

// LowLatencyPipeline は低遅延処理の中核を担い、音声入力と AI 応答のストリームを管理します。
type LowLatencyPipeline struct {
	// Live API クライアントインターフェース
	liveClient gemini.LiveClient

	// セッション設定
	config types.LiveAPIConfig
}

// NewLowLatencyPipeline は新しいパイプラインインスタンスを作成します。
func NewLowLatencyPipeline(client gemini.LiveClient, config types.LiveAPIConfig) *LowLatencyPipeline {
	return &LowLatencyPipeline{
		liveClient: client,
		config:     config,
	}
}

// Run は Live API への接続を確立し、入力ストリームと出力ストリームの処理を開始します。
// 実際のオーディオ入力処理とAI応答処理を実装するメインロジックがここに入ります。
func (p *LowLatencyPipeline) Run(ctx context.Context) error {
	log.Println("Starting Live API connection...")

	// Live API への接続を確立
	session, err := p.liveClient.Connect(ctx, p.config)
	if err != nil {
		return fmt.Errorf("failed to connect to Live API: %w", err)
	}
	defer session.Close()

	// ----------------------------------------------------
	// ステップ 1: ゴルーチンで AI 応答の受信 (Recv) を開始
	// ----------------------------------------------------
	responseChan := make(chan *types.LowLatencyResponse)
	errorChan := make(chan error, 1) // エラーを受け取るチャネル

	go p.handleReceive(session, responseChan, errorChan)

	// ----------------------------------------------------
	// ステップ 2: ダミーの入力ストリーム処理 (Send)
	// ----------------------------------------------------
	// TODO: 実際のアプリケーションでは、ここでマイク入力などのリアルタイムデータを読み込む
	go p.handleSend(session, errorChan)

	// ----------------------------------------------------
	// ステップ 3: メインループで応答を待機
	// ----------------------------------------------------
	for {
		select {
		case resp := <-responseChan:
			// リアルタイム応答の処理
			if resp.Done {
				log.Println("AI response stream finished.")
				return nil // 処理完了
			}

			// TODO: ここで YouTube Live Chat API にコメントを投稿するロジックを呼び出す
			if resp.Text != "" {
				log.Printf("Received AI Text: %s", resp.Text)
			}

		case err := <-errorChan:
			// エラーが発生したらパイプラインを終了
			log.Printf("Pipeline error: %v", err)
			return err

		case <-ctx.Done():
			// Contextがキャンセルされたら終了
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
			// io.EOF 以外のエラーを報告
			errorChan <- fmt.Errorf("error receiving response: %w", err)
			return
		}

		// レスポンスをメインループに送信
		responseChan <- resp

		// ストリームが完了したらゴルーチンを終了
		if resp.Done {
			return
		}
	}
}

// handleSend は LiveSession に入力データを継続的に送信します。
func (p *LowLatencyPipeline) handleSend(session gemini.LiveSession, errorChan chan error) {
	// ダミーのデータ送信ループ
	// 実際のアプリケーションでは、マイクからのデータチャンクをここに送信します。

	// 5秒後に送信を停止すると仮定
	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			// データを送信し終えたら、ストリームに終了を通知（通常は Close を待つ）
			// Live API の Go SDK では、入力ストリームの終了は通常セッションの Close で行われます。
			log.Println("Dummy input stream finished after 5 seconds.")
			// 入力側はこれで完了し、AIからの応答を待ちます。
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
