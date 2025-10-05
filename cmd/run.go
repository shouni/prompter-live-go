package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	// Gemini Live Client の初期化に使用
	"prompter-live-go/internal/gemini"
	// パイプラインの定義に使用
	"prompter-live-go/internal/pipeline"
	// 共通の型定義
	"prompter-live-go/internal/types"
	// YouTube クライアントの初期化に使用
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
	// ポーリング間隔用の変数を追加
	pollingInterval time.Duration
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
	// ポーリング間隔フラグを追加。デフォルト値を30秒に設定。
	runCmd.Flags().DurationVar(&pollingInterval, "polling-interval", 30*time.Second, "Polling interval for YouTube Live Chat messages (e.g., 15s, 1m).")

	// --- 必須フラグの指定 ---
	runCmd.MarkFlagRequired("api-key")
	runCmd.MarkFlagRequired("youtube-channel-id")
}

// runApplication はアプリケーションのメイン実行ロジックです。
func runApplication(cmd *cobra.Command, args []string) {
	ctx := context.Background()

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
		fmt.Printf("Pipeline execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Application finished successfully.")
}
