package youtube

import (
	"context"
	"fmt"
	"log"
	"time"
	// 💡 TODO: ここに実際の YouTube SDK や OAuth2 ライブラリのインポートが必要です
	// 例: "google.golang.org/api/youtube/v3"
	// 例: "golang.org/x/oauth2"
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
	// TODO: ここに youtube.Service などの実際の YouTube SDK サービスインスタンスを保持します
	// 例: youtubeService *youtube.Service

	// 認証ポートは初期化時のみ使用
	oauthPort int

	// MOCK: ポーリング用のカーソルを維持 (MOCK用)
	lastPollingTime time.Time
}

// NewClient は新しい YouTube Client のインスタンスを作成します。
func NewClient(ctx context.Context, channelID string, oauthPort int) (*Client, error) {
	if channelID == "" {
		return nil, fmt.Errorf("youtube channel ID is empty")
	}

	// 💡 TODO: ここに実際の OAuth 認証フローを実装
	// 1. 認証設定を取得 (oauthPort を利用)
	// 2. トークンを取得またはリフレッシュ
	// 3. YouTube API サービス (youtube.Service) を初期化し、*Client に保持

	log.Printf("YouTube Client initialized for channel %s, using OAuth Port: %d", channelID, oauthPort)

	return &Client{
		channelID:       channelID,
		oauthPort:       oauthPort,
		lastPollingTime: time.Now(),
	}, nil
}

// FetchLiveChatMessages は新しいライブチャットメッセージを取得します。
func (c *Client) FetchLiveChatMessages(ctx context.Context) ([]Comment, error) {
	// 💡 TODO: ここに実際の SDK 呼び出しロジックを実装
	// 1. c.youtubeService.LiveChatMessages.List を呼び出し、liveChatId を指定
	// 2. 応答から新しいコメントを抽出し、Comment スライスに変換

	// --- MOCK: 実際のコメント取得をシミュレート ---
	if time.Since(c.lastPollingTime) > 20*time.Second {
		c.lastPollingTime = time.Now()
		log.Println("Fetching live chat messages [MOCK]...")

		// ダミーデータを返す
		return []Comment{
			{
				Author:    "UserB",
				Message:   "AI さん、今日は天気が良いですね！何か面白い話をしてくれませんか？",
				Timestamp: time.Now(),
			},
		}, nil
	}

	return []Comment{}, nil
}

// PostComment は指定されたテキストをライブチャットに投稿します。
func (c *Client) PostComment(ctx context.Context, text string) error {
	// 💡 TODO: ここに実際の SDK 呼び出しロジックを実装
	// 1. c.youtubeService.LiveChatMessages.Insert を呼び出し、コメントを投稿

	// --- MOCK: 実際のコメント投稿をシミュレート ---
	log.Printf("YouTube Comment Posted [MOCK]: %s", text)
	return nil
}
