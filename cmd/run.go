package cmd

import (
	"fmt"
	"os" // os パッケージをインポート
	"time"

	"github.com/spf13/cobra"
)

// ... runFlags 構造体は省略 ...

var runCmd = &cobra.Command{
	// ... Use, Short, Long は省略 ...
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("--- Prompter Live Go: 自動応答サービス開始 ---")

		// 必須フラグのチェック
		if runFlags.channelID == "" || runFlags.promptFile == "" {
			fmt.Println("\n❌ エラー: --channel-id と --prompt-file は必須です。")
			cmd.Usage()
			os.Exit(1) // ⭐️ 【修正箇所】ここでプログラムを終了
			return     // os.Exit(1) で終了するため return は技術的には不要だが残しても問題なし
		}
		// ... 以降のコードは変更なし ...
	},
}

// ... init 関数は省略 ...
