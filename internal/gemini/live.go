package gemini

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"prompter-live-go/internal/types"

	"google.golang.org/genai"
)

// Session は Gemini Live API とのセッションインターフェースを定義します。
type Session interface {
	Send(ctx context.Context, data types.LiveStreamData) error
	RecvResponse() (*types.LowLatencyResponse, error)
	Close()
}

// geminiLiveSession は Session インターフェースの実装です。
type geminiLiveSession struct {
	mu         sync.Mutex
	baseClient *genai.Client
	modelName  string
	config     types.LiveAPIConfig

	// chatSession は *genai.Chat 型（GenerativeModel.StartChat の戻り値）
	chatSession *genai.Chat

	responseChan chan *types.LowLatencyResponse
	doneChan     chan struct{}
}

// newGeminiLiveSession は新しい geminiLiveSession を作成します。
func newGeminiLiveSession(client *genai.Client, modelName string, config types.LiveAPIConfig, systemInstruction string) Session {
	log.Printf("Internal Session created - Model: %s, Instruction: %s", modelName, systemInstruction)

	// 生成モデルの取得（NewGenerativeModel を使用）
	model := genai.NewGenerativeModel(client, modelName)

	// 履歴は []*genai.Content
	var history []*genai.Content
	if systemInstruction != "" {
		// genai.Part の値
		userPart := genai.Text(systemInstruction)
		modelPart := genai.Text("Ok, I understand.")

		// Content.Parts は []genai.Part（値）
		userContent := genai.Content{
			Parts: []genai.Part{userPart},
			Role:  "user",
		}
		modelContent := genai.Content{
			Parts: []genai.Part{modelPart},
			Role:  "model",
		}

		history = append(history, &userContent, &modelContent)
	}

	// StartChat の後に履歴を設定
	chatSession := model.StartChat()
	chatSession.History = history

	return &geminiLiveSession{
		baseClient:   client,
		modelName:    modelName,
		config:       config,
		chatSession:  chatSession,
		responseChan: make(chan *types.LowLatencyResponse, 10),
		doneChan:     make(chan struct{}),
	}
}

// Send はデータをAIに送信し、応答処理を開始します。
func (s *geminiLiveSession) Send(ctx context.Context, data types.LiveStreamData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.doneChan:
		return errors.New("session already closed")
	default:
	}

	if s.chatSession == nil {
		return errors.New("chat session is not initialized")
	}

	log.Printf("Sending data to Gemini - Author: %s, Text: %v", data.Author, data.Text)

	// genai.Part の値を作成
	userInput := genai.Text(data.Text)

	// 非同期でストリーム処理を実行
	go func() {
		defer func() {
			close(s.responseChan)
			select {
			case <-s.doneChan:
			default:
				close(s.doneChan)
			}
		}()

		// 可変長引数で genai.Part を渡す
		stream := s.chatSession.SendMessageStream(ctx, userInput)
		var responseBuilder strings.Builder

		for resp, err := range stream {
			if err != nil {
				log.Printf("Gemini stream error: %v", err)
				select {
				case s.responseChan <- &types.LowLatencyResponse{ResponseText: fmt.Sprintf("Error: %v", err.Error()), Done: true}:
				case <-ctx.Done():
				}
				return
			}

			// 応答のテキスト抽出（genai.Part は interface なので型スイッチ）
			if len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil {
				for _, part := range resp.Candidates[0].Content.Parts {
					switch p := part.(type) {
					case genai.Text:
						responseBuilder.WriteString(string(p))
					default:
						// 他種別は無視（必要に応じて拡張）
					}
				}
			}
		}

		select {
		case s.responseChan <- &types.LowLatencyResponse{ResponseText: responseBuilder.String(), Done: true}:
		case <-ctx.Done():
		}
	}()

	return nil
}

// RecvResponse は完全な応答が来るまで待ち受け、それを返します。
func (s *geminiLiveSession) RecvResponse() (*types.LowLatencyResponse, error) {
	select {
	case resp, ok := <-s.responseChan:
		if !ok {
			return nil, io.EOF
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
		return
	default:
		close(s.doneChan)
		log.Println("Gemini Live Session closed.")
	}
}
