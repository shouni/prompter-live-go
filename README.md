# 🤖 Prompter Live Go

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-reviewer-go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - エンゲージメントを掴み、サイト誘導を加速するAIプロモーションパートナー

**`Prompter Live Go`** は、**Google Gemini の強力な AI**と**Go言語の並行処理能力**、そして **Gemini Live API** の低遅延性能を活用し、YouTube配信における**リアルタイムな視聴者の発言を会話に溶け込む形でプロモーション導線**に変えるコマンドラインツールです。

従来の受動的な広告と異なり、本ツールは視聴者との会話に溶け込む形で、AIが**キャラクター設定を厳守**しつつ、**会話の流れに自然に溶け込む形**で自社サイトやキャンペーン情報をコメント欄に提供します。

これにより、視聴者は「宣伝」と感じることなく自然に情報に触れ、**会話の流れでサイトへ誘導**されます。単なるファン対応の効率化に留まらず、YouTubeという大規模なプラットフォームにおける**プロモーション効果とコンバージョン**を飛躍的に高める、戦略的な AI ソリューションです。

-----

### 🌸 導入がもたらすポジティブな変化

| メリット | チーム・配信者への影響 | 期待される効果 | 
 | :--- | :--- | :--- | 
| **リアルタイムなファン対応** | コメントへの即時応答で、ユーザー体験が向上します。特に**Live API**により応答速度が飛躍的に向上します。 | **エンゲージメント**が高まり、動画やチャンネルへの再訪問率が向上します。 | 
| **プロモーション導線の自動構築** | AIが会話の流れを読み取り、**自社サイトへの誘導リンク**やキャンペーン情報を自然にコメントに組み込みます。 | 従来の広告と異なり、**会話に溶け込んだ形でコンバージョン**を促し、サイト誘導率を向上させます。 | 
| **キャラクターの一貫性維持** | AIが厳密なプロンプトに従い、常にブランド設定を守って応答します。 | ブランドイメージを毀損することなく、**信頼できる情報源**としてサイトへの信頼感を高めます。 | 
| **データに基づいた効果測定** | どの応答やプロモーションコメントが反応を得たかを分析できます。 | **YouTubeプロモーション戦略のPDCAサイクル**を回すための貴重なデータを自動で収集できます。 | 

-----

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ | 役割 | 
 | :--- | :--- | :--- | 
| **言語** | **Go (Golang)** | ツールの開発言語。リアルタイム応答に必要な高い並行処理性能を提供します。 | 
| **CLI フレームワーク** | **Cobra** | コマンドライン引数や認証フローを管理するための構造化を提供します。 | 
| **AI モデル** | **Google Gemini Live API** | 視聴者のコメント分析、キャラクター設定に基づいた応答テキストの**リアルタイムストリーミング生成**に使用します。 |
| **YouTube 連携** | **Google OAuth 2.0 / YouTube Data API v3** | チャンネル所有者としての認証（OAuth）、アクティブなライブチャット ID の取得、**ライブチャットのポーリング**、AI応答コメントのポストに使用します。
| **リアルタイム処理** | **Go Goroutine & Channel** | コメントの取得、AI処理、APIポストを並行して実行し、低遅延での応答を実現します。 | 

-----

## 🛠️ 事前準備と環境設定

### 1\. Go のインストールとビルド

