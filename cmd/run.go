package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"prompter-live-go/internal/gemini"
	"prompter-live-go/internal/pipeline"
	"prompter-live-go/internal/types"
	"prompter-live-go/internal/youtube"
)

// runCmd はアプリケーションを起動するためのコマンド定義です。
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Gemini Live API chat application.",
	// RunE を使用してエラーを返し、クリーンシャットダウンフローに統合
	RunE: runApplication,
}

var (
	apiKey             string
	modelName          string
	systemInstruction  string
	responseModalities []string
	youtubeChannelID   string
	pollingInterval    time.Duration
	// 認証ポート用の変数を追加
	oauthPort int
)

func init() {
	rootCmd.AddCommand(runCmd)

	// --- Gemini Live API 関連のフラグ ---
	runCmd.Flags().StringVarP(&apiKey, "api-key", "k", os.Getenv("GEMINI_API_KEY"), "Gemini API key (or set GEMINI_API_KEY env var)")
	runCmd.Flags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "Model name to use for the live session")
	runCmd.Flags().StringVarP(&systemInstruction, "instruction", "i", "", "System instruction (prompt) for the AI personality")
	runCmd.Flags().StringSliceVarP(&responseModalities, "modalities", "r", []string{"TEXT"}, "Comma-separated list of response modalities (e.g., TEXT, AUDIO)")

	// --- YouTube 関連のフラグ ---
	runCmd.Flags().StringVarP(&youtubeChannelID, "youtube-channel-id", "c", "", "YouTube Channel ID (UCC... format) for live chat posting.")
	runCmd.Flags().DurationVar(&pollingInterval, "polling-interval", 30*time.Second, "Polling interval for YouTube Live Chat messages (e.g., 15s, 1m).")
	// 認証ポートフラグを追加
	runCmd.Flags().IntVar(&oauthPort, "oauth-port", 0, "Port used for OAuth2 authentication flow (must match 'auth' command).")

	runCmd.MarkFlagRequired("youtube-channel-id")
}

// runApplication はアプリケーションのメイン実行ロジックです。
func runApplication(cmd *cobra.Command, args []string) error {
	// APIキーの必須チェックとエラー伝播
	if apiKey == "" {
		return fmt.Errorf("gemini API key is required. Please set the GEMINI_API_KEY environment variable or use the --api-key flag")
	}

	// クリーンシャットダウンのためのコンテキスト設定
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// OSシグナルを捕捉し、コンテキストをキャンセル
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal %v. Initiating graceful shutdown...", sig)
		cancel()
	}()

	// 1. Gemini Live API 設定の構築
	geminiConfig := types.LiveAPIConfig{
		APIKey:             apiKey,
		Model:              modelName,
		SystemInstruction:  systemInstruction,
		ResponseModalities: responseModalities,
	}

	// 2. パイプライン設定の構築 (ポーリング間隔を含む)
	pipelineConfig := types.PipelineConfig{
		PollingInterval: pollingInterval,
	}

	log.Println("--- Gemini Live Prompter ---")
	log.Printf("Model: %s", geminiConfig.Model)
	log.Printf("System Instruction: %s", geminiConfig.SystemInstruction)
	log.Printf("Response Modalities: %v", geminiConfig.ResponseModalities)
	log.Printf("YouTube Channel ID: %s", youtubeChannelID)
	log.Printf("YouTube Polling Interval: %v", pipelineConfig.PollingInterval)
	log.Printf("OAuth Port: %d", oauthPort)
	log.Println("----------------------------")

	// 3. Gemini Live Client の初期化
	liveClient, err := gemini.NewLiveClient(ctx, geminiConfig.APIKey)
	if err != nil {
		return fmt.Errorf("error initializing Gemini Client: %w", err)
	}

	// 4. YouTube Client の初期化 (OAuthポートを渡す)
	youtubeClient, err := youtube.NewClient(ctx, youtubeChannelID, oauthPort)
	if err != nil {
		return fmt.Errorf("error initializing YouTube Client: %w", err)
	}

	// 5. パイプラインプロセッサの初期化
	lowLatencyProcessor := pipeline.NewLowLatencyPipeline(liveClient, youtubeClient, geminiConfig, pipelineConfig)

	// 6. パイプラインの実行
	if err := lowLatencyProcessor.Run(ctx); err != nil {
		if err == context.Canceled {
			log.Println("Application stopped gracefully.")
			return nil
		}
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	log.Println("Application finished successfully.")
	return nil
}
