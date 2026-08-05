---
title: ツール
sidebar:
  order: 9
---

Review と scan の model work は selected local subscription runner、つまり Codex CLI または Claude Code CLI に委任されます。OCR は deterministic selection、validation、coverage、session output を担います。

## Runner selection

```bash
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

preview ではない `ocr review` と `ocr scan` には `--runner` が必須です。Preview は read-only preflight で runner を呼び出しません。`--runner-model <name>` は selected runner にその run だけ渡す model hint です。

## Permissions and scope

OCR は review または scan につき authenticated local runner process を 1 つ起動します。Embedded prompt は repository を read-only で確認し strict JSON を返すよう runner に求めます。Model tools を追加、削除、再定義する CLI switch はありません。

- Codex は installed Codex CLI で実行されます。
- Claude は installed Claude Code CLI で実行され、read-only `Read`、`Glob`、`Grep` permission expectations に従います。
- Authentication、subscription quota、retry、model-side tool behavior は selected runner CLI の責務です。

`--timeout <minutes>` は runner process 全体を制限します。Run が大きすぎる場合は review range、`--exclude`、scan `--path` で selection を狭めてください。

## Supported customization

| Need | Use |
|---|---|
| Choose CLI | `--runner codex` / `--runner claude` |
| One-run model hint | `--runner-model <name>` |
| Bound duration | `--timeout <minutes>` |
| Add background | `--background` / `--background-file` |
| Change rules | `--rule <file>` |
| Narrow scope | `--exclude`, review ranges, scan `--path` |

Runner prompt、JSON schema、permission policy を変えるには source change と rebuild が必要です。これらは runtime configuration flags ではありません。
