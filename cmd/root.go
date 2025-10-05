package cmd

import (
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// 💡 修正: グローバル変数を定義し、cmd/run.go および cmd/auth.go で共有できるようにします。
var (
	// Gemini Live API 関連
	apiKey             string
	modelName          string
	systemInstruction  string
	responseModalities []string

	// YouTube Live Chat 関連
	youtubeChannelID string
	pollingInterval  time.Duration
	oauthPort        int
)

// rootCmd はアプリケーション全体のエントリポイントです。
var rootCmd = &cobra.Command{
	Use:   "prompter_live",
	Short: "AI Prompter for YouTube Live Chat using Gemini Live API",
	Long: `Prompter Live Go is a CLI tool that connects to YouTube Live Chat and uses 
Google Gemini Live API to provide low-latency, real-time responses and promotion.`,
	// RunE は、サブコマンドが指定されていない場合に実行されます（ここではヘルプ表示で十分）
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute は rootCmd を実行するエントリポイントです。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	// ここではグローバルな永続フラグを設定できますが、今回は各コマンドで個別に設定済みです。
	// 💡 修正: ここに存在していた runCmd や runApplication の重複定義を削除しました。
}
