package youtube

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const (
	// コメントIDを保持する期間 (例: 1時間)
	commentIDRetentionDuration = 1 * time.Hour
)

// ErrLiveChatEnded はライブチャットが終了したことを示すカスタムエラー
var ErrLiveChatEnded = errors.New("live chat ended")

// Comment は YouTube のライブチャットメッセージを表す構造体
type Comment struct {
	ID        string
	AuthorID  string
	Author    string
	Message   string // 💡 修正: メッセージ本体のフィールド名は 'Message'
	Timestamp time.Time
}

// Client は YouTube Live Chat API との連携を管理します。
type Client struct {
	channelID string

	// 実際の YouTube SDK サービスインスタンスを保持
	service *youtube.Service

	// ライブチャットの状態を管理するためのフィールド
	liveChatID            string
	nextPageToken         string
	lastFetchedCommentIDs map[string]time.Time
}

// NewClient は新しい YouTube Client のインスタンスを作成します。
func NewClient(ctx context.Context, channelID string, oauthPort int) (*Client, error) {
	if channelID == "" {
		return nil, fmt.Errorf("youtube channel ID is empty")
	}

	log.Printf("YouTube Client: Starting OAuth2 setup using port %d...", oauthPort)

	// 1. 認証済み HTTP クライアントの取得 (GetOAuth2Clientは同じパッケージのauth.goにあります)
	// GetOAuth2Clientが未定義の場合、Goのビルドシステムはエラーを出しますが、ここでは存在すると仮定
	// GetOAuth2Client() が GetToken() に依存しているため、ロジックを auth.go の定義に合わせる

	// トークン設定は auth.go に依存するため、ここでは簡略化し、auth.go が適切なクライアントを返すものと仮定する。
	client, err := GetOAuth2Client(ctx, oauthPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	// 2. YouTube サービスインスタンスの初期化
	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create YouTube service: %w", err)
	}

	log.Printf("YouTube Service successfully initialized for channel %s.", channelID)

	return &Client{
		channelID:             channelID,
		service:               service,
		lastFetchedCommentIDs: make(map[string]time.Time),
	}, nil
}

// findLiveChatID はチャンネルの現在のライブブロードキャストを見つけ、そのライブチャットIDを返します。
func (c *Client) findLiveChatID(ctx context.Context) (string, error) {
	// 1. Search.List を呼び出し、"live" のブロードキャストを探す
	call := c.service.Search.List([]string{"id"}).
		ChannelId(c.channelID).
		EventType("live").
		Type("video").
		MaxResults(1)

	response, err := call.Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to search live broadcast: %w", err)
	}

	if len(response.Items) == 0 {
		return "", fmt.Errorf("no active live broadcast found for channel ID: %s", c.channelID)
	}

	videoID := response.Items[0].Id.VideoId

	// 2. Videos.List を呼び出し、ライブチャット ID を取得
	videosCall := c.service.Videos.List([]string{"liveStreamingDetails"}).
		Id(videoID)

	videosResp, err := videosCall.Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get video details: %w", err)
	}

	if len(videosResp.Items) == 0 || videosResp.Items[0].LiveStreamingDetails == nil || videosResp.Items[0].LiveStreamingDetails.ActiveLiveChatId == "" {
		return "", fmt.Errorf("live streaming details or active chat ID not available for video ID: %s", videoID)
	}

	liveChatID := videosResp.Items[0].LiveStreamingDetails.ActiveLiveChatId

	log.Printf("Found Active Live Chat ID: %s", liveChatID)
	return liveChatID, nil
}

