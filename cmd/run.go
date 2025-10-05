package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	// 必須: LiveClient の初期化に使用
	"prompter-live-go/internal/gemini"
	// 必須: パイプラインの定義に使用
	"prompter-live-go/internal/pipeline"
	// 必須: 共通の型定義
	"prompter-live-go/internal/types"
)

// runCmd はアプリケーションを起動するためのコマンド定義です。
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Gemini Live API chat application.",
	Run:   runApplication,
}

// Live API の設定を保持するグローバル変数
var (
	apiKey             string
	modelName          string
	systemInstruction  string
	responseModalities []string
)

func init() {
	rootCmd.AddCommand(runCmd)

	// コマンドライン引数の設定
	runCmd.Flags().StringVarP(&apiKey, "api-key", "k", os.Getenv("GEMINI_API_KEY"), "Gemini API key (or set GEMINI_API_KEY env var)")
	runCmd.Flags().StringVarP(&modelName, "model", "m", "gemini-2.5-flash", "Model name to use for the live session")
	runCmd.Flags().StringVarP(&systemInstruction, "instruction", "i", "", "System instruction (prompt) for the AI personality")
	runCmd.Flags().StringSliceVarP(&responseModalities, "modalities", "r", []string{"TEXT"}, "Comma-separated list of response modalities (e.g., TEXT, AUDIO)")

	// APIキーは必須
	runCmd.MarkFlagRequired("api-key")
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
		// Tools: nil, // 今回は未実装
	}

	fmt.Println("--- Gemini Live Prompter ---")
	fmt.Printf("Model: %s\n", config.Model)
	fmt.Printf("System Instruction: %s\n", config.SystemInstruction)
	fmt.Printf("Response Modalities: %v\n", config.ResponseModalities)
	fmt.Println("----------------------------")

	// 2. Gemini Live Client の初期化 (修正済み)
	// apis.NewGeminiClient の代わりに gemini.NewLiveClient を使用
	liveClient, err := gemini.NewLiveClient(ctx, config.APIKey)
	if err != nil {
		fmt.Printf("Error initializing Gemini Client: %v\n", err)
		os.Exit(1)
	}

	// 3. パイプラインプロセッサの初期化 (修正済み)
	// live_processor の代わりに pipeline.NewLowLatencyPipeline を使用
	lowLatencyProcessor := pipeline.NewLowLatencyPipeline(liveClient, config)

	// 4. パイプラインの実行
	if err := lowLatencyProcessor.Run(ctx); err != nil {
		fmt.Printf("Pipeline execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Application finished successfully.")
}