```bash
# リポジトリをクローン
git clone git@github.com:shouni/prompter-live-go.git
cd prompter-live-go

# 依存関係を整理 (Live API への移行に伴うクリーンアップ)
go mod tidy

# 実行ファイルを bin/ ディレクトリに生成
go build -o bin/prompter_live
````

### 2\. 環境変数の設定 (必須)

Goアプリケーションは、以下の変数名で認証情報を読み込みます。

#### macOS / Linux (bash/zsh)

```bash
# 💡 注意: YouTube のクライアントID/シークレットは 'YT_' で始まります
export GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
export YT_CLIENT_ID="YOUR_GCP_CLIENT_ID"
export YT_CLIENT_SECRET="YOUR_GCP_CLIENT_SECRET"
```

#### Windows (PowerShell)

```powershell
# 💡 注意: YouTube のクライアントID/シークレットは 'YT_' で始まります
$env:GEMINI_API_KEY="YOUR_GEMINI_API_KEY"
$env:YT_CLIENT_ID="YOUR_GCP_CLIENT_ID"
$env:YT_CLIENT_SECRET="YOUR_GCP_CLIENT_SECRET"
```

### 3\. Google Cloud Platform (GCP) の設定 (重要)

本ツールを実行するには、GCPプロジェクトで以下の設定が完了している必要があります。

1.  **YouTube Data API v3 の有効化**:

    - GCPコンソールで、使用するプロジェクトの **「YouTube Data API v3」** を有効化してください。

2.  **OAuth リダイレクト URI の登録**:

    - OAuth 2.0 クライアント ID（ウェブ アプリケーション）の設定で、以下のコールバック URI を**承認済みリダイレクト URI** に追加してください。
        - `http://localhost:8080/callback`
        - `http://localhost:8081/callback` (ポート競合時の予備)

### 4\. プロンプトファイルの準備（現在の実装ではコマンドラインで指定）

AI のキャラクター設定はコマンドライン引数 (`-i` / `--instruction`) で直接渡す方式です。長い設定はMarkdownファイルなどに記述し、引数として渡すことを推奨します。

-----

## 🚀 使い方 (Usage) と実行例

### 1\. 認証コマンド (`auth`) 🔒

本ツールを最初に実行する際、YouTubeへのコメント投稿権限を得るための**OAuth 2.0 認証**を行います。

```bash
# 標準ポートで認証を開始
./bin/prompter_live auth

# ポート競合が発生した場合
./bin/prompter_live auth --oauth-port 8082
```

> **Note:** 認証成功後、プロジェクトルートに `config/token.json` ファイルが生成されます。

### 2\. 自動応答開始コマンド (`run`) 🤖

認証が完了したら、Gemini Live API と YouTube Live Chat への接続を確立し、自動応答を開始します。

このコマンドは、**YouTubeチャンネルID**と**Gemini APIキー**を必須とします。

```bash
# Live APIに接続し、AIによる自動応答とコメント投稿を開始
./bin/prompter_live run \
  -k "YOUR_GEMINI_API_KEY" \
  -c "UCxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  -m "gemini-2.5-flash" \
  -i "あなたは視聴者コメントを代行するAIです。フレンドリーかつ簡潔に返答してください。"
```

#### 固有フラグ

| フラグ | 説明 | デフォルト値 |
| :--- | :--- | :--- |
| `-k`, `--api-key` | **Gemini API Key** | `GEMINI_API_KEY` 環境変数 |
| `-c`, `--youtube-channel-id` | **監視対象の YouTube チャンネル ID (必須)** | **なし** |
| `-m`, `--model` | 使用する Gemini モデル名（Live API対応モデル推奨） | `gemini-2.5-flash` |
| `-i`, `--instruction` | AIの応答ルールやキャラクター設定（System Instruction） | **なし** |
| `-r`, `--modalities` | AIからの応答として期待するデータ形式 | `TEXT` |

> **重要**: `run` コマンドは、指定されたチャンネルが**現在アクティブなライブ配信を行っている場合のみ** Live Chat ID を取得し、コメントの投稿が可能です。

-----

### ⚠️ 動作原理に関する重要な注意点

* **Gemini Live API**: 非常に低遅延なリアルタイム対話のための機能です。

* **オーディオ入力**: 現在のパイプラインは**ダミーのオーディオデータ**を送信することで Gemini Live API との接続をテストしています。実際の利用には、マイク入力や音声処理ロジックの統合が必要です。

* **YouTube API クォータ**: YouTube Live Chat ID の取得やコメントの投稿は API クォータを消費します。

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
