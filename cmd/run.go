package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"prompter-live-go/internal/apis"
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

// runCmd は run コマンドを定義します。
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "指定されたチャンネルのコメントを監視し、AIによる自動応答を開始します。",
	RunE:  runE, // エラーを返す RunE を使用
}

func init() {
	// ここで rootCmd に runCmd を追加します。（rootCmdはcmd/root.goで定義）
	rootCmd.AddCommand(runCmd)

	// フラグの定義
	runCmd.Flags().StringVarP(&runFlags.channelID, "channel-id", "c", "", "監視対象の YouTube チャンネル ID (必須)")
	runCmd.MarkFlagRequired("channel-id")

	runCmd.Flags().DurationVarP(&runFlags.pollingInterval, "polling-interval", "i", 30*time.Second, "コメントをチェックする間隔 (例: 15s, 1m)")
	runCmd.Flags().StringVarP(&runFlags.promptFile, "prompt-file", "p", "", "キャラクター設定と応答指示が書かれたプロンプトファイルのパス (必須)")
	runCmd.MarkFlagRequired("prompt-file")

	runCmd.Flags().BoolVar(&runFlags.dryRun, "dry-run", false, "コメント投稿をスキップし、応答結果のみを表示するテストモード")
}

// runE は 'run' コマンドの実行ロジックです。
func runE(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("--- Prompter Live Go: 自動応答サービス開始 ---")
	fmt.Printf("✅ チャンネルID: %s\n", runFlags.channelID)
	fmt.Printf("✅ ポーリング間隔: %s\n", runFlags.pollingInterval)
	fmt.Printf("✅ プロンプトファイル: %s\n", runFlags.promptFile)
	if runFlags.dryRun {
		fmt.Println("⚠️ ドライランモード: コメントは投稿されず、応答結果のみ表示されます。")
	}

	// --- 1. 認証情報の読み込み ---
	// util.GetOAuth2Config() は internal/util/util.go で定義されています
	oauthConfig := util.GetOAuth2Config()
	token, err := util.LoadToken(util.TokenPath)
	if err != nil {
		return fmt.Errorf("認証トークンの読み込みに失敗: %w\n'./bin/prompter_live auth' で認証を行ってください", err)
	}

	// --- 2. クライアントの初期化 ---
	// YouTube クライアント
	ytClient, err := apis.NewYouTubeClient(ctx, oauthConfig, token)
	if err != nil {
		return fmt.Errorf("YouTube クライアントの初期化に失敗: %w", err)
	}

	// Gemini クライアント (APIキーは環境変数から取得されます)
	gc, err := apis.NewGeminiClient(ctx, runFlags.promptFile)
	if err != nil {
		return fmt.Errorf("Gemini クライアントの初期化に失敗: %w", err)
	}

	// --- 3. 終了シグナルハンドリング ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("\n⚠️ 終了シグナルを受信しました。サービスを停止しています...")
		cancel()
	}()

	// --- 4. ポーリング開始 ---
	return ytClient.StartPolling(
		ctx,
		runPollingCycle,
		gc,
		runFlags.channelID,
		runFlags.pollingInterval,
		runFlags.dryRun,
	)
}

// runPollingCycle は StartPolling から定期的に呼び出される、コメント処理の中核ロジックです。
func runPollingCycle(ctx context.Context, ytClient *apis.YouTubeClient, gc *apis.GeminiClient, channelID string, dryRun bool) error {
	// 1. 最新コメントの取得
	commentThreads, err := ytClient.GetLatestComments(channelID)
	if err != nil {
		return fmt.Errorf("コメント取得エラー: %w", err)
	}

	// 2. コメントスレッドを一つずつ処理
	for _, thread := range commentThreads {
		// 基本的なコメント情報
		commentSnippet := thread.Snippet.TopLevelComment.Snippet
		commentID := thread.Snippet.TopLevelComment.Id
		author := commentSnippet.AuthorDisplayName
		text := commentSnippet.TextOriginal

		fmt.Printf("\n[NEW COMMENT] ID: %s | Author: %s | Text: %s\n", commentID, author, text)

		// 3. AI応答の生成
		aiResponse, err := gc.GenerateResponse(ctx, text, author)
		if err != nil {
			fmt.Printf("❌ AI応答生成エラー (ID: %s): %v\n", commentID, err)
			continue
		}

		fmt.Printf("🤖 AI応答生成完了:\n%s\n", aiResponse)

		// 4. コメントの投稿（Dry Run チェック）
		if !dryRun {
			// 実際のコメント投稿
			_, err := ytClient.PostReply(commentID, aiResponse)
			if err != nil {
				fmt.Printf("❌ コメント投稿エラー (ID: %s): %v\n", commentID, err)
			}
		} else {
			fmt.Println("➡️ ドライランモードのため、投稿はスキップされました。")
		}
	}

	return nil
}
