package main

import (
	"log"

	"prompter-live-go/cmd" // ★ このパスは、go mod init で指定したモジュール名に依存します
)

func main() {
	// ログの形式を改善し、タイムスタンプを付加
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
