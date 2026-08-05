---
title: CLI Reference
sidebar:
  order: 6
---

Справочник команд ревью с локальными runner.

## Аутентификация

OCR не управляет учётными данными runner или настройками подключения провайдера. Сначала войдите в установленную CLI runner:

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

## Сводка команд

| Команда | Алиас | Назначение |
|---|---|---|
| `ocr review` | `ocr r` | Ревью рабочей копии, commit или диапазона ref. |
| `ocr scan` | `ocr s` | Сканирование полных файлов без Git diff. |
| `ocr rules check <file>` | — | Показать правило ревью для файла. |
| `ocr session list` | — | Список сохраненных сессий ревью. |
| `ocr session show <id>` | — | Просмотр сохраненной сессии. |
| `ocr session comments <id>` | — | Вывести комментарии сохраненной сессии. |
| `ocr viewer` | — | Запустить локальный просмотрщик сессий. |
| `ocr version` | — | Показать версию и сведения о сборке. |

## Обязательные флаги runner

Для `ocr review` и `ocr scan` вне preview обязателен `--runner codex` или `--runner claude`. `--runner-model <name>` необязателен и передается выбранному runner только для текущего запуска. OCR выполняет preflight выбранного runner и завершается до ревью, если CLI отсутствует или не аутентифицирована.

Read-only/preflight команды, включая `ocr review --preview` и `ocr scan --preview`, проверяют файлы и правила без вызова runner.

## Примеры `ocr review`

```bash
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr review --runner codex --runner-model gpt-5-codex --format json
ocr review --preview
```

Основные флаги: `--repo`, `--runner`, `--runner-model`, `--from`, `--to`, `--commit`, `--resume`, `--format`, `--audience`, `--background`, `--background-file`, `--timeout`, `--rule`, `--exclude`, `--max-git-procs`.

## Примеры `ocr scan`

```bash
ocr scan --runner claude
ocr scan --runner claude --path internal/agent
ocr scan --runner claude --runner-model sonnet --format json
ocr scan --preview --path internal/agent
```

`ocr scan` проверяет полные файлы. Он принимает общие флаги вывода, runner, timeout, rules, exclude и resume, а также scan-переключатели `--path`, `--no-plan`, `--no-dedup`, `--no-summary`.

## Локальная аутентификация runner и CI

Локальный вход принадлежит Codex или Claude CLI на данной машине. CI jobs должны аутентифицировать выбранный runner внутри CI; не добавляйте настройку учетных данных провайдера OCR.
