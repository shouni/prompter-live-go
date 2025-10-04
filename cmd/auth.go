package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// authCmd は "auth" コマンドを定義します。
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "YouTube APIにアクセスするためのOAuth 2.0認証フローを実行します",
	Long: `このコマンドは、Google Cloud Platformで取得したクライアントIDとシークレットを使用して、
YouTubeチャンネルへのコメント投稿権限を取得するためのOAuth 2.0認証フローを開始します。

認証後、アクセストークンとリフレッシュトークンがローカルファイルに保存されます。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("--- Prompter Live Go: OAuth 2.0 認証 ---")

		// 1. 環境変数からクライアントIDとシークレットを読み込み、チェックする (次のステップで実装)
		clientID := os.Getenv("YOUTUBE_CLIENT_ID")
		clientSecret := os.Getenv("YOUTUBE_CLIENT_SECRET")

		if clientID == "" || clientSecret == "" {
			fmt.Println("\n❌ エラー: YOUTUBE_CLIENT_ID および YOUTUBE_CLIENT_SECRET 環境変数が設定されていません。")
			fmt.Println("READMEを参照し、Google Cloud Platformで取得したクライアント情報を設定してください。")
			return
		}

		fmt.Println("✅ 環境変数を確認しました。認証フローを開始します。")

		// 2. 認証処理を内部サービスに委譲する (internal/apis/oauth_server.goと連携)
		// 例: internal.apis.StartOAuthFlow(clientID, clientSecret)
		fmt.Println("\n⚠️  [今後実装する認証ロジック]: ブラウザを開き、認証を完了してください...")

		// 認証が成功した場合のメッセージ（一時的なプレースホルダー）
		// fmt.Println("\n🎉 認証成功！アクセストークンが 'config/token.json' に保存されました。")
	},
}
