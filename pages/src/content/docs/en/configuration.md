---
title: Configuration
sidebar:
  order: 5
---

OCR does not store runner credentials or provider connection settings. Review execution shells out to an installed local runner, and that runner owns authentication through its supported native mechanism.

## Local runners

Authenticate the CLI you want OCR to use with its supported native login or API-token mechanism:

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

Supported `--runner` values are `codex` and `claude`. The flag is required for `ocr review` and `ocr scan` because OCR must know which local CLI should do the LLM work. `--runner-model <name>` is optional and applies only to that invocation.

## Read-only and preflight behavior

Read-only/preflight commands such as `ocr review --preview` inspect files and rules without invoking a runner. When a command does need a runner, OCR preflights the selected CLI and fails before review work if the CLI is missing. If the native command reports an authentication error, authenticate that CLI and retry.

## CI authentication

Local runner authentication does not automatically transfer to CI. CI workflows must install OCR and authenticate the selected Codex or Claude runner inside the job. Do not add OCR-owned credential setup commands; keep CI runner authentication in the CI platform's supported mechanism.