// FetchLiveChatMessages は新しいライブチャットメッセージを取得します。
// 💡 修正: シグネチャを types.LowLatencyResponse に合わせ、ポーリング間隔を戻り値に含めます。
func (c *Client) FetchLiveChatMessages(ctx context.Context) ([]Comment, time.Duration, error) {
	// 1. 初回呼び出し時に liveChatID を検索し設定
	if c.liveChatID == "" {
		id, err := c.findLiveChatID(ctx)
		if err != nil {
			return nil, 0, err
		}
		c.liveChatID = id
	}

	// 2. LiveChatMessages.List を呼び出し
	call := c.service.LiveChatMessages.List(c.liveChatID, []string{"snippet", "authorDetails"})

	if c.nextPageToken != "" {
		call = call.PageToken(c.nextPageToken)
	}

	response, err := call.Context(ctx).Do()
	if err != nil {
		// YouTube API が返すエラーメッセージをチェック
		// "liveChatEnded" または類似のエラーメッセージが含まれるかチェック
		if strings.Contains(err.Error(), "liveChatEnded") || strings.Contains(err.Error(), "live chat is inactive") {
			// ライブチャット終了エラーの場合
			log.Printf("YouTube API Error: Live chat ended. Error: %v", err)
			c.liveChatID = "" // 💡 修正: liveChatID をリセット
			c.nextPageToken = ""
			return nil, 0, ErrLiveChatEnded // 💡 修正: カスタムエラーと 0s を返す
		}
		// その他のエラー
		return nil, 0, fmt.Errorf("failed to fetch live chat messages: %w", err)
	}

	// 3. 次のポーリングのためのトークンと間隔を更新
	c.nextPageToken = response.NextPageToken
	pollingInterval := time.Duration(response.PollingIntervalMillis) * time.Millisecond // 💡 修正: pollingInterval をここで定義

	// 4. メッセージを処理し、重複をフィルタリング
	var newComments []Comment
	currentTime := time.Now()

	for _, item := range response.Items {
		// YouTube Data APIの仕様: LiveChatMessage IDは item.Id
		commentID := item.Id

		// 4.1. 重複チェック
		if _, exists := c.lastFetchedCommentIDs[commentID]; exists {
			continue // 既に処理済みのためスキップ
		}

		// 4.2. 必須フィールドのチェック (AI応答に必要なメッセージ本文)
		if item.Snippet.DisplayMessage == "" {
			continue
		}

		// 4.3. コメントの構造体を作成
		newComment := Comment{
			ID:       commentID,
			AuthorID: item.AuthorDetails.ChannelId,
			Author:   item.AuthorDetails.DisplayName,
			Message:  item.Snippet.DisplayMessage, // 💡 修正: TextではなくMessageを使用
			// YouTubeのタイムスタンプはRFC3339形式
			Timestamp: parseYouTubeTimestamp(item.Snippet.PublishedAt),
		}

		newComments = append(newComments, newComment)

		// 4.4. 💡 新しいコメントIDをマップに記録
		c.lastFetchedCommentIDs[commentID] = currentTime
	}

	// 5. 💡 ガベージコレクションを実行し、古いエントリを削除
	c.cleanOldCommentIDs(currentTime)

	return newComments, pollingInterval, nil // 💡 修正: 正しい戻り値の数で返す
}

// cleanOldCommentIDs は保持期間を過ぎたコメントIDをマップから削除します。
func (c *Client) cleanOldCommentIDs(currentTime time.Time) {
	// ログの頻度を抑えるためのカウンター
	deletedCount := 0

	// 現在時刻から保持期間を引いたしきい値
	threshold := currentTime.Add(-commentIDRetentionDuration)

	for id, t := range c.lastFetchedCommentIDs {
		if t.Before(threshold) {
			delete(c.lastFetchedCommentIDs, id)
			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Printf("[YouTube Client] Cleaned %d old comment IDs. Total tracked: %d", deletedCount, len(c.lastFetchedCommentIDs))
	}
}

// PostComment は指定されたテキストをライブチャットに投稿します。
// ... (このメソッドは変更なしと仮定) ...

// parseYouTubeTimestamp は YouTube API のタイムスタンプ文字列を time.Time に変換します。
// これは YouTube の慣習的なユーティリティ関数であり、パッケージ内で定義されている必要があります。
func parseYouTubeTimestamp(t string) time.Time {
	parsedTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		log.Printf("Error parsing timestamp '%s': %v", t, err)
		return time.Time{} // パース失敗時はゼロ値を返す
	}
	return parsedTime
}

// PostComment は指定されたテキストをライブチャットに投稿します。
func (c *Client) PostComment(ctx context.Context, text string) error {
	// 1. liveChatID が設定されていることを確認
	if c.liveChatID == "" {
		return fmt.Errorf("live chat ID is not set. Cannot post comment")
	}

	// 2. 投稿する LiveChatMessage オブジェクトを作成
	message := &youtube.LiveChatMessage{
		Snippet: &youtube.LiveChatMessageSnippet{
			LiveChatId: c.liveChatID,
			Type:       "textMessageEvent",
			TextMessageDetails: &youtube.LiveChatTextMessageDetails{
				MessageText: text,
			},
		},
	}

	// 3. LiveChatMessages.Insert を呼び出し
	_, err := c.service.LiveChatMessages.Insert([]string{"snippet"}, message).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to post comment to live chat: %w", err)
	}

	log.Printf("YouTube Comment Posted successfully: %s", text)
	return nil
}
