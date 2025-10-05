package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"prompter-live-go/internal/youtube"
)

// authCmd は YouTube 認証フローを開始するためのコマンド定義です。
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with YouTube Data API via OAuth 2.0.",
	Long: `This command initiates the OAuth 2.0 flow to authorize this application to 
read and post comments on your behalf to YouTube Live Chats.`,
	RunE: authApplication,
}

func init() {
	// rootCmd は cmd/root.go で定義され、同じパッケージ内にあるため、アクセス可能です。
	rootCmd.AddCommand(authCmd)

	// フラグは cmd/root.go のグローバル変数にバインドされます
	authCmd.Flags().IntVar(&oauthPort, "oauth-port", 8080, "Port used for OAuth2 authentication flow.")
}

// authApplication は認証フローを実行します。
func authApplication(cmd *cobra.Command, args []string) error {
	log.Println("Starting YouTube OAuth2 authentication flow...")

	// 💡 修正: 宣言されているが使用されていなかった ctx と cancel の行を削除します。
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	// defer cancel()

	// 1. OAuth2 設定を取得 (ここで YT_CLIENT_ID/SECRET が必要)
	config, err := youtube.GetOAuth2Config()
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 config. Ensure client_secret.json and environment variables are set: %w", err)
	}

	// 2. トークンを取得し、保存する
	// GetToken は、認証フローを処理し、トークンを保存するロジックを含んでいます。
	_, err = youtube.GetToken(config, oauthPort)
	if err != nil {
		return fmt.Errorf("failed to complete authentication and retrieve token: %w", err)
	}

	log.Println("✅ Authentication successful! The token has been saved.")
	return nil
}
