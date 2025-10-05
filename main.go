package main

import (
	"prompter-live-go/cmd"
)

func main() {
	// cmd.Execute() は内部でエラーハンドリングを行い、
	// エラー時には os.Exit(1) でプログラムを終了させます。
	cmd.Execute()
}
