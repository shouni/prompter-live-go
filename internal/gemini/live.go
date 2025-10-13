package gemini

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"prompter-live-go/internal/types"

	"google.golang.org/genai"
)

// geminiLiveSession は Session インターフェースの実際の内部的な実装です。
// Live API のストリーミングやチャットのロジックをラップします。
type geminiLiveSession struct {
	mu         sync.Mutex // 送信時の排他制御用
	baseClient *genai.Client
	modelName  string
	config     types.LiveAPIConfig

	// genai.ChatSession は、実際の Go SDK でチャット履歴と状態を管理するために使用されます。
	// ビルドエラーを避けるため、ここでは便宜的に interface{} を使用しますが、
	// 実際には *genai.ChatSession など適切な型に置き換える必要があります。
	chatSession interface{}

	responseChan chan *types.LowLatencyResponse // AIからの応答をパイプラインに送るチャネル
	doneChan     chan struct{}                  // セッション終了を通知するチャネル
}

// newGeminiLiveSession は Session を実装した新しい geminiLiveSession を作成します。
// Client.StartSessionから呼び出されます。
func newGeminiLiveSession(client *genai.Client, modelName string, config types.LiveAPIConfig, systemInstruction string) Session {
	log.Printf("Internal Session created - Model: %s, Instruction: %s", modelName, systemInstruction)

	// 実際には、ここで genai.Client を使って ChatSession の初期化を行います。

	return &geminiLiveSession{
		baseClient:   client,
		modelName:    modelName,
		config:       config,
		responseChan: make(chan *types.LowLatencyResponse, 10), // バッファ付きチャネル
		doneChan:     make(chan struct{}),
	}
}

// Send はデータをAIに送信し、応答処理を開始します。（仮実装）
func (s *geminiLiveSession) Send(ctx context.Context, data types.LiveStreamData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// チャットが閉じられているか確認
	select {
	case <-s.doneChan:
		return errors.New("session already closed")
	default:
		// 続行
	}

	log.Printf("Sending data to Gemini (Placeholder) - Author: %s, Text: %v", data.Author, data.Text)

	// 実際のロジック:
	// 1. data.Text を genai.Content に変換
	// 2. ChatSession または Streaming API を呼び出す
	// 3. 応答ストリームを読み取り、チャンクを LowLatencyResponse に変換して s.responseChan に書き込むゴルーチンを開始

	// デモ応答を responseChan に送る（パイプラインの動作確認用）
	go func() {
		time.Sleep(100 * time.Millisecond) // 応答遅延をシミュレート
		select {
		// types.LowLatencyResponse のフィールド名に合わせて ResponseText を使用
		case s.responseChan <- &types.LowLatencyResponse{ResponseText: fmt.Sprintf("AI response to: %s", data.Text), Done: true}:
			// 成功
		case <-ctx.Done():
			// コンテキストがキャンセルされた
		case <-s.doneChan:
			// セッションが閉じられた
		}
	}()

	return nil
}

// RecvResponse は完全な応答が来るまで待ち受け、それを返します。（仮実装）
func (s *geminiLiveSession) RecvResponse() (*types.LowLatencyResponse, error) {
	select {
	case resp, ok := <-s.responseChan:
		if !ok {
			return nil, io.EOF // チャネルが閉じられたら EOF を返す
		}
		return resp, nil
	case <-s.doneChan:
		return nil, io.EOF
	}
}

// Close はセッションを閉じ、リソースを解放します。
func (s *geminiLiveSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.doneChan:
		// 既に閉じている
		return
	default:
		// クローズ処理
		close(s.doneChan)
		// responseChan は doneChan が閉じられた後に処理を停止するため、安全のために閉じます
		close(s.responseChan)
		log.Println("Gemini Live Session closed.")
	}
}
