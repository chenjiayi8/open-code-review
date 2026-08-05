---
title: CLI Reference
sidebar:
  order: 6
---

ローカルサブスクリプション runner を使うレビューコマンドのリファレンスです。

## 認証

OCR は runner 資格情報やプロバイダー接続設定を管理しません。先にインストール済み runner CLI にログインします。

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

## コマンド概要

| コマンド | エイリアス | 内容 |
|---|---|---|
| `ocr review` | `ocr r` | ワークスペース、commit、ref 範囲をレビューします。 |
| `ocr scan` | `ocr s` | Git diff なしで完全なファイルをスキャンします。 |
| `ocr rules check <file>` | — | ファイルに適用されるレビュー規則を表示します。 |
| `ocr session list` | — | 保存済みレビューセッションを一覧表示します。 |
| `ocr session show <id>` | — | 保存済みセッションを確認します。 |
| `ocr session comments <id>` | — | 保存済みセッションのコメントを出力します。 |
| `ocr viewer` | — | ローカルセッションビューアを起動します。 |
| `ocr version` | — | バージョンとビルド情報を表示します。 |

## 必須 runner フラグ

preview ではない `ocr review` と `ocr scan` には `--runner codex` または `--runner claude` が必須です。`--runner-model <name>` は任意で、現在の実行だけ選択した runner に渡されます。OCR は選択した runner をプリフライトし、CLI がない、または未ログインの場合はレビュー開始前に失敗します。

`ocr review --preview` や `ocr scan --preview` などの読み取り専用/プリフライトコマンドは、ファイルと規則だけを検査し runner を呼び出しません。

## `ocr review` 例

```bash
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr review --runner codex --runner-model gpt-5-codex --format json
ocr review --preview
```

主なフラグは `--repo`、`--runner`、`--runner-model`、`--from`、`--to`、`--commit`、`--resume`、`--format`、`--audience`、`--background`、`--background-file`、`--timeout`、`--rule`、`--exclude`、`--max-git-procs` です。

## `ocr scan` 例

```bash
ocr scan --runner claude
ocr scan --runner claude --path internal/agent
ocr scan --runner claude --runner-model sonnet --format json
ocr scan --preview --path internal/agent
```

`ocr scan` は完全なファイルをレビューします。共有の出力、runner、タイムアウト、規則、除外、再開フラグに加え、`--path`、`--no-plan`、`--no-dedup`、`--no-summary` などのスキャン用トグルを受け付けます。

## ローカルサブスクリプションと CI

ローカルログイン状態はそのマシンの Codex または Claude CLI に属します。CI ジョブでは CI 内で選択した runner を認証してください。OCR のプロバイダー資格情報設定は追加しません。
