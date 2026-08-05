---
title: Configuration
sidebar:
  order: 5
---

OCR не хранит учётные данные runner или настройки подключения провайдера. OCR запускает установленный локальный runner, а аутентификацией через native login или API token владеет сам runner.

## Локальные runner

Аутентифицируйте CLI, который должен использовать OCR, через поддерживаемый native login или API token:

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` поддерживает значения `codex` и `claude`. Для `ocr review` и `ocr scan` этот флаг обязателен, потому что OCR должен знать, какая локальная CLI выполняет LLM-работу. `--runner-model <name>` необязателен и действует только на текущий запуск.

## Read-only и preflight

Read-only/preflight команды вроде `ocr review --preview` проверяют файлы и правила без вызова runner. Если команде нужен runner, OCR сначала выполняет preflight выбранной CLI и завершается до ревью, если CLI отсутствует или не аутентифицирована. Выполните нативную аутентификацию выбранной CLI (существующий вход или API-токен) и повторите.

## CI-аутентификация

Локальная аутентификация runner не переносится в CI автоматически. CI workflow должен установить OCR и аутентифицировать выбранный Codex или Claude runner внутри job. Не добавляйте в OCR команды настройки собственных учётных данных; используйте механизм аутентификации runner, поддерживаемый CI-платформой.
