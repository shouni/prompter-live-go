package util

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	// "google.golang.org/api/youtube/v3" // REMOVED: import cycleの原因となるため削除
)

// TokenPath はOAuth2トークンを保存するファイルのパスです。
// プロジェクトルートからの相対パスを指定します。
const TokenPath = "config/token.json"

// AutoSavingTokenSource は TokenSource をラップし、
// トークンがリフレッシュされるたびにファイルに保存する役割を果たします。
type AutoSavingTokenSource struct {
	oauth2.TokenSource
	mu sync.Mutex // スレッドセーフのためのロック
}

// NewAutoSavingTokenSource は、既存の TokenSource をラップします。
func NewAutoSavingTokenSource(ts oauth2.TokenSource) oauth2.TokenSource {
	return &AutoSavingTokenSource{
		TokenSource: ts,
	}
}

// Token は新しいトークンを取得します。TokenSource がリフレッシュを実行した場合、
// 新しいトークンをファイルに保存します。
func (ts *AutoSavingTokenSource) Token() (*oauth2.Token, error) {
	token, err := ts.TokenSource.Token()
	if err != nil {
		return nil, err
	}

	// トークンが有効な場合、新しいトークンがリフレッシュされた可能性があるため、保存を試みる。
	// RefreshToken が設定されている、またはトークン自体が更新されている場合は保存される。
	if token.Valid() && token.RefreshToken != "" {
		ts.mu.Lock()
		defer ts.mu.Unlock()
		// バックグラウンドでエラーを無視して保存。これにより、毎回リフレッシュされた最新のトークンが永続化されます。
		if err := SaveToken(TokenPath, token); err != nil {
			// 致命的なエラーではないため、ログに記録するのみ
			fmt.Fprintf(os.Stderr, "⚠️ 自動トークン保存に失敗: %v\n", err)
		}
	}

	return token, nil
}

// LoadToken はローカルファイルから認証トークンを読み込みます。
func LoadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("トークンファイルのオープンに失敗: %w", err)
	}
	defer f.Close()

	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, fmt.Errorf("トークンファイルのデコードに失敗: %w", err)
	}
	return token, nil
}

// SaveToken は認証トークンをローカルファイルに保存します。
func SaveToken(path string, token *oauth2.Token) error {
	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("ディレクトリ作成に失敗: %w", err)
	}

	// 0600 (オーナーのみ読み書き可) のセキュアなパーミッションでファイルを作成/上書き
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("トークンファイルの作成/オープンに失敗: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("トークンファイルのエンコードに失敗: %w", err)
	}
	return nil
}

// LoadPromptFile はプロンプトファイルの内容を文字列として読み込みます。
func LoadPromptFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("プロンプトファイルの読み込みに失敗: %w", err)
	}
	return string(content), nil
}

// GetOAuth2Config は環境変数から認証情報 (Client ID, Secret) を読み込み、
// OAuth2設定オブジェクトを返します。
func GetOAuth2Config() *oauth2.Config {
	// YouTube Data API への書き込み権限を含むスコープ
	// NOTE: import cycle を避けるため、スコープ文字列を直接定義
	scopes := []string{
		"https://www.googleapis.com/auth/youtube.force-ssl", // HTTPSを強制
		"https://www.googleapis.com/auth/youtube.readonly",  // 読み取り
		"https://www.googleapis.com/auth/youtube",           // コメント投稿に必要
	}

	return &oauth2.Config{
		ClientID:     os.Getenv("YT_CLIENT_ID"),
		ClientSecret: os.Getenv("YT_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
		RedirectURL:  "http://localhost:8080/callback",
	}
}

// OpenBrowser は OS に応じてブラウザで URL を開きます。
func OpenBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": // macOS
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("⚠️ ブラウザの自動起動に失敗しました。以下のURLを手動で開いてください:\n%s\n", url)
	}
}
