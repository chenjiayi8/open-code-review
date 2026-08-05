---
title: Configuration
sidebar:
  order: 5
---

OCR больше не хранит эндпоинты провайдеров, модели или API-ключи. Выполнение ревью опирается на подписку: OCR запускает установленный локальный runner, а аутентификацией владеет сам runner.

## Локальные subscription runner

Войдите в CLI, который должен использовать OCR:

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` поддерживает значения `codex` и `claude`. Для `ocr review` и `ocr scan` этот флаг обязателен, потому что OCR должен знать, какая локальная CLI выполняет LLM-работу. `--runner-model <name>` необязателен и действует только на текущий запуск.

## Read-only и preflight

Read-only/preflight команды вроде `ocr review --preview` проверяют файлы и правила без вызова runner. Если команде нужен runner, OCR сначала выполняет preflight выбранной CLI и завершается до ревью, если CLI отсутствует или не выполнен вход. Запустите соответствующую команду входа и повторите.

## CI-аутентификация

Локальный вход по подписке не переносится в CI автоматически. CI workflow должен установить OCR и аутентифицировать выбранный Codex или Claude runner внутри job. Не добавляйте в OCR команды настройки эндпоинтов провайдеров или API-ключей; используйте механизм аутентификации runner, поддерживаемый CI-платформой.
