package types

import "time"

// LiveAPIConfig は Gemini Live API の設定を保持します。
// NewClient (internal/gemini/client.go) で初期化時に使用されます。
type LiveAPIConfig struct {
	ModelName         string
	SystemInstruction string
}

// LiveStreamData は Live Chat からの入力データ構造体です。
// LiveSession.Send() メソッドの引数として使用され、ユーザーメッセージを運びます。
type LiveStreamData struct {
	Text      string
	Timestamp string
	Author    string
	// AudioData []byte // 将来のオーディオ入力用
}

// LowLatencyResponse は Live API からのストリーム応答チャンクです。
// LiveSession.RecvResponse() メソッドの戻り値として使用されます。
type LowLatencyResponse struct {
	ResponseText string
	Done         bool // ストリームの終了を示すフラグ
}

// PipelineConfig はパイプライン動作のための設定を保持します。
type PipelineConfig struct {
	PollingInterval time.Duration
}
