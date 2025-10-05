package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"prompter-live-go/internal/gemini"
	"prompter-live-go/internal/pipeline"
	"prompter-live-go/internal/types"
	"prompter-live-go/internal/youtube"

	"github.com/spf13/cobra"
)

var (
	// コマンドラインフラグを保持する変数
	apiKey            string
	youtubeChannelID  string
	modelName         string
	systemInstruction string
	pollingInterval   time.Duration
)

// rootCmd はアプリケーション全体のルートコマンドを定義します。
var rootCmd = &cobra.Command{
	Use:   "prompter_live",
	Short: "Gemini Live APIを活用し、YouTubeライブチャットへリアルタイム応答するAIエージェント",
	Long: `Prompter Live Go は、Go言語とGemini Live APIを用いて、YouTubeライブ配信者の代わりに
視聴者のコメントに対し、低遅延で自然な応答を生成し、コメント欄に投稿します。

アプリケーション起動後、すぐにチャットの監視と応答を開始します。`,
	RunE: runPipeline, // 実際の処理はこの関数に移動
}

// Execute はルートコマンドを実行します。
func Execute() {
	// ログ設定: タイムスタンプを付与
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Fatal error executing root command: %v", err)
	}
}

// init はフラグを設定します。
func init() {
	// 永続フラグ (全サブコマンドで利用可能)
	rootCmd.PersistentFlags().StringVarP(&apiKey, "gemini-api-key", "k", os.Getenv("GEMINI_API_KEY"), "Gemini APIキー。環境変数 GEMINI_API_KEY で設定可能。")
	rootCmd.PersistentFlags().StringVarP(&youtubeChannelID, "youtube-channel-id", "c", os.Getenv("YOUTUBE_CHANNEL_ID"), "監視対象のYouTubeチャンネルID (必須)。")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "使用するGeminiモデル名。")
	rootCmd.PersistentFlags().StringVarP(&systemInstruction, "system-instruction", "i", "あなたはYouTubeライブ配信者の代わりに応答するAIです。視聴者のコメントに対し、親切かつ簡潔に、ライブ配信の雰囲気に合わせたユーモアを交えて応答してください。", "Geminiに与えるシステム指示。")
	rootCmd.PersistentFlags().DurationVarP(&pollingInterval, "polling-interval", "p", 5*time.Second, "YouTubeライブチャットのポーリング間隔。")
}

// runPipeline は、ルートコマンドが実行されたときに実際にパイプラインを起動します。
func runPipeline(cmd *cobra.Command, args []string) error {
	// 1. 必須パラメータのチェック
	if apiKey == "" {
		log.Fatal("エラー: Gemini APIキー (--gemini-api-key または GEMINI_API_KEY) が設定されていません。")
	}
	if youtubeChannelID == "" {
		log.Fatal("エラー: YouTubeチャンネルID (--youtube-channel-id または YOUTUBE_CHANNEL_ID) が設定されていません。")
	}

	// 2. 設定の構築
	geminiConfig := types.LiveAPIConfig{
		Model:             modelName,
		SystemInstruction: systemInstruction,
	}

	pipelineConfig := types.PipelineConfig{
		PollingInterval:  pollingInterval,
		YouTubeChannelID: youtubeChannelID,
	}

	// 3. クライアントの初期化
	ctx := context.Background()

	// Gemini Live Client
	liveClient, err := gemini.NewLiveClient(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("Gemini Live Clientの初期化に失敗しました: %w", err)
	}

	// YouTube Client (モックまたは本物のクライアントを使用)
	// TODO: 適切なYouTubeクライアントの実装が完了次第、置き換える必要があります。
	youtubeClient := &youtube.Client{
		ChannelID: youtubeChannelID,
	}

	log.Printf("YouTube Client initialized for Channel ID: %s", youtubeChannelID)

	// 4. パイプラインの構築
	pl := pipeline.NewLowLatencyPipeline(liveClient, youtubeClient, geminiConfig, pipelineConfig)

	log.Println("--- Prompter Live Go アプリケーションを起動します ---")
	log.Printf("モデル: %s, ポーリング間隔: %v", geminiConfig.Model, pipelineConfig.PollingInterval)
	log.Printf("システム指示: %s", geminiConfig.SystemInstruction)

	// 5. パイプラインの実行
	if err := pl.Run(ctx); err != nil {
		return fmt.Errorf("パイプラインの実行中にエラーが発生しました: %w", err)
	}

	log.Println("アプリケーションが正常に終了しました。")
	return nil
}
