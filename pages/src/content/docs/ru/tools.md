---
title: Инструменты
sidebar:
  order: 9
---

Model work для review и scan делегируется выбранному local subscription runner: Codex CLI или Claude Code CLI. OCR отвечает за deterministic selection, validation, coverage и session output.

## Runner selection

```bash
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` обязателен для `ocr review` и `ocr scan` без preview. Preview — read-only preflight и не вызывает runner. `--runner-model <name>` — model hint для выбранного runner только на этот run.

## Permissions and scope

OCR запускает один authenticated local runner process для review или scan. Embedded prompt просит runner читать repository read-only и вернуть strict JSON. CLI switches для добавления, удаления или переопределения model tools нет.

- Codex выполняется через installed Codex CLI.
- Claude выполняется через installed Claude Code CLI с read-only `Read`, `Glob`, `Grep` permission expectations.
- Authentication, subscription quota, retry и model-side tool behavior принадлежат selected runner CLI.

`--timeout <minutes>` ограничивает весь runner process. Если run слишком большой, сузьте selection через review range, `--exclude` или scan `--path`.

## Supported customization

| Need | Use |
|---|---|
| Choose CLI | `--runner codex` / `--runner claude` |
| One-run model hint | `--runner-model <name>` |
| Bound duration | `--timeout <minutes>` |
| Add background | `--background` / `--background-file` |
| Change rules | `--rule <file>` |
| Narrow scope | `--exclude`, review ranges, scan `--path` |

Runner prompt, JSON schema или permission policy меняются только через source change и rebuild; это не runtime configuration flags.
