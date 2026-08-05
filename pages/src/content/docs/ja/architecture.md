---
title: アーキテクチャ
sidebar:
  order: 8
---

このページでは、local runner モデルで `ocr review` と `ocr scan` がどう動くかを説明します。OCR は file selection、runner prompt 生成、認証済み Codex または Claude local process の 1 回起動、structured result の検証、session 書き込みを決定的に行います。

## Pipeline

1. flags を検証し、`--runner codex` または `--runner claude` を選択します。
2. review では Git diff selection、scan では `--path` selection を作ります。
3. binary、`--exclude`、supported file types、built-in noisy paths、rules を適用します。
4. selected files から manifest を作り、`reviewed_files` coverage contract として使います。
5. local runner process を 1 つ呼び出します。
6. JSON、paths、line ranges、comment shape、coverage を検証し、text/JSON output と session を書きます。

`ocr review --preview` は selection/filtering までで停止し、runner を呼び出しません。そのため `--runner` や runner login は不要です。

## Local runner invocation

```bash
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

preview ではない review と scan には `--runner` が必須です。値は `codex` と `claude` です。OCR は selected CLI を preflight し、CLI がない、または未認証なら開始前に失敗します。`--runner-model <name>` はその run だけ selected runner に渡す model hint です。

OCR は command invocation ごとに local runner process を 1 つだけ起動します。scope は selection flags で制御し、process duration は `--timeout <minutes>` で制限します。

## Sessions, coverage, resume

各 run は次に append-only JSONL を書きます。

```text
~/.opencodereview/sessions/<encoded-repo-path>/<session-id>.jsonl
```

Session は selection、prompt metadata、runner result、validation warnings、coverage status、final comments を記録します。`--resume <session-id>` は saved session state を再利用し、同じ selection と coverage accounting で続行します。

## Source-code map

| Concern | File |
|---|---|
| Command dispatch | `cmd/opencodereview/main.go` |
| Flags | `cmd/opencodereview/flags.go` |
| Review/scan orchestration | `internal/agent/` |
| Preview / filter | `internal/agent/preview.go` |
| Diff loading | `internal/diff/git.go` |
| Runner adapters | `internal/runner/` |
| Session writer | `internal/session/persist.go` |
