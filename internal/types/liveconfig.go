// internal/types/liveconfig.go
package types

import "time"

// LiveAPIConfig は Gemini Live API の接続とセッション設定を保持します。
type LiveAPIConfig struct {
	// Gemini APIキー (認証に使用)
	APIKey string

	// Live APIで使用するモデル名
	Model string

	// 応答のキャラクター設定や指示を記述したプロンプト
	SystemInstruction string

	// 応答の出力形式 ("TEXT" や "AUDIO") を文字列スライスで定義します。
	// SDKの型ではなく、標準の文字列で扱うことで、このファイルからSDKへの依存を排除します。
	ResponseModalities []string

	// ツール(Function Calling)の定義は、一旦 nil や空のインターフェースで保持します。
	// Liveクライアント内でSDKの型に変換します。
	Tools interface{}
}

// LiveStreamData は Live API に送信するマルチモーダルデータ（音声または映像）の形式を定義します。
type LiveStreamData struct {
	// データの種類 (audio/pcm;rate=16000, image/jpeg など)
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

// PipelineConfig はパイプライン動作のための設定（ポーリング間隔など）を保持します。
type PipelineConfig struct {
	// YouTube Live Chatのポーリング間隔 (cmd/run.goで設定される)
	PollingInterval time.Duration
}
