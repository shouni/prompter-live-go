package pipeline

import (
	"context"
	"errors" // 💡 修正: errors パッケージを追加
	"fmt"
	"io" // 💡 修正: io パッケージを追加
	"log"
	"time"

	"prompter-live-go/internal/gemini"
	"prompter-live-go/internal/types"
	"prompter-live-go/internal/youtube"
)

// LowLatencyPipeline はライブチャットのリアルタイム処理を管理します。
type LowLatencyPipeline struct {
	geminiClient   *gemini.Client
	youtubeClient  *youtube.Client
	geminiConfig   types.LiveAPIConfig
	pipelineConfig types.PipelineConfig
	session        gemini.Session
}

// NewLowLatencyPipeline は新しいパイプラインインスタンスを作成します。
func NewLowLatencyPipeline(
	geminiClient *gemini.Client,
	youtubeClient *youtube.Client,
	geminiConfig types.LiveAPIConfig,
	pipelineConfig types.PipelineConfig,
) *LowLatencyPipeline {
	return &LowLatencyPipeline{
		geminiClient:   geminiClient,
		youtubeClient:  youtubeClient,
		geminiConfig:   geminiConfig,
		pipelineConfig: pipelineConfig,
	}
}

// Run はメインのパイプライン処理を開始します。
func (p *LowLatencyPipeline) Run(ctx context.Context) error {
	log.Println("Pipeline started.")

	// 1. Geminiセッションの初期化
	// Live Client は内部でセッションを開始します
	session, err := p.geminiClient.StartSession(ctx, p.geminiConfig)
	if err != nil {
		return fmt.Errorf("failed to start Gemini session: %w", err)
	}
	p.session = session
	defer p.session.Close()

	// 2. メインループの実行
	return p.runLoop(ctx)
}

// runLoop は定期的なポーリングとAI応答処理を行うメインのループです。
func (p *LowLatencyPipeline) runLoop(ctx context.Context) error {
	// YouTube Live Chat API から推奨されるポーリング間隔を初期値として設定
	nextPollDelay := p.pipelineConfig.PollingInterval

	for {
		select {
		case <-ctx.Done():
			// アプリケーション終了シグナルを受け取る
			log.Println("Pipeline context cancelled. Shutting down.")
			return ctx.Err()
		case <-time.After(nextPollDelay):
			// ポーリング間隔が経過したら実行

			// 1. YouTube から新しいコメントを取得
			comments, pollingInterval, err := p.youtubeClient.FetchLiveChatMessages(ctx)

			// 2. エラー処理
			if err != nil {
				if errors.Is(err, youtube.ErrLiveChatEnded) {
					log.Println("Live chat ended. Waiting 30s before trying to find a new chat.")
					// ライブチャットが終了した場合は、次の再試行まで長めに待つ
					nextPollDelay = 30 * time.Second
					continue
				}
				log.Printf("Error fetching live chat messages: %v. Retrying in %v.", err, nextPollDelay)
				// その他のエラーの場合は、次のポーリング間隔まで待って再試行
				continue
			}

			// 💡 修正: pollingInterval を使用して次の待機時間を動的に設定
			// APIが推奨するポーリング間隔に更新
			if pollingInterval > 0 {
				nextPollDelay = pollingInterval
			} else {
				// 0sが返された場合は、デフォルトに戻すか、前回値を維持
				log.Println("API returned 0s polling interval. Using default.")
				// nextPollDelay は変更しない (前回値を維持)
			}

			// 3. 取得したコメントを AI に送信し、応答処理を開始
			for _, comment := range comments {
				// AIが自分自身に応答しないように、AuthorIDでフィルタリングが必要だが、
				// youtube.Client がこのロジックを持っていないため、一旦すべて送信する。
				log.Printf("New Comment received from %s: %s", comment.Author, comment.Message)

				// AIにコメントを送信 (非同期で応答ストリームを開始する)
				data := types.LiveStreamData{
					Text: fmt.Sprintf("%s says: %s", comment.Author, comment.Message),
					// Modalitiesなどの追加情報をここに追加可能
				}
				if err := p.session.Send(ctx, data); err != nil {
					log.Printf("Error sending message to Gemini: %v", err)
					continue
				}

				// 4. AI応答の受信と YouTube への投稿（ブロック）
				p.handleAIResponse(ctx)
			}
		}
	}
}

// handleAIResponse はAIからの応答を受け取り、YouTubeに投稿します。
func (p *LowLatencyPipeline) handleAIResponse(ctx context.Context) {
	// 💡 RecvResponse は完全な応答が来るまで待機し、一度だけ返します。
	resp, err := p.session.RecvResponse()
	if err != nil {
		if errors.Is(err, io.EOF) {
			// ストリーム完了（正常終了）
			return
		}
		log.Printf("Error receiving Gemini response: %v", err)
		return
	}

	// 応答テキストが空でなければ投稿
	if resp.ResponseText != "" {
		log.Printf("AI Response: %s", resp.ResponseText)

		// YouTube にコメントを投稿
		if err := p.youtubeClient.PostComment(ctx, resp.ResponseText); err != nil {
			log.Printf("Error posting comment to YouTube: %v", err)
		}
	}

	// Done: true であれば、この応答でストリームが終了したことを意味します（RecvResponseのロジックで保証されています）
}
