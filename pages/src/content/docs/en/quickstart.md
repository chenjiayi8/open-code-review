---
title: QuickStart
sidebar:
  order: 3
---

Get your first code review running in a few minutes.

## Prerequisites

- **Git ≥ 2.41**
- **Node.js ≥ 18**
- **Codex CLI or Claude Code CLI with an active local subscription login**

## Step 1 — Install the CLI

```bash
npm install -g @alibaba-group/open-code-review
ocr version
```

## Step 2 — Authenticate a local runner

OCR delegates LLM work to an installed local CLI. The local CLI owns subscription authentication; OCR neither configures nor uses provider API keys.

```bash
codex login                 # or: claude auth login --claudeai
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

If OCR reports a preflight authentication failure, run the login command for the selected runner and retry. CI authentication is separate from local subscription login; authenticate Codex or Claude inside the CI job instead of configuring OCR provider secrets.

## See Also

- [Installation](../installation/) — every install method and OCR's state directory.
- [Configuration](../configuration/) — local runner authentication and CI notes.
- [CLI Reference](../cli-reference/) — every sub-command, flag, and output mode.
- [Review Rules](../review-rules/) — customize what gets reviewed.
