---
title: Configuration
sidebar:
  order: 5
---

OCR は runner 資格情報やプロバイダー接続設定を保存しません。レビュー実行はサブスクリプション対応で、OCR はインストール済みのローカル runner を呼び出し、その runner が認証を管理します。

## ローカルサブスクリプション runner

OCR で使う CLI に先にログインします。

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` は `codex` と `claude` をサポートします。`ocr review` と `ocr scan` では、どのローカルサブスクリプション CLI が LLM 作業を行うかを OCR に伝えるため必須です。`--runner-model <name>` は任意で、その実行だけに適用されます。

## 読み取り専用とプリフライト

`ocr review --preview` などの読み取り専用/プリフライトコマンドは、runner を呼び出さずにファイルとルールを検査します。runner が必要なコマンドでは OCR が選択された CLI を事前確認し、CLI がない、またはログインしていない場合はレビュー開始前に失敗します。対応するログインコマンドを実行して再試行してください。

## CI 認証

ローカルのサブスクリプションログインは CI に自動では移りません。CI ワークフローでは OCR をインストールし、ジョブ内で選択した Codex または Claude runner を認証します。OCR 独自の資格情報設定コマンドは追加しないでください。
