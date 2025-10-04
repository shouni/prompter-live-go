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
	// CSRF対策：期待されるstate値を格納
	ExpectedState string
}

// NewOAuthServer は新しいOAuthServerのインスタンスを作成します。
func NewOAuthServer(port string) *OAuthServer {
	return &OAuthServer{
		// バッファなしのチャネルを作成
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

// Stop はサーバーを停止し、チャネルをクローズします。
// チャネルのクローズ責任は、呼び出し元（cmd/auth.go）に移譲されました。
func (s *OAuthServer) Stop() {
	if s.server != nil {
		// タイムアウトを設定してサーバーを停止
		ctx, cancel := context.WithTimeout(context.Background(), 5)
		defer cancel()
		s.server.Shutdown(ctx)
		// サーバーを停止した後にチャネルをクローズ
		close(s.CodeChan)
	}
}

// handleCallback は Googleからの認証コードを含むリクエストを処理します。
func (s *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	// r.Context()がキャンセルされる前に処理を完了させる

	code := r.URL.Query().Get("code")
	receivedState := r.URL.Query().Get("state")

	// 期待されるstateと受信したstateを比較し、不一致ならエラーとする
	if receivedState != s.ExpectedState {
		errorMsg := "CSRF攻撃の可能性、またはstateパラメータが不正です。"
		// ユーザーに表示するエラーメッセージをブラウザに出力
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "<h1>認証エラー (セキュリティ)</h1><p>%s</p><p>認証を最初からやり直してください。</p>", errorMsg)

		// auth.go側へは空文字列（エラー通知）を送信
		s.CodeChan <- ""
		return
	}

	if code != "" {
		// 認証コードをチャネルに送信（チャネルのクローズは呼び出し元が行う）
		s.CodeChan <- code

		// ユーザーのブラウザに応答メッセージを返す
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>認証が完了しました！</h1><p>ブラウザを閉じて、Prompter Live Goに戻ってください。</p>")
	} else {
		// エラーハンドリング
		errorMsg := r.URL.Query().Get("error")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "<h1>認証エラー</h1><p>認証に失敗しました: %s</p>", errorMsg)

		// エラー時にも auth.go が待機を解除できるよう、空文字列を送信
		s.CodeChan <- ""
	}

	// ⚠️ ここでの s.Stop() は削除済み
}
