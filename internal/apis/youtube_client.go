package apis

import (
	"context"
	"fmt"
	"time" // net/http を削除

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

// YouTubeClient は YouTube Data API と連携するためのクライアントです。
type YouTubeClient struct {
	// YouTube Data API サービスオブジェクト。これを使ってAPI呼び出しを行います。
	service *youtube.Service
	// 認証トークンの設定
	config *oauth2.Config
	// 現在のトークン情報（アクセストークン、リフレッシュトークン、有効期限など）
	token *oauth2.Token
}

// NewYouTubeClient は新しい YouTubeClient のインスタンスを作成し、APIサービスを初期化します。
func NewYouTubeClient(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (*YouTubeClient, error) {
	client := &YouTubeClient{
		config: config,
		token:  token,
	}

	// 1. トークンを使って HTTP クライアントを生成
	// oauth2.Config.Client はトークン期限切れの場合、リフレッシュトークンを使って
	// 自動的にトークンをリフレッシュする機能を持っています。
	httpClient := config.Client(ctx, token)

	// 2. HTTP クライアントを使って YouTube サービスを初期化
	service, err := youtube.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("YouTube サービス初期化に失敗: %w", err)
	}
	client.service = service

	// 3. 初期トークンの有効性を確認（オプション）
	// この時点でトークンがリフレッシュされていれば、client.token に新しい情報が書き込まれます。
	// ただし、Client() はトークンリフレッシュが発生しても、client.tokenを自動更新しません。
	// 実際のリフレッシュの保存ロジックは、ポーリング時にトークンをチェックして行います。

	fmt.Println("YouTube API クライアントが正常に初期化されました。")
	return client, nil
}

// GetLatestComments は指定されたチャンネルIDの最新コメントを取得します。
// 実際のポーリングでは、pageTokenやlastCheckedTimeを使って絞り込みますが、今回は基本形を定義します。
func (c *YouTubeClient) GetLatestComments(channelID string) ([]*youtube.CommentThread, error) {
	fmt.Printf("API呼び出し: 最新コメントを取得中 (チャンネルID: %s)...\n", channelID)

	// チャンネルに紐づく全ての動画のコメントを取得するため、
	// 実際にはチャンネルの動画リストを取得し、それぞれの動画のコメントを取得する必要があります。
	// ここでは、最も簡単な方法として「チャンネルのアップロードリスト」のコメントスレッドを取得するクエリをシミュレートします。

	call := c.service.CommentThreads.List([]string{"snippet"})

	// APIクォータ節約のため、最大10件に制限
	call = call.MaxResults(10)

	// チャンネルの最新動画のコメントを取得するため、'channelId' を使用
	// AllThreads はCommentThreadsListCallのメソッドではないため削除
	call = call.ChannelId(channelID)

	response, err := call.Do()
	if err != nil {
		// APIからエラーが返された場合、トークン切れの可能性がある
		return nil, fmt.Errorf("YouTube APIからコメントスレッドの取得に失敗: %w", err)
	}

	// ログとして取得件数を出力
	fmt.Printf("API応答: コメントスレッドを %d 件取得しました。\n", len(response.Items))

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

	// CommentThreads ではなく Comments サービスを使って返信を投稿
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
// oauth2.Client は内部でリフレッシュを行いますが、新しいトークンを外部に公開するフックがないため、
// 実際にはトークンを保持しているクライアント側で状態を定期的に確認する必要があります。
// ここでは、ダミーとして現在のトークンの有効期限をチェックします。
func (c *YouTubeClient) CheckTokenRefreshStatus() bool {
	// 有効期限が残り5分を切ったら、トークンが古い可能性があると判断
	if time.Until(c.token.Expiry) < 5*time.Minute {
		// 本番環境では、ここでトークンをリフレッシュし、新しいトークンを返すべきですが、
		// oauth2.Client は呼び出し時に自動リフレッシュを行うため、ここでは状態通知のみ。
		// 実際のリフレッシュは GetLatestComments などAPI呼び出し時に暗黙的に行われます。
		return true // トークンが期限切れに近い、または期限切れである
	}
	return false
}
