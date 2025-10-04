package main

import (
	"log"
	"prompter-live-go/cmd"
)

func main() {
	// エラーが発生した場合、ログに出力してプログラムを終了する
	if err := cmd.Execute(); err != nil {
		// CobraのExecute()は通常、エラーを内部で処理するため、このログは致命的なエラーに限定されます
		log.Fatalf("CLI tool execution failed: %v", err)
	}
}
