package types

import "google.golang.org/genai/types"

// LiveAPIConfig は Gemini Live API の接続とセッション設定を保持します。
type LiveAPIConfig struct {
	// Gemini APIキー (認証に使用)
	APIKey string

	// Live APIで使用するモデル名 (例: gemini-live-2.5-flash)
	Model string

	// 応答のキャラクター設定や指示を記述したプロンプト (SystemInstruction)
	// liveconfig.go 側で文字列として保持
	SystemInstruction string

	// Live APIで受け取りたい出力形式 (例: TEXT, AUDIO)
	// genai/types.ResponseModality のスライス
	ResponseModalities []types.ResponseModality

	// ツール(Function Calling)の定義。現在は空でOKですが、将来的な拡張に備えます。
	Tools []*types.Tool
}

// LiveStreamData は Live API に送信するマルチモーダルデータ（音声または映像）の形式を定義します。
type LiveStreamData struct {
	// データの種類 (audio, video, text)
	MimeType string

	// データの生バイト列
	Data []byte
}

// LowLatencyResponse は Live API から得られる応答を簡潔に定義します。
type LowLatencyResponse struct {
	// AIが生成したテキスト（コメント投稿に使用）
	Text string

	// 応答が完了したかどうか
	Done bool
}
