---
title: Configuration
sidebar:
  order: 5
---

OCR no longer stores runner credentials or provider connection settings. Review execution is subscription-backed: OCR shells out to an installed local runner and that runner owns authentication.

## Local subscription runners

Log in to the CLI you want OCR to use:

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

Supported `--runner` values are `codex` and `claude`. The flag is required for `ocr review` and `ocr scan` because OCR must know which local subscription CLI should do the LLM work. `--runner-model <name>` is optional and applies only to that invocation.

## Read-only and preflight behavior

Read-only/preflight commands such as `ocr review --preview` inspect files and rules without invoking a runner. When a command does need a runner, OCR preflights the selected CLI and fails before review work if the CLI is missing or not logged in. Run the relevant login command, then retry.

## CI authentication

Local subscription login does not automatically transfer to CI. CI workflows must install OCR and authenticate the selected Codex or Claude runner inside the job. Do not add OCR-owned credential setup commands; keep CI runner authentication in the CI platform's supported mechanism.
