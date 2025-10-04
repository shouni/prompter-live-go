package cmd

import (
	"fmt"
	"os"
	"time" // time.Duration を使用するために必要

	"github.com/spf13/cobra"
)

// runFlags は 'run' コマンド固有のフラグ値を保持するための構造体です。
// ⭐️ パッケージレベルで定義し、init() と Run() の両方からアクセス可能にする
var runFlags struct {
	channelID       string
	pollingInterval time.Duration
	promptFile      string
	dryRun          bool
}

// runCmd は "run" コマンドを定義します。
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "指定されたチャンネルのコメントを監視し、AIによる自動応答を開始します",
	Long: `このコマンドは、YouTube Data APIを使って定期的に新しいコメントをチェック（ポーリング）し、
Gemini AIで応答を生成した後、コメント欄に自動で投稿します。
	
リアルタイムなファン対応とプロモーション導線構築の中核となる機能です。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("--- Prompter Live Go: 自動応答サービス開始 ---")

		// 必須フラグのチェック
		if runFlags.channelID == "" || runFlags.promptFile == "" {
			fmt.Println("\n❌ エラー: --channel-id と --prompt-file は必須です。")
			cmd.Usage()
			os.Exit(1) // フラグが不足している場合はここで終了
		}

		fmt.Printf("✅ チャンネルID: %s\n", runFlags.channelID)
		fmt.Printf("✅ ポーリング間隔: %v\n", runFlags.pollingInterval) // runFlags のフィールドを使用 (timeパッケージを使用)
		fmt.Printf("✅ プロンプトファイル: %s\n", runFlags.promptFile)

		if runFlags.dryRun {
			fmt.Println("⚠️  ドライランモード: コメントは投稿されず、応答結果のみ表示されます。")
		} else {
			fmt.Println("🚀 ライブモード: コメントが検出され次第、自動で応答が投稿されます。")
		}

		fmt.Println("\n⚠️  [今後実装するロジック]: 認証トークンを確認し、ポーリングを開始します...")
	},
}

// init 関数で runCmd の固有フラグを定義します。
func init() {
	runCmd.Flags().StringVar(
		&runFlags.channelID, "channel-id", "",
		"監視対象の YouTube チャンネル ID (必須)",
	)
	runCmd.Flags().DurationVar(
		&runFlags.pollingInterval, "polling-interval", 30*time.Second,
		"コメントをチェックする間隔（例: 15s, 1m）",
	)
	runCmd.Flags().StringVar(
		&runFlags.promptFile, "prompt-file", "",
		"キャラクター設定と応答指示が書かれたプロンプトファイルのパス (必須)",
	)
	runCmd.Flags().BoolVar(
		&runFlags.dryRun, "dry-run", false,
		"実際のコメント投稿をスキップし、応答結果を標準出力する（テスト用）",
	)
}
