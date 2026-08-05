---
title: QuickStart
sidebar:
  order: 3
---

数分で最初のコードレビューを実行します。

## 前提条件

- **Git ≥ 2.41**
- **Node.js ≥ 18**
- **サポートされるネイティブ方式で認証済みの Codex CLI または Claude Code CLI**

## Step 1 — CLI をインストール

```bash
npm install -g @alibaba-group/open-code-review
ocr version
```

## Step 2 — ローカル runner を認証

OCR は LLM 作業をインストール済みのローカル CLI に委任します。認証はその CLI がサポートするネイティブ方式（既存ログインまたは API トークン）で管理し、OCR はプロバイダー API キーを設定・使用しません。

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`ocr review` と `ocr scan` には `--runner` が必須です。1 回の実行だけ runner の既定モデルを変える場合は `--runner-model <name>` を任意で指定します。`--preview` などの読み取り専用/プリフライトコマンドは runner を呼び出さないため、認証前にも実行できます。

## Step 3 — 最初のレビューを実行

```bash
cd path/to/your-repo
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr scan --runner claude --path internal/agent
```

OCR が runner 認証エラーを出した場合は、選択した CLI をネイティブログインまたは API トークンで認証してから再試行してください。CI 認証はローカル状態とは別です。OCR のプロバイダー秘密情報ではなく、CI ジョブ内で Codex または Claude を認証します。

## 関連項目

- [インストール](../installation/)
- [設定](../configuration/)
- [CLI リファレンス](../cli-reference/)
- [レビュー規則](../review-rules/)
