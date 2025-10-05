# 🤖 Prompter Live Go

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/git-gemini-reviewer-go)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 🚀 概要 (About) - エンゲージメントを掴み、サイト誘導を加速するAIプロモーションパートナー

**`Prompter Live Go`** は、**Google Gemini の強力なAI**と**Go言語の並行処理能力**を活用し、YouTube配信や動画のコメント欄を**生きたプロモーション導線**に変えるコマンドラインツールです。

従来の受動的な広告（Xなど）と異なり、本ツールは視聴者との会話に溶け込む形で、AIが**キャラクター設定を厳守**しつつ、**最も効果的なタイミング**で自社サイトやキャンペーン情報をコメント欄に提供します。

これにより、視聴者は「宣伝」と感じることなく自然に情報に触れ、**会話の流れでサイトへ誘導**されます。単なるファン対応の効率化に留まらず、YouTubeという大規模なプラットフォームにおける**プロモーション効果とコンバージョン**を飛躍的に高める、戦略的な AI ソリューションです。

-----

### 🌸 導入がもたらすポジティブな変化

| メリット | チーム・配信者への影響 | 期待される効果 |
| :--- | :--- | :--- |
| **リアルタイムなファン対応** | コメントへの即時応答で、ユーザー体験が向上します。 | **エンゲージメント**が高まり、動画やチャンネルへの再訪問率が向上します。 |
| **プロモーション導線の自動構築** | AIが会話の流れを読み取り、**自社サイトへの誘導リンク**やキャンペーン情報を自然にコメントに組み込みます。 | 従来の広告と異なり、**会話に溶け込んだ形でコンバージョン**を促し、サイト誘導率を向上させます。 |
| **キャラクターの一貫性維持** | AIが厳密なプロンプトに従い、常にブランド設定を守って応答します。 | ブランドイメージを毀損することなく、**信頼できる情報源**としてサイトへの信頼感を高めます。 |
| **データに基づいた効果測定** | どの応答やプロモーションコメントが反応を得たかを分析できます。 | **YouTubeプロモーション戦略のPDCAサイクル**を回すための貴重なデータを自動で収集できます。 |

-----

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ | 役割 |
| :--- | :--- | :--- |
| **言語** | **Go (Golang)** | ツールの開発言語。リアルタイム応答に必要な高い並行処理性能を提供します。 |
| **CLI フレームワーク** | **Cobra** | コマンドライン引数や認証フローを管理するための構造化を提供します。 |
| **AI モデル** | **Google Gemini API** | 視聴者のコメント分析、キャラクター設定に基づいた応答テキストのリアルタイム生成に使用します。 |
| **YouTube 連携** | **Google OAuth 2.0 / YouTube Data API v3** | チャンネル所有者としての認証（OAuth）フロー、コメントのポーリング、応答コメントのポストに使用します。 |
| **リアルタイム処理** | **Go Goroutine & Channel** | コメントのポーリング、AI処理、APIポストを並行して実行し、低遅延での応答を実現します。 |

-----

## 🛠️ 事前準備と環境設定

### 1\. Go のインストールとビルド

```bash
# リポジトリをクローン
git clone git@github.com:shouni/prompter-live-go.git
cd prompter-live-go

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
    - GCPコンソールで、使用するプロジェクトの **「YouTube Data API v3」** を有効化してください。これが無効だと `Error 403: SERVICE_DISABLED` が発生します。
2.  **OAuth リダイレクト URI の登録**:
    - OAuth 2.0 クライアント ID（ウェブ アプリケーション）の設定で、以下のコールバック URI を**承認済みリダイレクト URI** に追加してください。
        - `http://localhost:8080/callback`
        - `http://localhost:8081/callback` (ポート競合時の予備)

### 4\. プロンプトファイルの準備

応答の核となるキャラクター設定と応答ルールをMarkdownファイルに記述します。

```
prompter-live-go/
└── prompts/
    └── character_setting.md  # 必須。このファイルに応答ルールとキャラ設定を記述します。
```

-----

## 🚀 使い方 (Usage) と実行例

### 1\. 認証コマンド (`auth`) 🔒

本ツールを最初に実行する際、YouTubeへのコメント投稿権限を得るための**OAuth 2.0 認証**を行います。

認証に失敗する場合、`--oauth-port` フラグを使用して、空いているポートに変更できます（例: 8082）。

```bash
# 標準ポートで認証を開始
./bin/prompter_live auth

# ポート競合が発生した場合 (推奨: 8082など空きポートを使用)
./bin/prompter_live auth --oauth-port 8082
```

> **Note:** 認証成功後、プロジェクトルートに `config/token.json` ファイルが生成されます。

### 2\. 自動応答開始コマンド (`run`) 🤖

認証が完了したら、指定した YouTube チャンネルのコメントを監視し、AIによる自動応答を開始します。

```bash
# 特定のチャンネルIDの動画/コメントをポーリングして応答を開始
./bin/prompter_live run \
  --channel-id "UCxxxxxxxxxxxxxxxxxxxxxxxxxx" \
  --polling-interval 15s \
  --prompt-file "prompts/character_setting.md"
```

#### 固有フラグ

| フラグ | 説明 | デフォルト値 |
| :--- | :--- | :--- |
| `--channel-id` | 監視対象の YouTube チャンネル ID | **なし (必須)** |
| `--polling-interval` | コメントをチェックする間隔（秒）。短くするとリアルタイム性が増しますが、API利用制限に注意。 | `30s` |
| `--prompt-file` | キャラクター設定と応答指示が書かれたプロンプトファイルのパス | **なし (必須)** |
| `--dry-run` | 実際のコメント投稿をスキップし、応答結果を標準出力する（テスト用） | `false` |

> **重要**: `run` コマンドは、指定されたチャンネルが**現在アクティブなライブ配信を行っている場合のみ**コメントのポーリングを開始できます。ライブ中でない場合、`videoNotFound` エラーが発生します。

-----

### ⚠️ 動作原理に関する重要な注意点

* **ポーリング（定期取得）を採用**: YouTube API の仕様により、本ツールは指定された間隔（`--polling-interval`）で API へ問い合わせを行う「**ポーリング**」方式を採用しています。
* **API クォータの消費**: ポーリングのたびに YouTube API のクォータ（利用制限）を消費します。頻度を高く設定しすぎると、クォータを使い切る可能性があるため、適切な間隔を設定してください。

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
