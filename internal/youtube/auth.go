package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

// TokenFileName は保存されたトークンファイルの名前です。
const TokenFileName = "token.json"

// GetConfigPath は設定ファイルが置かれるディレクトリを取得します。
// 実際のアプリケーションでは、ユーザーのホームディレクトリなどに設定されます。
func GetConfigPath() (string, error) {
	// 簡略化のため、現在の実行ディレクトリを使用
	return ".", nil
}

// GetOAuth2Config は設定ファイル（client_secret.json）を読み込み、OAuth2 Configを返します。
func GetOAuth2Config() (*oauth2.Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	credsFile := filepath.Join(configPath, "client_secret.json")

	data, err := os.ReadFile(credsFile)
	if err != nil {
		return nil, fmt.Errorf("error reading client secret file (%s). Please ensure 'client_secret.json' is in the root directory: %w", credsFile, err)
	}

	// YouTubeのAPIスコープを設定
	// Write権限が必要なため、uploadスコープも追加
	config, err := google.ConfigFromJSON(data, youtube.YoutubeForceSslScope, youtube.YoutubeReadonlyScope, youtube.YoutubeUploadScope)
	if err != nil {
		return nil, fmt.Errorf("error parsing client secret config: %w", err)
	}

	return config, nil
}

// saveToken はトークンをファイルに保存します。
func saveToken(token *oauth2.Token) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	tokenFile := filepath.Join(configPath, TokenFileName)

	f, err := os.OpenFile(tokenFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()

	log.Printf("Saving token to %s", tokenFile)
	return json.NewEncoder(f).Encode(token)
}

// loadToken は保存されたトークンファイルを読み込みます。
func loadToken() (*oauth2.Token, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	tokenFile := filepath.Join(configPath, TokenFileName)

	f, err := os.Open(tokenFile)
	if err != nil {
		// ファイルが存在しない場合はエラーではない（新規認証が必要）
		return nil, fmt.Errorf("token file not found: %w", err)
	}
	defer f.Close()

	log.Printf("Loading token from %s", tokenFile)
	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	if err != nil {
		return nil, fmt.Errorf("error decoding token file: %w", err)
	}
	return token, nil
}

// getTokenFromWeb はウェブ認証フローを実行し、トークンを取得します。
func getTokenFromWeb(config *oauth2.Config, oauthPort int) (*oauth2.Token, error) {
	// HTTPサーバーを立ち上げるポートを設定
	serverPort := strconv.Itoa(oauthPort)
	if oauthPort == 0 {
		serverPort = "8080" // デフォルトポート
	}

	redirectURL := "http://localhost:" + serverPort
	config.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", serverPort)

	// ユーザーに認証を促す
	log.Printf("Please go to the following URL in your browser and authorize the app:")
	log.Printf("You will be redirected to: %s", redirectURL)

	// ローカルサーバーを立ち上げてリダイレクトを待ち受ける
	ch := make(chan string)

	srv := &http.Server{Addr: ":" + serverPort}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		if code != "" {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Authentication successful. You can close this window now."))
			ch <- code
		} else {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Authentication failed. No authorization code received."))
			ch <- ""
		}
	})

	// サーバーを非同期で起動
	go func() {
		log.Printf("Listening for OAuth callback on http://localhost:%d/callback", serverPort)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("Error: HTTP server failed unexpectedly: %v", err)
			// errorChan に送信するとブロッキングする可能性があるため、ログ出力のみとする
		}
	}()

	// タイムアウトを設定
	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var code string
	select {
	case code = <-ch:
		// 認証コードを受け取ったらサーバーを停止
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)

		if code == "" {
			return nil, fmt.Errorf("authorization code was empty")
		}

	case <-ctxTimeout.Done():
		// タイムアウトしたらサーバーを停止
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)

		return nil, fmt.Errorf("authentication timeout")
	}

	// 認証コードを使ってトークンを取得
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return token, nil
}

// GetToken は既存のトークンをロードまたはウェブ認証フローを通じて取得します。
func GetToken(config *oauth2.Config, oauthPort int) (*oauth2.Token, error) {
	// 1. 保存されたトークンをロード
	token, err := loadToken()
	if err == nil && token.Valid() {
		log.Println("Successfully loaded valid cached token.")
		return token, nil
	}

	// 2. トークンが無効または存在しない場合、ウェブから取得
	log.Println("Cached token expired or not found. Initiating new web authentication.")
	token, err = getTokenFromWeb(config, oauthPort)
	if err != nil {
		return nil, err
	}

	// 3. 取得した新しいトークンを保存
	if err := saveToken(token); err != nil {
		log.Printf("Warning: Failed to save new token: %v", err)
	}

	return token, nil
}

// GetOAuth2Client は認証済みの *http.Client を返す、外部パッケージ向けの公開関数です。
// internal/youtube/client.go の NewClient から呼び出されます。
func GetOAuth2Client(ctx context.Context, oauthPort int) (*http.Client, error) {
	// 1. OAuth2 設定の取得
	config, err := GetOAuth2Config()
	if err != nil {
		return nil, err
	}

	// 2. トークンの取得
	token, err := GetToken(config, oauthPort)
	if err != nil {
		return nil, err
	}

	// 3. 認証済みクライアントの作成 (トークンの自動リフレッシュ機能を含む)
	client := config.Client(ctx, token)
	return client, nil
}
