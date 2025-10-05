package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal" // シグナルパッケージを追加
	"syscall"   // シグナルパッケージを追加
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
	Run:   runApplication,
}

// Live API および YouTube 関連の設定を保持するグローバル変数
var (
	apiKey             string
	modelName          string
	systemInstruction  string
	responseModalities []string
	youtubeChannelID   string
	pollingInterval    time.Duration
)

func init() {
	rootCmd.AddCommand(runCmd)

	// --- Gemini Live API 関連のフラグ ---
	// デフォルト値として環境変数を設定。必須チェックは後段で行う。
	runCmd.Flags().StringVarP(&apiKey, "api-key", "k", os.Getenv("GEMINI_API_KEY"), "Gemini API key (or set GEMINI_API_KEY env var)")
	runCmd.Flags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "Model name to use for the live session")
	runCmd.Flags().StringVarP(&systemInstruction, "instruction", "i", "", "System instruction (prompt) for the AI personality")
	runCmd.Flags().StringSliceVarP(&responseModalities, "modalities", "r", []string{"TEXT"}, "Comma-separated list of response modalities (e.g., TEXT, AUDIO)")

	// --- YouTube 関連のフラグ ---
	runCmd.Flags().StringVarP(&youtubeChannelID, "youtube-channel-id", "c", "", "YouTube Channel ID (UCC... format) for live chat posting.")
	runCmd.Flags().DurationVar(&pollingInterval, "polling-interval", 30*time.Second, "Polling interval for YouTube Live Chat messages (e.g., 15s, 1m).")

	// --- 必須フラグの指定 ---
	// APIキーは環境変数からも取得可能にしたため、ここでは必須にしない。
	// youtube-channel-idは環境変数がないため、必須のまま。
	runCmd.MarkFlagRequired("youtube-channel-id")
}

// runApplication はアプリケーションのメイン実行ロジックです。
func runApplication(cmd *cobra.Command, args []string) {
	// クリーンシャットダウンのためのコンテキスト設定
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// OSシグナルを捕捉し、コンテキストをキャンセル
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal %v. Initiating graceful shutdown...\n", sig)
		cancel()
	}()

	// APIキーの最終チェック
	if apiKey == "" {
		fmt.Println("Error: Gemini API key is required. Please set the GEMINI_API_KEY environment variable or use the --api-key flag.")
		os.Exit(1)
	}

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

	fmt.Println("--- Gemini Live Prompter ---")
	fmt.Printf("Model: %s\n", geminiConfig.Model)
	fmt.Printf("System Instruction: %s\n", geminiConfig.SystemInstruction)
	fmt.Printf("Response Modalities: %v\n", geminiConfig.ResponseModalities)
	fmt.Printf("YouTube Channel ID: %s\n", youtubeChannelID)
	fmt.Printf("YouTube Polling Interval: %v\n", pipelineConfig.PollingInterval)
	fmt.Println("----------------------------")

	// 3. Gemini Live Client の初期化
	liveClient, err := gemini.NewLiveClient(ctx, geminiConfig.APIKey)
	if err != nil {
		fmt.Printf("Error initializing Gemini Client: %v\n", err)
		os.Exit(1)
	}

	// 4. YouTube Client の初期化
	youtubeClient, err := youtube.NewClient(ctx, youtubeChannelID)
	if err != nil {
		fmt.Printf("Error initializing YouTube Client (Check credentials/token): %v\n", err)
		os.Exit(1)
	}

	// 5. パイプラインプロセッサの初期化 (両方の設定を渡す)
	lowLatencyProcessor := pipeline.NewLowLatencyPipeline(liveClient, youtubeClient, geminiConfig, pipelineConfig)

	// 6. パイプラインの実行
	if err := lowLatencyProcessor.Run(ctx); err != nil {
		if err != context.Canceled { // Ctrl+Cによるキャンセルはエラーと見なさない
			fmt.Printf("Pipeline execution failed: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Println("Application finished successfully.")
}
