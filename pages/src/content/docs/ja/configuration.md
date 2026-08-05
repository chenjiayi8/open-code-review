---
title: Configuration
sidebar:
  order: 5
---

OCR は runner 資格情報やプロバイダー接続設定を保存しません。レビュー実行では、OCR はインストール済みのローカル runner を呼び出し、その runner がサポートするネイティブ方式（既存ログインまたは API トークン）で認証を管理します。

## ローカル runner

OCR で使う CLI を、その CLI がサポートするネイティブ方式（既存ログインまたは API トークン）で先に認証します。

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` は `codex` と `claude` をサポートします。`ocr review` と `ocr scan` では、どのローカル CLI が LLM 作業を行うかを OCR に伝えるため必須です。`--runner-model <name>` は任意で、その実行だけに適用されます。

## 読み取り専用とプリフライト

`ocr review --preview` などの読み取り専用/プリフライトコマンドは、runner を呼び出さずにファイルとルールを検査します。runner が必要なコマンドでは OCR が選択された CLI を事前確認し、CLI がない、または認証されていない場合はレビュー開始前に失敗します。対応するネイティブ認証（既存ログインまたは API トークン）を行って再試行してください。

## CI 認証

ローカル runner の認証状態は CI に自動では移りません。CI ワークフローでは OCR をインストールし、ジョブ内で選択した Codex または Claude runner を認証します。OCR 独自の資格情報設定コマンドは追加しないでください。
