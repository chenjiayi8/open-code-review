---
title: QuickStart
sidebar:
  order: 3
---

Get your first code review running in a few minutes.

## Prerequisites

- **Git ≥ 2.41**
- **Node.js ≥ 18**
- **Codex CLI or Claude Code CLI authenticated with its supported native mechanism**

## Step 1 — Install the CLI

```bash
npm install -g @alibaba-group/open-code-review
ocr version
```

## Step 2 — Authenticate a local runner

OCR delegates LLM work to an installed local CLI. The local CLI owns authentication through its supported native mechanism (existing login or API token); OCR does not store runner credentials.

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` is required for review and scan runs. Use optional `--runner-model <name>` per run when you want to override the runner default. Read-only/preflight commands such as `--preview` do not invoke a runner and can run before login.

## Step 3 — Run your first review

```bash
cd path/to/your-repo
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr scan --runner claude --path internal/agent
```

If OCR reports a runner authentication failure, authenticate the selected CLI with its native login or API-token mechanism and retry. CI authentication is separate from local machine state; authenticate Codex or Claude inside the CI job instead of configuring OCR provider secrets.

## See Also

- [Installation](../installation/) — every install method and OCR's state directory.
- [Configuration](../configuration/) — local runner authentication and CI notes.
- [CLI Reference](../cli-reference/) — every sub-command, flag, and output mode.
- [Review Rules](../review-rules/) — customize what gets reviewed.
