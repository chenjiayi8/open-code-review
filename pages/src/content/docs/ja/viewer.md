---
title: セッションビューア
sidebar:
  order: 10
---

`ocr viewer` は小型の組み込み HTTP サーバーで、過去のレビューセッションをブラウザで扱いやすい UI でレンダリングします。
外部依存はありません——セッションは、OCR が各レビュー中にディスクに書き込む JSONL ファイルから直接読み取られます。

## 起動

```bash
ocr viewer                  # binds localhost:5483
ocr viewer --addr :3000     # bind to all interfaces on port 3000
ocr viewer --addr 0.0.0.0:8080   # bind on all interfaces
```

デフォルトのアドレスは `localhost:5483` です。サーバーはフォアグラウンドで実行されます——`Ctrl+C` で停止します。セッションは各リクエスト時に
`~/.opencodereview/sessions/` から遅延スキャンされるため、別のターミナルで実行中のレビューも、その
JSONL ファイルが現れ次第表示されます。

> **DNS リバインディング対策。** ビューアは `Host` ヘッダーをループバックのホワイトリスト
> （`localhost`、`127.0.0.1`、`::1`）と照合します。具体的なバインドホスト
> （`--addr 192.168.1.10:5483` など）は自動的に追加されますが、**ワイルドカード**バインド
> （`:3000`、`0.0.0.0`、`::`）は追加されません——この場合、LAN IP やホスト名から UI にアクセスすると
> `forbidden host` が返されます。ワイルドカードバインドをアクセス可能にするには、
> `OCR_VIEWER_ALLOWED_HOSTS` にカンマ区切りの許可ホスト名リストを設定します
> （例：`OCR_VIEWER_ALLOWED_HOSTS=box.local,192.168.1.10`）。

## 3 つのページ

ビューアには 3 つの URL があります：

| URL | 表示内容 |
|---|---|
| `/` | ディスク上にセッションを持つすべてのリポジトリの一覧。 |
| `/r/{repo}` | 単一リポジトリのセッション一覧。最新が先頭。 |
| `/r/{repo}/{sessionID}` | 単一セッションの完全な詳細。 |

`{repo}` はパスエンコードされた文字列です（区切り文字 `/` と `\` は `-` に、コロンは
`_` に置換されます——ディスク上のディレクトリ命名と同じエンコード）。通常は手動で入力することはなく——クリックして遷移します。

### `/`——リポジトリ一覧

少なくとも 1 件のセッションを持つ各リポジトリについて、リポジトリパス、総セッション数、最新のアクティビティのタイムスタンプを表示します。

### `/r/{repo}`——単一リポジトリのセッション一覧

各セッションについて：ID（UUID）、ブランチ名（OCR が検出できた場合）、レビューモード、モデル、ファイル数、
所要時間、開始タイムスタンプ。

### `/r/{repo}/{sessionID}` — Session detail

The detail page shows the important records for the local runner model:

1. **Header** — diff range, runner, branch, duration, and session id.
2. **Selection** — the review or scan manifest plus filter/exclusion results.
3. **Runner** — invocation metadata and raw structured output from the selected local Codex or Claude runner.
4. **Validation** — JSON-shape, path, line-range, and `reviewed_files` coverage warnings.
5. **Comments** — final comments after OCR validation.

The page is organized around one local runner invocation and the deterministic validation that follows it. When you need exact evidence, the JSONL session is the source of truth.

## ユースケース


ビューアは 3 つのワークフローを想定して設計されています：

### 「なぜモデルはこう言ったのか？」

ターミナル出力であるコメントを開き、ビューアでそのファイルを見つけ、その runner / validation recordsを下にたどります。
**runner invocation**の中に、あなたが気にしている comment を含むカードこそが、それを生み出したラウンドです。カードの
Response にはモデルの推論が表示されます。モデルに送信された prompt + コンテキストを正確に知るには、JSONL トランスクリプトで
そのリクエスト番号の `llm_request` レコード（その `messages` フィールド）を開いてください。

### 「なぜこのファイルは沈黙しているのか？」

**コメントのない**ファイルは、モデルが*能動的に* reviewed_files を呼び出した場合にのみ成功したレビューです。スイムレーンに
runner invocationはあるが comment がない場合、それはモデルが能動的に下したクリーンなレビューです。スイムレーンがエラーカードで終わっている場合、それは
沈黙を装った失敗です——警告として扱うべきです。

### 「圧縮は何を保持 / 破棄したのか？」

`memory_compression_task` スイムレーンは各圧縮ラウンドを表示します。その中で、Response ペインには結果の要約があり、
圧縮された compress 領域からレンダリングされた XML は、そのラウンドの `llm_request` の `messages`（JSONL トランスクリプト内）にあります。
「モデルが以前のコンテキストを忘れた」というフィードバックの調査に有用です——圧縮が関連する詳細を破棄したかどうかを確認できます。

## ディスクのストレージレイアウト

ビューアは以下を読み取ります：

```
~/.opencodereview/sessions/
└── <path-encoded-repo-path>/
    └── <session-id>.jsonl
```

JSONL ファイルの各行は 1 つのイベントです：

```json
{"type": "llm_request", "filePath": "src/foo.go", "taskType": "runner", "request_no": 1, "messages": [{"role": "user", "content": "Review this diff…"}], "timestamp": "2026-06-02T10:15:23Z"}
{"type": "llm_response", "filePath": "src/foo.go", "taskType": "runner", "model": "claude-sonnet-4-6", "content": "Found 2 issues…", "duration_ms": 8421, "usage": {"prompt_tokens": 12450, "completion_tokens": 320}}
{"type": "tool_call", "filePath": "src/foo.go", "tool_name": "file_read", "arguments": "{\"file_path\":\"src/foo.go\",\"start_line\":1,\"end_line\":50}", "result": "File: src/foo.go (Total lines: 220)\nIS_TRUNCATED: false\nLINE_RANGE: 1-50\n1|package foo…", "ok": true, "duration_ms": 14}
```

行は追記専用（append-only）です——不完全な JSONL は、セッションが実行中に中断されたことを意味し、ビューアは書き込み済みの
内容をレンダリングします。

ディスク容量を解放するには、セッションファイル全体を削除します。ビューアは次のリクエスト時にインデックスを再構築します。

## プライバシー

JSONL トランスクリプトには、LLM に送信され LLM から受信した**すべて**が含まれ、diff 内のあらゆるコードも含まれます。これらは
完全にあなたのマシンの `~/.opencodereview/` 内に存在します。OCR はそれらをどこにもアップロードしません。

レビューに長期保存したくないコードが含まれる場合は、以下が可能です：

- 定期的にセッションファイルを削除する、または
- CI で `--audience agent --format json` の出力を一時的なパイプにリダイレクトし、一時的な
  `HOME` で実行して JSONL が永続化されないようにする。

OpenTelemetry exporter は別の話です——prompt の内容をエクスポートされる trace に含めない方法については
[テレメトリ](../telemetry/)を参照してください。

## ビューアが適さない場合

- プログラムによる後処理（CI、ダッシュボード）には `ocr review --runner codex --format json --audience agent` を使用します。
  ビューアは人間向けのレンダリングであり、機械向けではありません。
- 複数セッションにまたがる grep が必要な場合は、JSONL ファイルに対して直接 `jq` を使用します。UI にはまだ検索ボックスがありません。

## 関連項目

- [アーキテクチャ](../architecture/)——それら 5 つのタスクタイプが内部で実際に何をするか。
- [ツール](../tools/)——`runner` カードで目にするrunner invocation。
