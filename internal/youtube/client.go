package youtube

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"prompter-live-go/internal/util"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// Comment はライブチャットのコメントデータを保持します。
type Comment struct {
	ID      string
	Author  string
	Message string
	Time    time.Time
}

// Client は YouTube Data API とやり取りするためのクライアントです。
type Client struct {
	service         *youtube.Service // YouTube API Service
	channelID       string           // 監視対象のチャンネルID
	liveChatID      string           // 現在アクティブなライブチャットID (ポーリング中に更新される)
	lastCommentTime time.Time        // 最後に処理したコメントの投稿時間
}

// NewClient は新しい YouTubeClient インスタンスを作成します。
func NewClient(ctx context.Context, channelID string) (*Client, error) { // 関数名と戻り値の型を修正
	// 1. OAuth2 Config とトークンを読み込み
	// 修正: util.GetOAuth2Config に authPort を示す 0 を渡す
	config := util.GetOAuth2Config(0)
	token, err := util.LoadToken(util.TokenPath)
	if err != nil {
		return nil, fmt.Errorf("トークンファイルのロードに失敗: %w", err)
	}

	// 2. カスタム TokenSource の作成: リフレッシュ時に自動保存するロジックをラップ
	tokenSource := util.NewAutoSavingTokenSource(config.TokenSource(ctx, token))

	// 3. HTTP Client の作成: トークンソースを使用
	httpClient := oauth2.NewClient(ctx, tokenSource)

	// 4. YouTube Service の作成
	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("YouTubeサービスAPIの初期化に失敗: %w", err)
	}

	slog.Info("YouTube API クライアントが正常に初期化されました。", "client_id_prefix", config.ClientID[:8])

	return &Client{
		service:   service,
		channelID: channelID,
		// lastCommentTime の初期値は time.Time{} (ゼロ値)
		lastCommentTime: time.Time{},
	}, nil
}

// GetLiveChatIDFromChannel はチャンネルIDからアクティブなライブチャットIDを見つけます。
func (c *Client) GetLiveChatIDFromChannel(ctx context.Context) (string, error) {
	slog.Info("API呼び出し: アクティブなライブチャットIDを検索中", "channel_id", c.channelID)

	// 1. チャンネルのライブ配信中の動画を検索 (eventType=live)
	searchCall := c.service.Search.List([]string{"id"}).
		Context(ctx).
		ChannelId(c.channelID).
		EventType("live").
		Type("video").
		MaxResults(1) // 最新の1件のみ取得

	searchResponse, err := searchCall.Do()
	if err != nil {
		return "", fmt.Errorf("ライブ動画の検索に失敗: %w", err)
	}

	if len(searchResponse.Items) == 0 {
		return "", fmt.Errorf("現在アクティブなライブ配信が見つかりません (チャンネルID: %s)", c.channelID)
	}

	// 2. 見つかった動画のIDを取得
	videoID := searchResponse.Items[0].Id.VideoId

	// 3. 動画IDからライブチャットIDを取得
	videoCall := c.service.Videos.List([]string{"liveStreamingDetails"}).
		Context(ctx).
		Id(videoID)

	videoResponse, err := videoCall.Do()
	if err != nil {
		return "", fmt.Errorf("動画の詳細取得に失敗: %w", err)
	}

	// ライブストリーミングの詳細情報が存在し、かつライブチャットIDが存在するか確認
	if len(videoResponse.Items) == 0 || videoResponse.Items[0].LiveStreamingDetails == nil || videoResponse.Items[0].LiveStreamingDetails.ActiveLiveChatId == "" {
		return "", fmt.Errorf("動画ID %s にアクティブなライブチャットIDが見つかりません", videoID)
	}

	liveChatID := videoResponse.Items[0].LiveStreamingDetails.ActiveLiveChatId
	slog.Info("ライブチャットIDを取得しました。", "live_chat_id", liveChatID, "video_id", videoID)
	// ライブチャットIDを更新
	c.liveChatID = liveChatID
	return liveChatID, nil
}

// FetchLiveChatMessages はライブチャットIDを使用して新しいコメントを取得します。
func (c *Client) FetchLiveChatMessages(ctx context.Context) ([]Comment, error) {
	if c.liveChatID == "" {
		// liveChatIDがまだ設定されていない場合、取得を試みる
		_, err := c.GetLiveChatIDFromChannel(ctx)
		if err != nil {
			return nil, fmt.Errorf("ライブチャットIDの取得に失敗: %w", err)
		}
	}

	slog.Info("API呼び出し: 新しいライブチャットメッセージを取得中", "live_chat_id", c.liveChatID)

	call := c.service.LiveChatMessages.List(c.liveChatID, []string{"snippet", "authorDetails"}).
		Context(ctx).
		MaxResults(200) // 一度に取得するコメント数の上限

	response, err := call.Do()
	if err != nil {
		// APIエラーの場合、liveChatIDをリセットして次のポーリングで再取得を試みる
		if strings.Contains(err.Error(), "liveChatEnded") {
			slog.Warn("ライブチャットが終了しました。liveChatIDをリセットします。")
			c.liveChatID = ""
		}
		return nil, fmt.Errorf("ライブチャットメッセージの取得に失敗: %w", err)
	}

	newComments := []Comment{}
	for _, item := range response.Items {
		// 投稿時間をパース
		publishedTime, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		if err != nil {
			slog.Error("コメント時間のパースに失敗", "error", err, "time_string", item.Snippet.PublishedAt)
			continue
		}

		// lastCommentTime より厳密に新しいコメントのみを処理
		if publishedTime.After(c.lastCommentTime) {
			comment := Comment{
				ID:      item.Id,
				Author:  item.AuthorDetails.DisplayName,
				Message: item.Snippet.DisplayMessage,
				Time:    publishedTime,
			}
			newComments = append(newComments, comment)
		}
	}

	// 最後に処理したコメントの時間を更新
	if len(newComments) > 0 {
		// 新しいコメントの中で最も新しい時間を取得
		c.lastCommentTime = newComments[len(newComments)-1].Time
	}

	return newComments, nil
}

// PostComment は指定された動画のチャットにコメントを投稿します。
func (c *Client) PostComment(ctx context.Context, message string) error {
	slog.Info("API呼び出し: コメント投稿中", "live_chat_id", c.liveChatID, "message_len", len(message))

	// コメントの構造を作成
	comment := &youtube.LiveChatMessage{
		Snippet: &youtube.LiveChatMessageSnippet{
			LiveChatId: c.liveChatID,
			Type:       "textMessageEvent",
			TextMessageDetails: &youtube.LiveChatTextMessageDetails{
				MessageText: message,
			},
		},
	}

	// 投稿実行
	_, err := c.service.LiveChatMessages.Insert([]string{"snippet"}, comment).
		Context(ctx).
		Do()

	if err != nil {
		return fmt.Errorf("コメントの投稿に失敗: %w", err)
	}

	slog.Info("コメント投稿成功", "message", message)
	return nil
}

// GetLiveChatID は現在の LiveChatID を返します。
func (c *Client) GetLiveChatID() string { // ここを修正
	return c.liveChatID
}
