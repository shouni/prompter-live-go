package cmd

import (
	"fmt"

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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := cmd.Flags().GetString("api-key")
		if err != nil {
			return fmt.Errorf("failed to get api-key flag: %w", err)
		}
		if apiKey == "" {
			return fmt.Errorf("Gemini API key is required. Please set it via --api-key flag or GEMINI_API_KEY environment variable.")
		}
		// 他のPersistentPreRunE処理（もしあれば）
		return nil
	},
}
