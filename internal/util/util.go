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
)

// TokenPath は認証トークンを保存するファイルのパスです。
var TokenPath = filepath.Join("config", "token.json")

// LoadToken は認証トークンをファイルから読み込みます。
func LoadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("トークンファイルが見つかりません: %w", err)
	}
	defer f.Close()
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	if err != nil {
		return nil, fmt.Errorf("トークンのデコードに失敗: %w", err)
	}
	return t, nil
}

// SaveToken は認証トークンをファイルに保存します。
func SaveToken(path string, token *oauth2.Token) error {
	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("設定ディレクトリの作成に失敗: %w", err)
		}
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("トークンファイルのオープンに失敗: %w", err)
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(token)
}

// LoadPromptFile はプロンプトファイルを読み込み、文字列として返します。
func LoadPromptFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("プロンプトファイルの読み込みに失敗: %w", err)
	}
	return string(b), nil
}

// GetOAuth2Config は環境変数から認証情報 (Client ID, Secret) を読み込み、
// OAuth2設定オブジェクトを返します。
// authPort: OAuth認証サーバーがリッスンするポート番号。RedirectURLの生成に使用されます。
func GetOAuth2Config(authPort int) *oauth2.Config {
	scopes := []string{
		// 必要なスコープを定義
		"https://www.googleapis.com/auth/youtube.force-ssl",
	}

	return &oauth2.Config{
		ClientID:     os.Getenv("YT_CLIENT_ID"),
		ClientSecret: os.Getenv("YT_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
		// ポート番号を動的に使用して RedirectURL を構築
		RedirectURL: fmt.Sprintf("http://localhost:%d/callback", authPort),
	}
}

// OpenBrowser は指定されたURLをシステムのデフォルトブラウザで開きます。
func OpenBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("cmd", "/c", "start", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("サポートされていないプラットフォームです")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ ブラウザの起動に失敗: %v\n", err)
		fmt.Println("手動で以下のURLを開いてください:", url)
	}
}

// --- カスタム TokenSource の実装 (トークン自動保存機能) ---

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
		// バックグラウンドでエラーを無視して保存。
		if err := SaveToken(TokenPath, token); err != nil {
			// 致命的なエラーではないため、ログに記録するのみ
			fmt.Fprintf(os.Stderr, "⚠️ 自動トークン保存に失敗: %v\n", err)
		}
	}

	return token, nil
}
