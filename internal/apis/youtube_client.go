package apis

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
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

// YouTubeClient は YouTube Data API とやり取りするためのクライアントです。
type YouTubeClient struct {
	service         *youtube.Service // YouTube API Service
	channelID       string           // 監視対象のチャンネルID
	liveChatID      string           // 現在アクティブなライブチャットID (ポーリング中に更新される)
	lastCommentTime time.Time        // 最後に処理したコメントの投稿時間
}

// NewYouTubeClient は新しい YouTubeClient インスタンスを作成します。
func NewYouTubeClient(ctx context.Context, channelID string) (*YouTubeClient, error) {
	// 1. OAuth2 Config とトークンを読み込み
	config := util.GetOAuth2Config()
	token, err := util.LoadToken(util.TokenPath)
	if err != nil {
		return nil, fmt.Errorf("トークンファイルのロードに失敗: %w", err)
	}

	// 2. カスタム TokenSource の作成: リフレッシュ時に自動保存するロジックをラップ
	// これにより、トークンがリフレッシュされるたびに config/token.json が自動更新されます。
	tokenSource := util.NewAutoSavingTokenSource(config.TokenSource(ctx, token))

	// 3. HTTP Client の作成: トークンソースを使用
	httpClient := oauth2.NewClient(ctx, tokenSource)

	// 4. YouTube Service の作成
	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("YouTubeサービスAPIの初期化に失敗: %w", err)
	}

	slog.Info("YouTube API クライアントが正常に初期化されました。", "client_id_prefix", config.ClientID[:8])

	return &YouTubeClient{
		service:   service,
		channelID: channelID,
		// lastCommentTime の初期値は time.Time{} (ゼロ値)
		lastCommentTime: time.Time{},
	}, nil
}

// GetLiveChatIDFromChannel はチャンネルIDからアクティブなライブチャットIDを見つけます。
func (c *YouTubeClient) GetLiveChatIDFromChannel(ctx context.Context) (string, error) {
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
	return liveChatID, nil
}

// FetchLiveChatMessages はライブチャットIDを使用して新しいコメントを取得します。
func (c *YouTubeClient) FetchLiveChatMessages(ctx context.Context) ([]Comment, error) {
	// liveChatIDが設定されていない場合、アクティブなライブ配信を検索して設定
	if c.liveChatID == "" {
		var err error
		c.liveChatID, err = c.GetLiveChatIDFromChannel(ctx)
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
		// NOTE: APIは最新のメッセージを返すため、過去のメッセージを誤って再処理しないよう注意が必要です。
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
func (c *YouTubeClient) PostComment(ctx context.Context, message string) error {
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

// sanitizeMessage は Gemini からの応答を YouTube の制約に合わせて整形します。
func sanitizeMessage(message string) string {
	// 1. マークダウンのコードブロックを削除 (例: ```json)
	re := regexp.MustCompile("(?s)```.*?```")
	message = re.ReplaceAllString(message, "")

	// 2. 過剰な改行を削除し、文字列の先頭と末尾の空白を削除
	message = strings.TrimSpace(message)
	message = strings.ReplaceAll(message, "\n", " ")

	return message
}

// FetchAndProcessComments はコメントの取得、AI応答生成、および投稿を実行するメインロジックです。
// この関数は cmd/run.go からポーリングループ内で呼び出されます。
func FetchAndProcessComments(ctx context.Context, ytClient *YouTubeClient, geminiClient *GeminiClient, dryRun bool) error {
	newComments, err := ytClient.FetchLiveChatMessages(ctx)
	if err != nil {
		return fmt.Errorf("コメント取得エラー: %w", err)
	}

	if len(newComments) == 0 {
		slog.Info("新しいコメントはありませんでした。")
		return nil
	}

	slog.Info("新しいコメントを検出しました。", "count", len(newComments))

	// コメントごとにAI応答を生成
	for _, comment := range newComments {
		// 自身のコメントへの応答を防ぐ (DisplayNameが 'ライブちゃん' の場合など、設定に応じて調整)
		// 厳密には authorDetails.IsChatOwner/IsChatModerator などを確認すべきだが、シンプルにDisplayNameでフィルタリング
		// ただし、YouTube APIではAIの投稿者名（チャンネル名）を判別する標準的な方法がないため、
		// 投稿者IDではなくDisplayNameで「ライブちゃん」を弾くと、ユーザー名が「ライブちゃん」の視聴者も弾かれる可能性がある。
		// ここでは、投稿者を弾くのはやめて、応答の生成時にプロンプトで制御することを推奨します。

		slog.Info("コメントを処理中", "author", comment.Author, "message", comment.Message)

		// 1. Geminiで応答を生成
		response, err := geminiClient.GenerateResponse(ctx, comment.Author, comment.Message)
		if err != nil {
			slog.Error("Gemini応答生成エラー", "error", err, "author", comment.Author)
			continue
		}

		// 2. 応答を整形
		sanitizedResponse := sanitizeMessage(response)

		if sanitizedResponse == "" {
			slog.Info("AIが空の応答を生成しました (スパムまたは無効なコメントと判断)")
			continue
		}

		// 3. コメントの投稿
		if dryRun {
			slog.Warn("ドライラン: コメントは投稿されません。", "応答", sanitizedResponse)
		} else {
			if err := ytClient.PostComment(ctx, sanitizedResponse); err != nil {
				slog.Error("コメント投稿失敗", "error", err, "response", sanitizedResponse)
			}
		}
	}

	return nil
}
