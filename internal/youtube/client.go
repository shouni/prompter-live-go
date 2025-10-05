package youtube

import (
	"context"
	"fmt"
	"log"
	"time"
	// 認証ヘルパーやSDKを想定（MOCK）
	// "prompter-live-go/internal/util"
	// "google.golang.org/api/youtube/v3"
)

// Comment は YouTube のライブチャットメッセージを表す構造体
type Comment struct {
	Author    string
	Message   string
	Timestamp time.Time
}

// Client は YouTube Live Chat API との連携を管理します。
type Client struct {
	channelID string
	// youtubeService *youtube.Service // 実際のYouTube SDKサービス
	// MOCK: 認証ポートを保持
	oauthPort int
	// MOCK: ポーリング用のカーソル
	lastPollingTime time.Time
}

// NewClient は新しい YouTube Client のインスタンスを作成します。
// 💡 修正: 認証ポート (oauthPort) を引数で受け取る
func NewClient(ctx context.Context, channelID string, oauthPort int) (*Client, error) {
	if channelID == "" {
		return nil, fmt.Errorf("youtube channel ID is empty")
	}

	// 💡 修正: util.GetOAuth2Config(0) のハードコードを避けるため、ポートを渡す
	// MOCK: 実際にはここで認証フローを実行し、APIサービスを初期化

	// 例: config, err := util.GetOAuth2Config(oauthPort)
	// 例: token, err := util.GetToken(config)
	// 例: service, err := youtube.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, token)))

	log.Printf("YouTube Client initialized for channel %s, using OAuth Port: %d", channelID, oauthPort)

	return &Client{
		channelID:       channelID,
		oauthPort:       oauthPort,
		lastPollingTime: time.Now(),
	}, nil
}

// FetchLiveChatMessages は新しいライブチャットメッセージを取得します。
func (c *Client) FetchLiveChatMessages(ctx context.Context) ([]Comment, error) {
	// MOCK: 実際には youtubeService.LiveChatMessages.List を呼び出す

	// 現在のチャットを取得するためのMOCKロジック
	// 5秒ごとに1つの新しいコメントをシミュレート
	if time.Since(c.lastPollingTime) > 20*time.Second {
		c.lastPollingTime = time.Now()
		log.Println("Fetching live chat messages [MOCK]...")

		// ダミーデータを返す
		return []Comment{
			{
				Author:    "UserA",
				Message:   "こんにちは、AIプロンプターさん！**今日のテーマ**は何ですか？",
				Timestamp: time.Now(),
			},
		}, nil
	}

	return []Comment{}, nil
}

// PostComment は指定されたテキストをライブチャットに投稿します。
func (c *Client) PostComment(ctx context.Context, text string) error {
	// MOCK: 実際には youtubeService.LiveChatMessages.Insert を呼び出す
	log.Printf("YouTube Comment Posted [MOCK]: %s", text)
	return nil
}
