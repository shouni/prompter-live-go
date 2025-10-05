package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
)

// TokenPath はOAuth2トークンを保存するファイルのパスです。
// プロジェクトルートからの相対パスを指定します。
const TokenPath = "config/token.json"

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
	scopes := []string{
		youtube.YoutubeForceSslScope, // OAuthをHTTPSのみに強制
		youtube.YoutubeReadonlyScope, // 読み取り
		youtube.YoutubeScope,         // フルアクセス (コメント投稿に必要)
	}

	return &oauth2.Config{
		ClientID:     os.Getenv("YT_CLIENT_ID"),
		ClientSecret: os.Getenv("YT_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		Scopes:       scopes,
		RedirectURL:  "http://localhost:8080/callback",
	}
}
