package apis

import (
	"context"
	"fmt"
	"net/http"
)

// OAuthServer は認証フローでGoogleからのコールバックを受け取るためのサーバーを定義します。
type OAuthServer struct {
	// 受け取った認証コードを格納するためのチャネル
	CodeChan chan string
	// サーバー自身への参照
	server *http.Server
	// サーバーが待ち受けるポート
	port string
}

// NewOAuthServer は新しいOAuthServerのインスタンスを作成します。
func NewOAuthServer(port string) *OAuthServer {
	return &OAuthServer{
		CodeChan: make(chan string),
		port:     port,
	}
}

// Start はローカルサーバーを起動し、認証コードのコールバックを待ち受けます。
func (s *OAuthServer) Start() {
	// サーバーを定義
	s.server = &http.Server{Addr: ":" + s.port}

	// コールバックURLのパスとハンドラーを設定
	http.HandleFunc("/oauth/callback", s.handleCallback)

	fmt.Printf("💡 認証コードを待ち受けています: http://localhost:%s/oauth/callback\n", s.port)

	// 非同期でサーバーを起動
	go func() {
		// サーバー起動。Graceful Shutdownのためにエラーログを出力
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("❌ OAuthサーバーが予期せぬエラーで停止しました: %v\n", err)
		}
	}()
}

// Stop はサーバーを停止します。
func (s *OAuthServer) Stop() {
	if s.server != nil {
		// タイムアウトを設定してサーバーを停止
		ctx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()
		s.server.Shutdown(ctx)
		// CodeChanを閉じ、リソースを解放
		close(s.CodeChan)
	}
}

// handleCallback は Googleからの認証コードを含むリクエストを処理します。
func (s *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// URLクエリから認証コードを取得
	code := r.URL.Query().Get("code")

	if code != "" {
		// 認証コードをチャネルに送信
		s.CodeChan <- code

		// ユーザーのブラウザに応答メッセージを返す
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>認証が完了しました！</h1><p>ブラウザを閉じて、Prompter Live Goに戻ってください。</p>")
	} else {
		// エラーハンドリング
		errorMsg := r.URL.Query().Get("error")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "<h1>認証エラー</h1><p>認証に失敗しました: %s</p>", errorMsg)
		// エラーの場合もチャネルに空文字列を送るなどで処理を先に進める設計も可能
	}

	// コードを受け取ったら、即座にサーバーを停止
	s.Stop()
}
