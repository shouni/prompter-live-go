package gemini

import (
	"context"
	"fmt"
	"log"
	"time"

	"prompter-live-go/internal/types"
)

// LiveSession はパイプラインが利用する、Gemini Liveセッションのインターフェースです。
type LiveSession interface {
	Send(data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close() error
}

// LiveClient は Gemini Live API への接続を管理するためのクライアント構造体です。
type LiveClient struct {
	// 実際のSDKクライアントを保持
	apiKey string
	// MOCK: SDKクライアントは外部から注入されると仮定
}

// --- SDK型をラップする構造体とインターフェースの定義 ---

// sdkLiveStream は、SDKのConnect呼び出しが返す生のセッションオブジェクトのインターフェースです。
type sdkLiveStream interface {
	SDKSend(data interface{}) error
	SDKRecv() (interface{}, error)
	Close() error
}

// liveSessionWrapper は sdkLiveStream をラップし、LiveSession インターフェースを満たします。
type liveSessionWrapper struct {
	session sdkLiveStream
}

// Send はパイプラインの型をSDKが要求するペイロードに変換して送信します。
func (w *liveSessionWrapper) Send(data types.LiveStreamData) error {
	log.Printf("LiveSession: Sending input data (MimeType: %s)", data.MimeType)
	return w.session.SDKSend(data)
}

// RecvResponse はSDKからの応答をパイプラインの型に変換して返します。
func (w *liveSessionWrapper) RecvResponse() (*types.LowLatencyResponse, error) {
	// 💡 修正点: rawResp, err := w.session.SDKRecv() の戻り値として rawResp を受け取る必要がないため、
	// 変数を破棄する (_) に変更します。
	_, err := w.session.SDKRecv()
	if err != nil {
		return nil, err
	}

	// 実際には rawResp を解析し、types.LowLatencyResponse に変換するロジックが必要
	// MOCK: SDK応答構造体を types.LowLatencyResponse に変換すると仮定

	// ダミーの待ち時間を追加し、ストリーム応答をシミュレート
	time.Sleep(50 * time.Millisecond)

	// MOCK: テキストを空にし、Doneをfalseにすることで、パイプライン側で応答を待機させる
	return &types.LowLatencyResponse{Text: "", Done: false}, nil
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
func (c *LiveClient) Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("cannot connect: api key is missing")
	}

	log.Printf("Connecting to Live API with model: %s, Instruction: %s...", config.Model, config.SystemInstruction)

	// --- MOCK: 実際のSDK接続ロジックをシミュレート ---
	rawSession := newMockSession() // sdkLiveStreamを満たすダミーセッション

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
	return nil
}

func (m *mockSession) SDKRecv() (interface{}, error) {
	// パイプラインがブロックされないように、短い遅延を入れる
	time.Sleep(10 * time.Millisecond)
	return nil, nil
}

func (m *mockSession) Close() error {
	return nil
}
