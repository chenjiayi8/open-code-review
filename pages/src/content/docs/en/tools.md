---
title: Tools
sidebar:
  order: 9
---

Review and scan model work is delegated to the selected local subscription runner: Codex CLI or Claude Code CLI. OCR owns deterministic selection, validation, coverage, and session output.

## Runner selection

Use `--runner` to choose which installed local CLI performs the model work:

```bash
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` is required for non-preview `ocr review` and `ocr scan` runs. Preview commands are read-only preflight commands and do not invoke a runner.

Use `--runner-model <name>` only when you want to pass a one-run model hint to the selected runner:

```bash
ocr review --runner codex --runner-model gpt-5 --timeout 20
```

## Permissions and runner tools

OCR starts one authenticated local runner process for the selected review or scan. The embedded prompt asks the runner to inspect repository content read-only and return strict JSON. OCR does not provide command-line switches for adding, removing, or redefining model tools.

- Codex runs through the installed Codex CLI with OCR's read-only sandbox expectations.
- Claude runs through the installed Claude Code CLI with OCR's read-only `Read`, `Glob`, and `Grep` permission expectations.
- Authentication, subscription quota, provider retries, and model-side tool behavior belong to the selected runner CLI.

## Timeouts and scope

`--timeout <minutes>` bounds the overall local runner process. If a run is too large or the selected subscription hits quota limits, narrow the deterministic selection instead of looking for per-file concurrency controls:

```bash
ocr review --runner codex --from origin/main --to HEAD --exclude "docs/**"
ocr scan --runner claude --path internal/agent --timeout 30
```

## Customizing behavior

Use OCR flags and inputs for supported customization:

| Need | Use |
|---|---|
| Choose local CLI | `--runner codex` or `--runner claude` |
| Hint a model for one run | `--runner-model <name>` |
| Bound process duration | `--timeout <minutes>` |
| Add project context | `--background` or `--background-file` |
| Change review guidance | `--rule <file>` |
| Narrow selected files | `--exclude`, review ranges, or scan `--path` |

Changing embedded runner prompts, JSON schemas, or permission policy requires a source change and rebuild; these are not runtime configuration flags.

## See Also

- [Architecture](../architecture/) — how OCR selects files, invokes the runner, validates coverage, and writes sessions.
- [Configuration](../configuration/) — runner flags and environment behavior.
- [Session Viewer](../viewer/) — inspect saved review and scan sessions.
