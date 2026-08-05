---
title: QuickStart
sidebar:
  order: 3
---

Запустите первое ревью за несколько минут.

## Требования

- **Git ≥ 2.41**
- **Node.js ≥ 18**
- **Codex CLI или Claude Code CLI с активным локальным входом по подписке**

## Шаг 1 — Установите CLI

```bash
npm install -g @alibaba-group/open-code-review
ocr version
```

## Шаг 2 — Войдите в локальный runner

OCR передает LLM-работу установленной локальной CLI. Подписочной аутентификацией владеет локальная CLI; OCR не хранит учётные данные runner.

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

Для `ocr review` и `ocr scan` обязателен `--runner`. Чтобы переопределить модель runner только для одного запуска, используйте необязательный `--runner-model <name>`. Read-only/preflight команды вроде `--preview` не вызывают runner и могут выполняться до входа.

## Шаг 3 — Запустите первое ревью

```bash
cd path/to/your-repo
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr scan --runner claude --path internal/agent
```

Если OCR сообщает об ошибке preflight-аутентификации, выполните вход для выбранного runner и повторите команду. CI-аутентификация отделена от локального входа по подписке: входите в Codex или Claude внутри CI-задачи, а не храните секреты провайдера OCR.

## См. также

- [Установка](../installation/)
- [Конфигурация](../configuration/)
- [CLI Reference](../cli-reference/)
- [Правила ревью](../review-rules/)
