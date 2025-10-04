package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// RootCmd は Prompter Live Go のエントリポイントとなるコマンドです。
var RootCmd = &cobra.Command{
	Use:   "prompter_live",
	Short: "Gemini AIを活用し、YouTubeコメントに自動応答するCLIツール",
	Long: `Prompter Live Go は、Google Gemini AIとYouTube Data APIを連携させ、
設定されたキャラクター設定に基づいて、YouTubeコメントにリアルタイムで自動応答します。

利用可能なサブコマンド:
  auth : YouTube APIアクセス用のOAuth認証フローを実行します。(初回必須)
  run  : 指定したチャンネルのコメントをポーリングし、AI自動応答を開始します。`,
}

// Execute はルートコマンドの実行を担います。main.goから呼び出されます。
func Execute() error {
	// ここでサブコマンドをルートに追加
	// Goのパッケージシステムにより、authCmdとrunCmdは自動的にこのパッケージ内で利用可能です
	RootCmd.AddCommand(authCmd)
	RootCmd.AddCommand(runCmd)

	// ... 省略 ...

	return RootCmd.Execute()
}

// 永続的なフラグ（すべてのサブコマンドで利用可能）を定義します。
func init() {
	// Gemini API Key の定義 (環境変数 GEMINI_API_KEY を優先)
	RootCmd.PersistentFlags().StringP("api-key", "k", os.Getenv("GEMINI_API_KEY"), "Gemini API key. (Env: GEMINI_API_KEY)")

	// 使用する Gemini モデルの定義
	RootCmd.PersistentFlags().String("model", "gemini-2.5-flash", "Gemini model name to use (e.g., gemini-2.5-flash)")
}
