---
title: CLI Reference
sidebar:
  order: 6
---

The complete reference for local subscription runner review commands.

## Authentication

OCR does not manage runner credentials or provider connection settings. Authenticate the installed runner CLI first:

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

## Command summary

| Command | Alias | What it does |
|---|---|---|
| `ocr review` | `ocr r` | Review a Git workspace, commit, or ref range. |
| `ocr scan` | `ocr s` | Scan complete files without requiring a Git diff. |
| `ocr rules check <file>` | — | Show which review rule applies to a file. |
| `ocr session list` | — | List saved review sessions. |
| `ocr session show <id>` | — | Inspect one saved review session. |
| `ocr session comments <id>` | — | Print comments recorded in a saved session. |
| `ocr viewer` | — | Launch the local session viewer. |
| `ocr version` | — | Print version and build information. |

## Required runner flags

`ocr review` and `ocr scan` require `--runner codex` or `--runner claude` for non-preview execution. `--runner-model <name>` is optional and is passed to the selected runner for that invocation only. OCR preflights the selected runner and fails before review work if the runner CLI is missing or not authenticated.

Read-only/preflight commands, including `ocr review --preview` and `ocr scan --preview`, inspect files and rules without invoking a runner.

## `ocr review` examples

```bash
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr review --runner codex --runner-model gpt-5-codex --format json
ocr review --preview
```

Common review flags include `--repo`, `--runner`, `--runner-model`, `--from`, `--to`, `--commit`, `--resume`, `--format`, `--audience`, `--background`, `--background-file`, `--timeout`, `--rule`, `--exclude`, and `--max-git-procs`.

## `ocr scan` examples

```bash
ocr scan --runner claude
ocr scan --runner claude --path internal/agent
ocr scan --runner claude --runner-model sonnet --format json
ocr scan --preview --path internal/agent
```

`ocr scan` reviews full files. It accepts the shared output, runner, timeout, rule, exclusion, and resume flags plus scan-specific toggles such as `--path`, `--no-plan`, `--no-dedup`, and `--no-summary`.

## Local subscription vs CI

Local login state belongs to the Codex or Claude CLI on that machine. CI jobs must authenticate the selected runner inside CI; do not add OCR provider credential setup.
