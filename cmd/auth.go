package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"prompter-live-go/internal/util"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// authFlags は auth コマンドのフラグを保持するための構造体です。
var authFlags struct {
	port int
}

// authCmd は OAuth2 認証フローを開始し、トークンをファイルに保存します。
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Google/YouTube OAuth2 認証フローを開始し、トークンを保存します。",
	Long:  "このコマンドを実行するとブラウザが開かれ、YouTube チャンネルへのアクセスを許可するよう求められます。",
	RunE:  authRunE,
}

func init() {
	rootCmd.AddCommand(authCmd)
	// ポート番号を指定できるように新しいフラグを追加
	authCmd.Flags().IntVar(&authFlags.port, "oauth-port", 8080, "認証コールバックサーバーがリッスンするポート番号")
}

// authRunE は auth コマンドの実行ロジックです。
func authRunE(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	callbackURL := fmt.Sprintf("http://localhost:%d/callback", authFlags.port)

	// 1. OAuth2 Config を取得
	config := util.GetOAuth2Config()
	// 実行時に設定されたポートに合わせてリダイレクトURLを上書き
	config.RedirectURL = callbackURL

	// 2. 認証 URL を生成
	state := "random-state-string"
	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))

	fmt.Printf("➡️ ブラウザで以下のURLを開き、YouTube へのアクセスを許可してください:\n%s\n", authURL)

	// 3. ユーザー認証を待つための HTTP サーバーを起動
	serverAddr := fmt.Sprintf(":%d", authFlags.port)
	serverMux := http.NewServeMux()
	server := &http.Server{
		Addr:    serverAddr,
		Handler: serverMux,
	}

	// サーバーを起動 (Go routine で実行)
	go func() {
		fmt.Printf("🌐 認証コールバックサーバー (%s) を起動しました。\n", callbackURL)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "❌ 認証サーバーエラー: %v\n", err)
			os.Exit(1)
		}
	}()

	// 4. トークンを格納するためのチャネル
	tokenChan := make(chan *oauth2.Token)
	errChan := make(chan error)

	// 5. コールバックハンドラーの設定
	serverMux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// State の検証
		if r.FormValue("state") != state {
			http.Error(w, "State mismatch", http.StatusBadRequest)
			errChan <- fmt.Errorf("state mismatch")
			return
		}

		// エラーチェック
		if r.FormValue("error") != "" {
			http.Error(w, "Authentication error", http.StatusBadRequest)
			errChan <- fmt.Errorf("authentication failed: %s", r.FormValue("error"))
			return
		}

		// 認証コードを取得
		code := r.FormValue("code")

		// トークンに交換
		token, err := config.Exchange(ctx, code)
		if err != nil {
			http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
			errChan <- fmt.Errorf("トークンの交換に失敗: %w", err)
			return
		}

		// 成功メッセージを表示し、サーバーをシャットダウン
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<h1>✅ 認証成功！</h1><p>アプリケーションに戻り、トークンが保存されたことを確認してください。</p><p>このウィンドウは閉じて構いません。</p>")

		tokenChan <- token

		// サーバーを停止
		go func() {
			time.Sleep(1 * time.Second) // クライアントへのレスポンス完了を待つ
			server.Shutdown(ctx)
		}()
	})

	// 6. ブラウザを開く
	fmt.Println("🚀 ブラウザを開いています...")
	util.OpenBrowser(authURL)

	// 7. 結果を待つ
	select {
	case token := <-tokenChan:
		// トークンをファイルに保存
		if err := util.SaveToken(util.TokenPath, token); err != nil {
			return fmt.Errorf("トークンのファイル保存に失敗: %w", err)
		}
		fmt.Printf("\n✅ 認証トークンを '%s' に保存しました。サービスを実行できます。\n", util.TokenPath)
		return nil
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("認証プロセスがキャンセルされました")
	}
}
