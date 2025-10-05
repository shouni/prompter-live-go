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

// ErrLiveChatEnded はライブチャットが終了したことを示すカスタムエラー
var ErrLiveChatEnded = errors.New("live chat ended")

// Comment は YouTube のライブチャットメッセージを表す構造体
type Comment struct {
	Author    string
	Message   string
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
	lastFetchedCommentIDs map[string]struct{}
}

// NewClient は新しい YouTube Client のインスタンスを作成します。
func NewClient(ctx context.Context, channelID string, oauthPort int) (*Client, error) {
	if channelID == "" {
		return nil, fmt.Errorf("youtube channel ID is empty")
	}

	log.Printf("YouTube Client: Starting OAuth2 setup using port %d...", oauthPort)

	// 1. 認証済み HTTP クライアントの取得 (GetOAuth2Clientは同じパッケージのauth.goにあります)
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
		lastFetchedCommentIDs: make(map[string]struct{}),
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
func (c *Client) FetchLiveChatMessages(ctx context.Context) ([]Comment, error) {
	// 1. 初回呼び出し時に liveChatID を検索し設定
	if c.liveChatID == "" {
		id, err := c.findLiveChatID(ctx)
		if err != nil {
			return nil, err
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
			return nil, ErrLiveChatEnded // 💡 修正: カスタムエラーを返す
		}
		// その他のエラー
		return nil, fmt.Errorf("failed to fetch live chat messages: %w", err)
	}

	// 3. 次のポーリングのためのトークンを更新
	c.nextPageToken = response.NextPageToken

	// 4. 結果を Comment スライスに変換
	var comments []Comment
	for _, item := range response.Items {
		// 💡 修正: PublishedAt (RFC3339文字列) をパースする
		publishedAt := time.Now() // フォールバック

		if item.Snippet.PublishedAt != "" {
			parsedTime, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
			if err != nil {
				log.Printf("Warning: Failed to parse timestamp '%s': %v", item.Snippet.PublishedAt, err)
			} else {
				publishedAt = parsedTime
			}
		}

		comments = append(comments, Comment{
			Author:    item.AuthorDetails.DisplayName,
			Message:   item.Snippet.DisplayMessage,
			Timestamp: publishedAt,
		})
	}

	log.Printf("Successfully fetched %d new messages. Next token: %s", len(comments), c.nextPageToken)
	return comments, nil
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
