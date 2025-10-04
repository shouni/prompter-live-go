package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"prompter-live-go/internal/apis"
	"prompter-live-go/internal/util"
)

// トークン保存先ファイルパス
const tokenFilePath = "config/token.json"

// ローカルサーバーが待ち受けるポート
const oauthPort = "8080"

// authCmd は "auth" コマンドを定義します。
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "YouTube APIにアクセスするためのOAuth 2.0認証フローを実行します",
	Long: `このコマンドは、Google Cloud Platformで取得したクライアントIDとシークレットを使用して、
YouTubeチャンネルへのコメント投稿権限を取得するためのOAuth 2.0認証フローを開始します。

認証後、アクセストークンとリフレッシュトークンがローカルファイルに保存されます。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("--- Prompter Live Go: OAuth 2.0 認証 ---")

		// 1. 環境変数からクライアントIDとシークレットを読み込み、チェック
		clientID := os.Getenv("YOUTUBE_CLIENT_ID")
		clientSecret := os.Getenv("YOUTUBE_CLIENT_SECRET")

		if clientID == "" || clientSecret == "" {
			return fmt.Errorf("\n❌ エラー: YOUTUBE_CLIENT_ID または YOUTUBE_CLIENT_SECRET 環境変数が設定されていません。\nREADMEを参照し、Google Cloud Platformでクライアント情報を設定してください。")
		}

		// 2. OAuth2 設定の構成
		config := &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			// YouTubeのコメント投稿に必要なスコープ
			Scopes: []string{"https://www.googleapis.com/auth/youtube.force-ssl"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
			// ローカルサーバーのコールバックURLを指定
			RedirectURL: "http://localhost:" + oauthPort + "/oauth/callback",
		}

		// 3. ローカルサーバーを起動し、認証コードを待ち受ける
		server := apis.NewOAuthServer(oauthPort)
		server.Start()

		// 4. ユーザーを認証URLに誘導
		authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
		fmt.Printf("\n🚀 以下のURLをブラウザで開いて、YouTubeチャンネルに権限を与えてください:\n%s\n", authURL)

		// 5. チャネルから認証コードを受け取るまで待機
		// タイムアウトを設定して、サーバーが永遠に待ち続けないようにする
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		var code string
		select {
		case code = <-server.CodeChan:
			// コードを受信
		case <-ctx.Done():
			// タイムアウト
			return fmt.Errorf("\n❌ 認証タイムアウト: 5分以内に認証コードが受信されませんでした。")
		}

		// 6. 認証コードをアクセストークンに交換
		fmt.Println("\n✅ 認証コードを受信。アクセストークンを取得中...")
		token, err := config.Exchange(context.Background(), code)
		if err != nil {
			return fmt.Errorf("\n❌ トークン交換に失敗: %w", err)
		}

		// 7. トークンをファイルに保存
		if err := util.SaveToken(tokenFilePath, token); err != nil {
			return fmt.Errorf("\n❌ トークンの保存に失敗: %w", err)
		}

		fmt.Printf("\n🎉 認証成功！アクセストークンとリフレッシュトークンが '%s' に保存されました。\n", tokenFilePath)
		return nil
	},
}
