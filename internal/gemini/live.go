package gemini

import (
	"context"
	"io"
	"log"
	"sync"
	// "fmt" は未使用のため削除
	// "time" は未使用のため削除

	"prompter-live-go/internal/types"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
)

// geminiLiveSession は Gemini Live API との対話セッションを管理します。
type geminiLiveSession struct {
	chatSession *genai.ChatSession
	// 💡 修正: currentResponse を統合されたコンテンツを保持する []genai.Part に変更するか、
	// 		  応答全体を保持したまま、内容を適切に扱うようにロジックを修正します。
	// 		  ここでは簡略化のため、ストリーム中に蓄積されるテキスト全体を保持します。
	currentText string                              // ストリーム中に蓄積される応答テキスト
	streamChan  chan *genai.GenerateContentResponse // ストリーム応答を送信するチャネル
	doneChan    chan error                          // ストリーム終了またはエラーを通知するチャネル
	mu          sync.Mutex
}

// newGeminiLiveSession は新しい geminiLiveSession を作成します。
func newGeminiLiveSession(model *genai.GenerativeModel, config types.LiveAPIConfig) *geminiLiveSession {
	// システム指示がある場合は、ChatSession の履歴に先行する Content として設定できますが、
	// Live Chat のユースケースでは通常、モデル設定として渡されます。
	// ここでは単純に ChatSession を開始します。
	chatSession := model.StartChat()

	return &geminiLiveSession{
		chatSession: chatSession,
		streamChan:  make(chan *genai.GenerateContentResponse),
		doneChan:    make(chan error, 1),
	}
}

// Send はメッセージをモデルに送信し、ストリーミングを開始します。
func (s *geminiLiveSession) Send(ctx context.Context, data types.LiveStreamData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 既存のストリームがまだ終了していない場合、先に終了シグナルを送る
	if s.streamChan != nil {
		select {
		case s.doneChan <- io.EOF: // 処理完了シグナル
		default:
		}
		close(s.streamChan)
	}

	// 新しいストリームセッションの初期化
	s.streamChan = make(chan *genai.GenerateContentResponse)
	s.currentText = "" // 💡 修正: 応答テキストをリセット

	userInput := genai.Text(data.Text)

	log.Printf("Gemini sending: %s", data.Text)

	// 非同期でストリーミングを実行するゴルーチンを開始
	go func() {
		defer func() {
			s.doneChan <- io.EOF
			close(s.streamChan)
			log.Println("Gemini stream finished.")
		}()

		stream := s.chatSession.SendMessageStream(ctx, userInput)

		for {
			resp, err := stream.Next()
			if err == iterator.Done {
				return
			}
			if err != nil {
				log.Printf("Gemini stream error: %v", err)
				s.doneChan <- err
				return
			}
			s.streamChan <- resp
		}
	}()

	return nil
}

// RecvResponse はストリームから次の応答チャンクを受け取ります。
func (s *geminiLiveSession) RecvResponse() (*types.LowLatencyResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case err := <-s.doneChan:
		if err != nil && err != io.EOF {
			return nil, err
		}
		return &types.LowLatencyResponse{Done: true}, nil

	case resp, ok := <-s.streamChan:
		if !ok {
			return &types.LowLatencyResponse{Done: true}, nil
		}

		// 応答チャンクからテキストを取得
		chunkText := ""
		if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
			// ストリーム応答からテキストチャンクを取得し、統合
			for _, part := range resp.Candidates[0].Content.Parts {
				if textPart, ok := part.(genai.Text); ok {
					chunkText += string(textPart)
				}
			}
		}

		// 💡 修正: currentText に追記
		s.currentText += chunkText

		// 応答をタイプ変換して返す
		return &types.LowLatencyResponse{
			ResponseText: chunkText,
			Done:         false,
		}, nil
	}
}

// Close はセッションを閉じ、リソースを解放します。
func (s *geminiLiveSession) Close() {
	// ChatSession は明示的なクローズメソッドがないため、特に処理は不要です。
	// チャネルのクローズは Send/RecvResponse のロジックで安全に実行されています。
}
