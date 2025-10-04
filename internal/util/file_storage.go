package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// SaveToken はトークン構造体（通常はoauth2.Token）をJSON形式で指定されたパスに保存します。
func SaveToken(path string, token interface{}) error {
	// ファイルが配置されるディレクトリが存在しない場合、作成する
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("トークンのJSONエンコードに失敗: %w", err)
	}

	// トークンをファイルに書き込む (パーミッション 0600 でセキュアに)
	if err := ioutil.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("トークンのファイル保存に失敗: %w", err)
	}
	return nil
}

// LoadToken は指定されたパスからトークン（通常はoauth2.Token）を読み込みます。
// tokenPtr には、トークンを受け取る構造体へのポインタを渡します。
func LoadToken(path string, tokenPtr interface{}) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		// ファイルが存在しない場合は、認証が必要であることを示すエラーとする
		return fmt.Errorf("トークンファイルの読み込みに失敗 (認証が必要かもしれません): %w", err)
	}

	if err := json.Unmarshal(data, tokenPtr); err != nil {
		return fmt.Errorf("トークンのJSONデコードに失敗: %w", err)
	}
	return nil
}
