package apis

import (
	"context"
	"fmt"
	"time"

	"prompter-live-go/internal/util"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// YouTubeClient は YouTube Data API と連携するためのクライアントです。
type YouTubeClient struct {
	service *youtube.Service
	config  *oauth2.Config
	token   *oauth2.Token
}

// NewYouTubeClient は新しい YouTubeClient のインスタンスを作成し、APIサービスを初期化します。
func NewYouTubeClient(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (*YouTubeClient, error) {
	client := &YouTubeClient{
		config: config,
		token:  token,
	}
	httpClient := config.Client(ctx, token)
	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("YouTube サービス初期化に失敗: %w", err)
	}
	client.service = service
	fmt.Println("YouTube API クライアントが正常に初期化されました。")
	return client, nil
}

// PollingFunc はポーリングサイクルで実行される処理の型定義です。
type PollingFunc func(ctx context.Context, yc *YouTubeClient, gc *GeminiClient, channelID string, dryRun bool) error

// StartPolling は定期的に API を呼び出すポーリングループを実行します。
func (c *YouTubeClient) StartPolling(ctx context.Context, cycleFunc PollingFunc, gc *GeminiClient, channelID string, pollingInterval time.Duration, dryRun bool) error {
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	fmt.Printf("📢 ポーリングを開始します。間隔: %s\n", pollingInterval)

	// サービス開始時にも一度ポーリングを実行
	if err := cycleFunc(ctx, c, gc, channelID, dryRun); err != nil {
		fmt.Printf("⚠️ サービス起動時の初回ポーリングエラー: %v\n", err)
	}

	for {
		select {
		case <-ticker.C:
			// Ticker が切れるごとにポーリングサイクルを実行
			if err := cycleFunc(ctx, c, gc, channelID, dryRun); err != nil {
				fmt.Printf("⚠️ ポーリング実行エラー: %v\n", err)
			}

			// ポーリング後、トークンの状態をチェックし、リフレッシュされていたら保存
			if c.CheckTokenRefreshStatus() {
				fmt.Println("♻️ トークンがリフレッシュされました。ファイルに保存します...")

				newToken := c.GetToken()
				if err := util.SaveToken(util.TokenPath, newToken); err != nil {
					fmt.Printf("❌ リフレッシュトークンの保存に失敗: %v\n", err)
				} else {
					fmt.Println("✅ 新しいトークンを 'config/token.json' に保存しました。")
				}
			}

		case <-ctx.Done():
			fmt.Println("📢 ポーリングサービスが停止されました。")
			return nil
		}
	}
}

// GetLatestComments は指定されたチャンネルIDの最新コメントを取得します。
func (c *YouTubeClient) GetLatestComments(channelID string) ([]*youtube.CommentThread, error) {
	fmt.Printf("[%s] API呼び出し: 最新コメントを取得中 (チャンネルID: %s)...\n", time.Now().Format("15:04:05"), channelID)

	call := c.service.CommentThreads.List([]string{"snippet"})
	call = call.MaxResults(10)
	call = call.ChannelId(channelID)
	call = call.Order("time")

	response, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("YouTube APIからコメントスレッドの取得に失敗: %w", err)
	}

	fmt.Printf("[%s] API応答: コメントスレッドを %d 件取得しました。\n", time.Now().Format("15:04:05"), len(response.Items))

	return response.Items, nil
}

// PostReply は指定された親コメントIDに対して、AI応答を返信として投稿します。
func (c *YouTubeClient) PostReply(parentCommentID, text string) (*youtube.Comment, error) {
	fmt.Printf("API呼び出し: コメントID %s に返信を投稿中...\n", parentCommentID)

	comment := &youtube.Comment{
		Snippet: &youtube.CommentSnippet{
			ParentId:     parentCommentID,
			TextOriginal: text,
		},
	}

	call := c.service.Comments.Insert([]string{"snippet"}, comment)

	result, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("YouTube APIへ返信コメントの投稿に失敗: %w", err)
	}

	fmt.Printf("✅ 返信投稿成功！\n")
	return result, nil
}

// GetToken returns the current token, primarily used for saving the token after a refresh.
func (c *YouTubeClient) GetToken() *oauth2.Token {
	return c.token
}

// CheckTokenRefreshStatus は、トークンがリフレッシュされたかどうかをチェックするダミー関数です。
func (c *YouTubeClient) CheckTokenRefreshStatus() bool {
	if time.Until(c.token.Expiry) < 5*time.Minute {
		return true
	}
	return false
}
