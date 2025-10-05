package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"prompter-live-go/internal/apis"
	"prompter-live-go/internal/services/live_processor"
	"prompter-live-go/internal/util"

	"github.com/spf13/cobra"
)

// runFlags は run コマンドのフラグを保持するための構造体です。
var runFlags struct {
	channelID       string
	pollingInterval time.Duration
	promptFile      string
	dryRun          bool
}

// runCmd は AI自動応答サービスを開始するためのコマンドです。
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "YouTubeライブコメント監視とAI自動応答サービスを開始します。",
	Long:  `指定されたチャンネルのコメントを定期的にポーリングし、Gemini AIが生成した応答を自動で投稿します。`,
	RunE:  runRunE,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVar(&runFlags.channelID, "channel-id", "", "監視対象のYouTubeチャンネルID (必須)")
	runCmd.MarkFlagRequired("channel-id")

	runCmd.Flags().DurationVar(&runFlags.pollingInterval, "polling-interval", 30*time.Second, "コメントをチェックする間隔 (例: 15s, 30s)")
	runCmd.Flags().StringVar(&runFlags.promptFile, "prompt-file", "", "AIのキャラクター設定と応答指示が書かれたプロンプトファイルのパス (必須)")
	runCmd.MarkFlagRequired("prompt-file")

	runCmd.Flags().BoolVar(&runFlags.dryRun, "dry-run", false, "実際のコメント投稿をスキップし、応答結果のみを表示する (テスト用)")
}

func runRunE(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 環境設定の表示と検証
	fmt.Println("--- Prompter Live Go: 自動応答サービス開始 ---")
	fmt.Printf("✅ チャンネルID: %s\n", runFlags.channelID)
	fmt.Printf("✅ ポーリング間隔: %s\n", runFlags.pollingInterval)
	fmt.Printf("✅ プロンプトファイル: %s\n", runFlags.promptFile)
	if runFlags.dryRun {
		fmt.Println("⚠️ ドライランモード: コメントは投稿されず、応答結果のみ表示されます。")
	}

	// 2. クライアントの初期化

	// Geminiクライアントの初期化
	prompt, err := util.LoadPromptFile(runFlags.promptFile)
	if err != nil {
		return fmt.Errorf("プロンプトファイルの読み込みに失敗: %w", err)
	}
	geminiClient, err := apis.NewGeminiClient(ctx, prompt)
	if err != nil {
		return fmt.Errorf("Gemini クライアントの初期化に失敗: %w", err)
	}
	slog.Info("Gemini API クライアントが正常に初期化されました。")

	// YouTubeクライアントの初期化
	youtubeClient, err := apis.NewYouTubeClient(ctx, runFlags.channelID)
	if err != nil {
		return fmt.Errorf("YouTube クライアントの初期化に失敗: %w", err)
	}

	// 3. プロセッサの初期化 (ビジネスロジックの注入)
	processor := live_processor.NewProcessor(youtubeClient, geminiClient, runFlags.dryRun)

	// 4. ポーリングの開始とループ

	slog.Info("📢 ポーリングを開始します。", "間隔", runFlags.pollingInterval)

	// OSシグナルハンドリング (Ctrl+Cなどで終了できるように)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// メインのポーリングループ
	ticker := time.NewTicker(runFlags.pollingInterval)
	defer ticker.Stop()

	// 初回ポーリング
	// 修正済み: apis.FetchAndProcessComments の代わりに processor.ProcessNextBatch を呼び出します。
	if err := processor.ProcessNextBatch(ctx); err != nil {
		slog.Warn("サービス起動時の初回ポーリングエラー", "error", err)
	}

	for {
		select {
		case <-ticker.C:
			// 定期的なポーリング
			// 修正済み: apis.FetchAndProcessComments の代わりに processor.ProcessNextBatch を呼び出します。
			if err := processor.ProcessNextBatch(ctx); err != nil {
				slog.Error("ポーリングエラー", "error", err)
			}
		case sig := <-sigCh:
			// 終了シグナル受信
			slog.Info("サービスを終了します", "signal", sig.String())
			return nil
		case <-ctx.Done():
			// コンテキストキャンセルによる終了
			slog.Info("サービスがキャンセルされました")
			return nil
		}
	}
}
