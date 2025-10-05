package cmd

import (
	"context"
	"fmt"
	"os"

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
	youtubeChannelID   string // YouTube チャンネルID (必須)
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

	// --- 必須フラグの指定 ---
	runCmd.MarkFlagRequired("api-key")
	runCmd.MarkFlagRequired("youtube-channel-id")
}

// runApplication はアプリケーションのメイン実行ロジックです。
func runApplication(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. API 設定の構築 (types.LiveAPIConfig)
	config := types.LiveAPIConfig{
		APIKey:             apiKey,
		Model:              modelName,
		SystemInstruction:  systemInstruction,
		ResponseModalities: responseModalities,
	}

	fmt.Println("--- Gemini Live Prompter ---")
	fmt.Printf("Model: %s\n", config.Model)
	fmt.Printf("System Instruction: %s\n", config.SystemInstruction)
	fmt.Printf("Response Modalities: %v\n", config.ResponseModalities)
	fmt.Printf("YouTube Channel ID: %s\n", youtubeChannelID)
	fmt.Println("----------------------------")

	// 2. Gemini Live Client の初期化
	liveClient, err := gemini.NewLiveClient(ctx, config.APIKey)
	if err != nil {
		fmt.Printf("Error initializing Gemini Client: %v\n", err)
		os.Exit(1)
	}

	// 3. YouTube Client の初期化
	youtubeClient, err := youtube.NewClient(ctx, youtubeChannelID)
	if err != nil {
		fmt.Printf("Error initializing YouTube Client (Check credentials/token): %v\n", err)
		os.Exit(1)
	}

	// 4. パイプラインプロセッサの初期化 (両クライアントを渡す)
	lowLatencyProcessor := pipeline.NewLowLatencyPipeline(liveClient, youtubeClient, config)

	// 5. パイプラインの実行
	if err := lowLatencyProcessor.Run(ctx); err != nil {
		fmt.Printf("Pipeline execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Application finished successfully.")
}
