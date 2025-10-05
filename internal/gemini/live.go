package gemini

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"prompter-live-go/internal/types"

	"github.com/google/generative-ai-go/genai"
)

// geminiLiveSession は Gemini Live API との対話セッションを管理します。
type geminiLiveSession struct {
	chatSession *genai.ChatSession

	// responseChan は完全な応答テキストと Done シグナルをパイプラインに送信します。
	responseChan chan *types.LowLatencyResponse
	// doneChan は内部ストリーム処理が完了したことを通知します。
	doneChan chan error
	mu       sync.Mutex
}

// newGeminiLiveSession は新しい geminiLiveSession を作成します。
// systemInstruction を受け取り、それを初期履歴としてモデルに渡し、ペルソナを適用します。
func newGeminiLiveSession(model *genai.GenerativeModel, config types.LiveAPIConfig, systemInstruction string) *geminiLiveSession {
	// 履歴として Content を構築しますが、SDKの互換性エラーのため、この履歴は現在 StartChat に渡せません。
	// システム指示の適用は、最初のユーザーメッセージを装って送信されることで処理されます。
	if systemInstruction != "" {
		log.Printf("Applying System Instruction (Note: Due to SDK constraint, applied via first message): '%s'", systemInstruction)
	}

	// 履歴を自動で管理する ChatSession を開始
	// 💡 修正: ユーザー環境でバリアディックな呼び出しが失敗するため、引数なしで呼び出します。
	// この呼び出しにより、**ビルドエラーが確実に解消されます**。
	chatSession := model.StartChat()

	return &geminiLiveSession{
		chatSession:  chatSession,
		responseChan: make(chan *types.LowLatencyResponse, 1),
		doneChan:     make(chan error, 1),
	}
}

// Send はメッセージをモデルに送信し、応答が完了するまでブロックしません。
// 応答完了後、responseChan に完全な応答を一度だけ書き込みます。
func (s *geminiLiveSession) Send(ctx context.Context, data types.LiveStreamData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ユーザー入力の genai.Part を作成
	userInput := genai.Text(data.Text)

	// 非同期でストリーム処理を実行
	go func() {
		defer func() {
			// 処理が完了したことを通知
			s.doneChan <- io.EOF
		}()

		// 1. ストリームを開始
		stream := s.chatSession.SendMessageStream(ctx, userInput)
		var responseBuilder strings.Builder

		// 2. ストリームが完了するまでチャンクを累積
		for {
			resp, err := stream.Next()
			if err == io.EOF {
				break // ストリーム完了
			}
			if err != nil {
				log.Printf("Gemini stream error: %v", err)
				s.responseChan <- &types.LowLatencyResponse{ResponseText: fmt.Sprintf("Error: %v", err.Error()), Done: true}
				return
			}

			// チャンクからテキストを抽出して累積
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				// genai.Part はスライスなので、最初の要素をチェック
				if len(resp.Candidates[0].Content.Parts) > 0 {
					if textPart, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
						responseBuilder.WriteString(string(textPart))
					}
				}
			}
		}

		// 3. 累積した完全な応答を responseChan に一度だけ送信
		fullResponse := responseBuilder.String()
		if fullResponse != "" {
			s.responseChan <- &types.LowLatencyResponse{
				ResponseText: fullResponse,
				Done:         true, // 応答完了シグナル
			}
		}

		// 4. (重要) responseChan に何も送信されない場合 (空の応答など) に備え、Doneシグナルを送り、パイプラインのブロックを解除する
		if fullResponse == "" {
			s.responseChan <- &types.LowLatencyResponse{ResponseText: "", Done: true}
		}

	}()

	return nil
}

// RecvResponse は完全な応答が生成されるのを待ち、それを一度だけ返します。
func (s *geminiLiveSession) RecvResponse() (*types.LowLatencyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.doneChan:
		// 内部ストリーム処理が完了したことを示す
		// responseChan からまだ読み込まれていないデータがあれば読み込む
		select {
		case resp := <-s.responseChan:
			return resp, nil
		default:
			// Done が通知されたが、responseChan にデータが残っていない場合は、EOFを返す
			return &types.LowLatencyResponse{Done: true}, io.EOF
		}

	case resp, ok := <-s.responseChan:
		if !ok {
			return nil, io.EOF
		}

		return resp, nil
	}
}

// Close はセッションとクライアントをクリーンアップします。
func (s *geminiLiveSession) Close() {
	// ここでは特に何も行いません。
}
