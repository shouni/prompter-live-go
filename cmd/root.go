package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd はアプリケーション全体のコマンドのルートです。
var rootCmd = &cobra.Command{
	Use:   "prompter_live",
	Short: "YouTube Live ChatのコメントにAIで自動応答するCLIツール",
	Long: `Prompter Live Go は、YouTube Live Chat APIとGoogle Gemini APIを連携させ、
設定したキャラクターに応じてコメントに自動応答するサービスを構築します。`,
}

// Execute はルートコマンドを実行します。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "エラーが発生しました: %v\n", err)
		os.Exit(1)
	}
}
