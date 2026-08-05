---
title: FAQ
sidebar:
  order: 14
---

よくあるエラー、想定外の挙動、そして「これは仕様ですか？」という疑問をまとめました。ここに
あなたの問題が見つからない場合は、実行手順と完全な出力を添えて
[GitHub issue](https://github.com/alibaba/open-code-review/issues) を作成してください。

## 設定と起動

### Runner authentication failed

OCR の review/scan はインストール済みのローカル runner を使います。preview 以外では `--runner codex` または `--runner claude` を指定し、先に対応する CLI を native login または API token で認証してください。

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

このエラーは、選択した runner が見つからない、ログアウトしている、または現在の環境で利用できないことを示します。runner をインストール/ログインしてから再試行してください。OCR 独自の runner 認証変数を作らず、選択した runner の公式ログインフローで認証してください。

### Preview は動くが review が失敗する

`ocr review --preview` は読み取り専用で runner を呼び出しません。完全な review は、`--runner` がない場合や選択した Codex/Claude CLI が未認証の場合に失敗します。ログイン後、`--runner codex` または `--runner claude` 付きで再実行してください。

### Runner の認証エラー

ログイン、API token、quota、権限エラーは選択した Codex または Claude CLI から返されます。同じ shell または CI job 内でそのツールを再認証してから OCR を再実行してください。CI では、`ocr review --runner ...` の前に CI プラットフォームがサポートする secret/OIDC/device-flow などで runner を認証します。

### `not a git repository`

`ocr review --runner codex` はカレントディレクトリに対して `git diff`（および untracked ファイルに対する
`git ls-files`）を実行します。Git ワークツリー内にいない場合は、早期に終了します。リポジトリに
`cd` するか、`--repo /path/to/repo` を渡してください。

### local runner がツール、quota、タイムアウトのエラーを返す

OCR はモデル実行をログイン済みの Codex または Claude CLI に委譲します。runner がツール、quota、retry、timeout の問題を報告する場合は runner のログイン状態を確認し、`--path`、`--exclude`、または小さな diff で範囲を絞ってください。処理に本当に時間が必要な場合は `--timeout <minutes>` を上げます。

## フィルタリングとルール

### ファイルがレビューされない

`ocr review --preview` を実行してください（LLM コストなし）。出力には各候補ファイルと、それが
保持されたか破棄されたかの**理由**が一覧されます。

```
src/foo.go              modified
src/foo_test.go         modified  (excluded: user_exclude)
node_modules/lib.js     added     (excluded: default_path)
imgs/logo.png           binary    (excluded: unsupported_ext)
```

5 種類の除外理由は、[ファイルフィルタリング](../review-rules/#how-files-are-filtered)のゲートに対応します。

| 理由 | 修正方法 |
|---|---|
| `binary` | 対処不要——バイナリファイルにはレビュー可能なテキストがありません。 |
| `user_exclude` | あなたの `exclude` リストからそのパターンを削除してください。 |
| `unsupported_ext` | ホワイトリストゲートを回避するため、拡張子を `include` リストに追加してください。 |
| `default_path` | ファイルを `include` に追加してください——組み込みのテストファイル除外パターンを上書きします。 |
| `deleted` | 対処不要——レビュー対象の新しい内容がありません。 |

### カスタムルールが発火しない

`ocr rules check <file-path>` を実行してください。マッチした**レイヤー**と **glob パターン**を
すべて表示します。

```
File: src/api/UserHandler.go
Source: Project (.opencodereview/rule.json)
Pattern: src/api/**/*.go
Rule: …
```

レイヤーが正しくない場合（プロジェクトルールを期待していたのに "System built-in" と表示される等）、
多くは**宣言順序**の問題です——最初にマッチしたパターンが適用されます。より具体的なルールを
`rules` 配列内で前に移動するか、glob を修正してください。

### 波括弧展開が動作しない

`bmatcuk/doublestar/v4` は `{ts,tsx}` の波括弧をサポートします。マッチしない場合は、余分な空白を
確認してください——`{ts, tsx}` のように空白があると、`tsx` に静かにマッチしなくなります。

## レビュー

### あるファイルにコメントが 0 件——本当にレビューされたのか？

[セッションビューア](../viewer/)（`ocr viewer`）を開いて session を確認してください。コメントが 0 件のファイルは、`reviewed_files` に含まれ validation warning がない場合だけ clean review です。coverage にない場合は、review 前に filtered されたか local runner output が incomplete だったかを確認してください。

### コメントの `start_line: 0` と `end_line: 0`

OCR はコメントを diff 内の正確な行にアンカーできませんでした。よくある原因は 2 つです。

- モデルが `existing_code` を diff からそのままコピーせず、書き換えてしまった。モデルには
  そうしないよう指示されていますが、時々起きます。
- diff のフォーマットが異常（CRLF、tab/空白の混在）で、スライディングウィンドウのマッチングが
  壊れた。

コメント自体は本物です——ただ自動的に配置されなかっただけです。ほとんどのエージェント統合
（SKILL、Claude Code plugin）は `existing_code` フィールドを読み、ファイル内で自ら位置を特定します。

### Token threshold exceeded

```
[ocr] WARNING: prompt tokens (94000) exceed 80% of max_tokens(58888) for src/big.sql
```

そのファイルの初期 prompt（ルール + diff + change-files リスト）が、モデルが応答できる前に
すでに `MAX_TOKENS = 58888` の 80 % を超えました。OCR はそのファイルをスキップして続行します——
JSON モードでは `warnings` にも表示されます。

緩和策:

- 自動生成されたものなら、ファイルを `exclude` リストに追加してください。
- 大きなリファクタリングをより小さな commit に分割してください。
- 一連の小さな commit に対しては、一括のワークスペースモードではなく `--commit` モードで
  レビューしてください。

### ファイルが小さいのに plan フェーズに時間がかかる

まず `ocr review --preview` を実行してください。ファイルの `lines.changed` が
`PLAN_MODE_LINE_THRESHOLD`（デフォルト **50**）を超えると、plan フェーズが実行されます。これは
意図的なものです——大きな diff は plan の恩恵を受けます。単一のレビューでスキップするには、
より小さな diff で実行するか、埋め込みテンプレートを一時的に編集してください（上級者向け。
内蔵 runner prompt の変更と再ビルドが必要です）。


### local runner がタイムアウトする、または一部ファイルを報告しない

OCR は runner の `reviewed_files` を coverage の証拠として扱います。含まれないファイルは session 出力で未完了として記録されます。timeout の場合は範囲を絞るか、`--timeout <minutes>` を上げてください。

### 一部のファイルが未完了でも結果は残る

OCR は検証済みコメントを保持し、未 coverage または失敗したファイルを session/JSON 出力に記録します。再実行が必要なパスは `warnings` と session 詳細で確認できます。

### CI での実行がローカルよりずっと遅い

CI 環境では選択した runner のインストールと認証が毎回必要になり、ローカル cache もありません。OCR 実行前に Codex または Claude にログインしていることを確認してください。runner が quota/timeout を報告する場合は範囲を絞るか、`--timeout <minutes>` を使います。

## 出力と統合

### `--audience agent` でも進捗行が出る

見ているのが **stderr** でないことを確認してください。進捗メッセージは時々 stderr に出ます
（警告、エラー）。`--audience agent` が保証するクリーンな stdout は*パーサーに優しい*ものです——
すべてを遮断するにはリダイレクトしてください: `ocr review --runner codex --audience agent 2>/dev/null`（先に選択した runner を認証します）。

### JSON 出力が `{ "files_reviewed": 0, "comments": [] }`

ワークスペースに対象ファイルがありません。これは意図的なものです——明示的な形により、呼び出し側は
「レビュー対象がない」ことと「レビューしたファイルに指摘がない」ことを区別できます。コメントが
ゼロの正常なレビューは、通常の空配列 `[]` を返します。

### セッション JSONL はどこにある？

```
~/.opencodereview/sessions/<path-encoded-repo-path>/<session-id>.jsonl
```

リポジトリパスは、`/` と `\` を `-` に、`:` を `_` に置き換えてエンコードされます
（例: `/Users/foo/my-repo` → `Users-foo-my-repo`）。`ocr viewer` でセッションを閲覧できます。
このディレクトリを削除すると履歴が消えます。OCR は次回実行時にエンコード済みパスを再生成します。

## パフォーマンスとコスト

### どの token にどれだけ費やしたかを知るには？

テレメトリを有効にします。

```bash
ocr config set telemetry.enabled true
ocr config set telemetry.exporter console
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
```

LLM 呼び出しには独自の span がありません——metric として記録されます。`ocr.llm.tokens_used`
（counter、`model` + `type` でラベル付け）、`ocr.llm.requests_total`（counter、`model`
+ `status` でラベル付け）、`ocr.llm.request_duration_seconds`（histogram、`model` でラベル付け）に
注目してください。console exporter はこれらの集計をインラインで出力します。ダッシュボードが必要な場合は、
OTLP exporter に切り替えて metrics 基盤に送ってください——[テレメトリ](../telemetry/)を参照。

### runner の作業量を減らすには？

- `include`/`exclude` ルールで不要なファイルを外します。
- 大きな変更は小さな commit、範囲、または `--path` に分けます。
- `--background` で必要な業務コンテキストを先に渡します。

## プライバシーとセキュリティ

### OCR は私のコードをどこかに送るのか？

OCR はあなたの **diff**（および任意の read-tool スニペット）を、選択済みかつ認証済みのローカル runner に渡します。
それ以外のものは一切あなたのマシンから出ません——セッション JSONL とルールファイルはローカルにのみ
存在します。

テレメトリを有効にしている場合、`content_logging` フラグは設定レイヤーに接続されていますが、
現時点ではどのコードパスも**制御しません**——このフラグの値にかかわらず、prompt と応答の内容が
collector にエクスポートされることは決してありません。予約項目とみなしてください。本番環境では
`false` のままにしてください。詳細は[テレメトリ](../telemetry/#content-logging)を参照してください。

### LLM に送る前に secret をマスクできるか？

組み込み機能ではありません。推奨のワークフロー:

1. secret をリポジトリにコミットしない（一般的なルール）。
2. 機密情報を含むと分かっているファイルを `exclude` に追加する。
3. `git diff --no-textconv` フィルターまたは pre-commit でマスクし、secret が diff に入らないようにする。

「マスクルール」機能はロードマップにあります。
[issue トラッカー](https://github.com/alibaba/open-code-review/issues)をご覧ください。

## その他

### changelog はどこにある？

[GitHub Releases](https://github.com/alibaba/open-code-review/releases)
——各 release には Conventional Commits から生成された notes が付属しています。

### OCR は Git 以外の VCS をサポートするか？

しません。diff provider は shell 経由で `git` を呼び出します。SVN / Mercurial などには新しい
provider が必要です。Hg サポートの issue は[こちら](https://github.com/alibaba/open-code-review/issues)で
オープンになっています。

### なぜバイナリは `opencodereview` なのに CLI は `ocr` なのか？

release で配布される静的バイナリはプロジェクト名（`opencodereview`）を持ちます。NPM wrapper は
使いやすさのため `ocr` としてインストールされます。ソースからビルドすると `dist/opencodereview` が
得られます——`$PATH` 上の `ocr` としてコピーしてください。

### アンインストールするには？

```bash
npm uninstall -g @alibaba-group/open-code-review        # NPM install
sudo rm /usr/local/bin/ocr                              # binary install
rm -rf ~/.opencodereview                                # all state
```

OCR は `~/.opencodereview` の外には書き込みません（NPM がダウンロードするバイナリを除く）。
したがってこのディレクトリを削除すれば、履歴、設定、ユーザーごとのルールが消去されます。

## 関連項目

- [設定](../configuration/)——runner 選択と config key。
- [レビュールール](../review-rules/)——ファイルフィルターとルール解決チェーン。
- [セッションビューア](../viewer/)——過去のレビューセッションを表示する。
- [テレメトリ](../telemetry/)——token 使用量と LLM メトリクス。
