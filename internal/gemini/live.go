package gemini

import (
	"context"
	"fmt"
	"log"
	"time"

	"prompter-live-go/internal/types"
)

// LiveSession はパイプラインが使用するGemini Liveセッションのインターフェースを定義します。
// これは pipeline.LiveSession と統合されます。
type LiveSession interface {
	Send(data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close() error
}

// LiveClient は Gemini Live API への接続を管理するためのクライアント構造体です。
type LiveClient struct {
	// 実際のSDKクライアントは構造体内部で保持されるが、ここでは抽象化
	apiKey string
}

// --- SDK型をラップする構造体とインターフェースの定義 ---

// sdkLiveStream は、SDKがConnect呼び出しで返すオブジェクトが満たすべき
// 最小限のメソッドを定義します。これにより、Connectメソッドの戻り値の型を
// interface{} からこのインターフェース型へ置き換えることを意図します。
type sdkLiveStream interface {
	SDKSend(data interface{}) error // SDKが要求する形式で送信
	SDKRecv() (interface{}, error)  // SDKの生応答を受信
	Close() error
}

// liveSessionWrapper は sdkLiveStream をラップし、外部から利用される LiveSession インターフェースを満たします。
type liveSessionWrapper struct {
	session sdkLiveStream
}

// Send はパイプラインの型をSDKの型に変換して送信します。
func (w *liveSessionWrapper) Send(data types.LiveStreamData) error {
	// 実際には data をSDKが要求するペイロードに変換して w.session.SDKSend(..) を呼び出す
	log.Printf("[MOCK] Sending data: MimeType=%s", data.MimeType)
	return w.session.SDKSend(data)
}

// RecvResponse はSDKからの応答をパイプラインの型に変換して返します。
func (w *liveSessionWrapper) RecvResponse() (*types.LowLatencyResponse, error) {
	// 実際には w.session.SDKRecv() からSDKの応答を取得し、types.LowLatencyResponse に変換する
	// MOCK: ダミーの応答を返す
	// Note: パイプラインが動作するようにダミー応答を継続的に返す必要がある
	return &types.LowLatencyResponse{Text: "[MOCK] AI response chunk.", Done: false}, nil
}

// Close はセッションを閉じます。
func (w *liveSessionWrapper) Close() error {
	log.Println("Closing Gemini Live session.")
	return w.session.Close()
}

// --- LiveClientの実装 ---

// NewLiveClient は LiveClient の新しいインスタンスを作成します。
func NewLiveClient(ctx context.Context, apiKey string) (*LiveClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is empty")
	}
	log.Println("Gemini Live Client initialized.")
	return &LiveClient{apiKey: apiKey}, nil
}

// Connect は Gemini Live API への新しいセッションを確立します。
//
// 💡 クリティカルな指摘への対応:
// 接続結果を interface{} で受け取り型アサートするのではなく、SDKが直接 Liveセッションインターフェースを
// 返すと仮定する設計に変更することで、アプリケーションコード内の脆い型アサーションを排除します。
func (c *LiveClient) Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("cannot connect: api key is missing")
	}
	log.Printf("Connecting to Live API with model: %s", config.Model)

	// --- MOCK: 実際のSDK接続ロジックをシミュレート ---
	// 実際には SDK の Connect 呼び出しが行われ、sdkLiveStream インターフェースを満たすオブジェクトが返る

	// SDK接続が成功した場合、sdkLiveStreamを満たすオブジェクトが返されると仮定
	rawSession := newMockSession() // newMockSessionはsdkLiveStreamを満たす必要あり

	// 脆弱な型アサーションを伴う interface{} の受け渡しを排除し、
	// LiveSession ラッパーを作成して返します。
	return &liveSessionWrapper{session: rawSession}, nil
}

// --- MOCK: SDKの挙動をシミュレートするためのダミー実装 ---

// mockSession は sdkLiveStream インターフェースを満たすダミー構造体
type mockSession struct{}

func newMockSession() *mockSession {
	log.Println("[MOCK] Created dummy SDK Live Session.")
	return &mockSession{}
}

func (m *mockSession) SDKSend(data interface{}) error {
	// ダミー送信ロジック
	return nil
}

func (m *mockSession) SDKRecv() (interface{}, error) {
	// ダミー受信ロジック
	time.Sleep(100 * time.Millisecond) // 遅延をシミュレート
	// RecvResponseラッパー内でテキストを生成するため、ここではnilを返す
	return nil, nil
}

func (m *mockSession) Close() error {
	return nil
}
