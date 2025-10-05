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
	// 実際のSDKクライアントの認証情報を保持
	apiKey string
	// TODO: ここに実際の Gemini SDK クライアントインスタンスを保持します
}

// --- SDK型をラップする構造体とインターフェースの定義 ---

// sdkLiveStream は、SDKのConnect呼び出しが返す生のセッションオブジェクトのインターフェースです。
type sdkLiveStream interface {
	// 💡 TODO: SDK の Send メソッドのシグネチャに置き換える必要があります。
	SDKSend(data interface{}) error

	// 💡 TODO: SDK の Recv メソッドのシグネチャに置き換える必要があります。
	SDKRecv() (interface{}, error)
	Close() error
}

// liveSessionWrapper は sdkLiveStream をラップし、LiveSession インターフェースを満たします。
type liveSessionWrapper struct {
	session sdkLiveStream
}

// Send はパイプラインの型をSDKが要求するペイロードに変換して送信します。
func (w *liveSessionWrapper) Send(data types.LiveStreamData) error {
	log.Printf("LiveSession: Sending input data (MimeType: %s, Data length: %d)", data.MimeType, len(data.Data))

	// 💡 TODO: ここに実際の SDK 呼び出しロジックを実装
	// 1. data を SDK が要求するペイロード型に変換
	// 2. w.session.SDKSend(convertedPayload) を呼び出す
	return w.session.SDKSend(data)
}

// RecvResponse はSDKからの応答をパイプラインの型に変換して返します。
func (w *liveSessionWrapper) RecvResponse() (*types.LowLatencyResponse, error) {
	// 💡 修正: 未使用の rawResp を破棄変数 (_) に変更し、エラーをチェック
	_, err := w.session.SDKRecv()
	if err != nil {
		return nil, err
	}

	// 💡 TODO: ここに実際の SDK 応答の解析ロジックを実装

	// MOCK: 解析結果をシミュレーション
	time.Sleep(50 * time.Millisecond) // 遅延をシミュレート

	return &types.LowLatencyResponse{
		Text: "AIが生成したテキスト（MOCK）",
		Done: false,
	}, nil
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
	// TODO: ここで実際の Gemini SDK Client を初期化し、*LiveClient に保持します。
	return &LiveClient{apiKey: apiKey}, nil
}

// Connect は Gemini Live API への新しいセッションを確立します。
func (c *LiveClient) Connect(ctx context.Context, config types.LiveAPIConfig) (LiveSession, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("cannot connect: api key is missing")
	}

	log.Printf("Connecting to Live API with model: %s, Instruction: %s...", config.Model, config.SystemInstruction)

	// 💡 TODO: ここに実際の SDK 接続ロジックを実装
	rawSession := newMockSession()

	return &liveSessionWrapper{session: rawSession}, nil
}

// --- MOCK: SDKの挙動をシミュレートするためのダミー実装 ---

// mockSession は sdkLiveStream インターフェースを満たすダミー構造体
type mockSession struct {
	// ストリームの終了をシミュレートするためのカウンタ
	recvCount int
}

func newMockSession() *mockSession {
	log.Println("[MOCK] Created dummy SDK Live Session. Only 5 messages will be simulated.")
	return &mockSession{}
}

func (m *mockSession) SDKSend(data interface{}) error {
	log.Printf("[MOCK] Input data received by SDK MOCK. (Type: %T)", data)
	return nil
}

func (m *mockSession) SDKRecv() (interface{}, error) {
	m.recvCount++
	if m.recvCount > 5 {
		// 5回応答をシミュレートした後、ストリーム終了をシミュレート
		log.Println("[MOCK] Simulated stream end.")
		return nil, fmt.Errorf("EOF") // ストリーム終了をエラーとして返すのが一般的
	}

	time.Sleep(10 * time.Millisecond)

	// 実際の SDK 応答型のダミー構造体を返す
	return struct {
		Text string
		Done bool
	}{"chunk", false}, nil
}

func (m *mockSession) Close() error {
	log.Println("[MOCK] SDK Session Closed.")
	return nil
}
