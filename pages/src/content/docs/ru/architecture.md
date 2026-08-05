---
title: Архитектура
sidebar:
  order: 8
---

Эта страница описывает, как `ocr review` и `ocr scan` работают в модели local runner. OCR детерминированно выбирает файлы, формирует runner prompt, запускает один аутентифицированный local process Codex или Claude, проверяет structured result и записывает session.

## Pipeline

1. Проверить flags и выбрать `--runner codex` или `--runner claude`.
2. Для review собрать Git diff selection; для scan собрать `--path` selection.
3. Применить фильтры binary, `--exclude`, supported file types, built-in noisy paths и rules.
4. Создать manifest выбранных файлов и использовать его как `reviewed_files` coverage contract.
5. Вызвать один local runner process.
6. Проверить JSON, paths, line ranges, comment shape и coverage, затем записать text/JSON output и session.

`ocr review --preview` выполняет только selection/filtering и не запускает runner, поэтому `--runner` и runner login не нужны.

## Local runner invocation

```bash
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

Для review и scan без preview обязателен `--runner`. Поддерживаются `codex` и `claude`. OCR preflight-проверяет выбранную CLI и завершается до начала, если CLI отсутствует или не аутентифицирована. `--runner-model <name>` — model hint для выбранного runner только на этот run.

OCR запускает только один local runner process на command invocation. Scope задаётся selection flags, а process duration ограничивается `--timeout <minutes>`.

## Sessions, coverage, resume

Каждый run пишет append-only JSONL:

```text
~/.opencodereview/sessions/<encoded-repo-path>/<session-id>.jsonl
```

Session фиксирует selection, prompt metadata, runner result, validation warnings, coverage status и final comments. `--resume <session-id>` использует saved session state, чтобы продолжить с тем же selection и coverage accounting.

## Source-code map

| Concern | File |
|---|---|
| Command dispatch | `cmd/opencodereview/main.go` |
| Flags | `cmd/opencodereview/flags.go` |
| Review/scan orchestration | `internal/agent/` |
| Preview / filter | `internal/agent/preview.go` |
| Diff loading | `internal/diff/git.go` |
| Runner adapters | `internal/runner/` |
| Session writer | `internal/session/persist.go` |
