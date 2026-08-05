---
title: CI/CD
sidebar:
  order: 4
---

Запускайте OCR для каждого Pull Request или Merge Request. В основном
репозитории есть два готовых конвейера, которые достаточно скопировать и
настроить: один для GitHub Actions, другой для GitLab CI. Оба являются тонкими
обёртками над основной командой, описанной в
[справочнике CLI](../cli-reference/#json).

## Как работает интеграция с CI/CD

Все рецепты на этой странице следуют одной схеме; разделы GitHub Actions и
GitLab CI ниже представляют её конкретные реализации:

1. **Запуск по событию PR / MR.** Новый pull request, обновлённый merge request
   или ручной комментарий `/open-code-review` запускает задачу.
2. **Установка `ocr`** на runner, обычно командой
   `npm install -g @alibaba-group/open-code-review`. Runner временный, поэтому
   это выполняется при каждом запуске.
3. **Аутентификация выбранного локального runner** внутри CI job. Установите Codex или Claude и используйте поддерживаемый CI-платформой неинтерактивный механизм входа (например, утверждённый secret, OIDC exchange или device-flow/bootstrap step). Сам OCR не потребляет напрямую учётные данные модельного сервиса.
4. **Запуск ревью в режиме диапазона** с машиночитаемым выводом, чтобы stdout
   содержал чистый объект JSON:

   ```bash
   ocr review --runner codex \
     --from "origin/<base-branch>" \
     --to "origin/<head-branch>" \
     --format json \
     --audience agent
   ```

   `--format json` предоставляет данные для разбора, а `--audience agent`
   подавляет строки хода выполнения. Формат объекта, используемого всеми
   рецептами, описан в разделе [Вывод JSON](../cli-reference/#json).
5. **Разбор JSON** и обход `comments[]`.
6. **Публикация комментариев** обратно в PR / MR через API ревью провайдера.
   Записи без допустимых сведений о строках (замечания на уровне файла)
   включаются в итоговую заметку вместо встроенных комментариев. Если API
   пакетной публикации отклоняет запрос, этап публикации также возвращается к
   обычному итоговому комментарию.

Всегда используются два вида учётных данных: выбранная **аутентификация runner** для
Codex или Claude, с помощью которой создаются замечания, и **токен записи PR/MR**, с помощью которого
этап публикации отправляет комментарии. В рецепте GitHub второй токен бесплатно
предоставляется как `GITHUB_TOKEN`. Для GitLab рекомендуется явно заданный
`GITLAB_API_TOKEN`, но для MR из форков в качестве резервного варианта
используется встроенный `CI_JOB_TOKEN` (он может публиковать обсуждения через
`/discussions`). Для надёжности рекомендуется отдельный токен.

## GitHub Actions

Исходный workflow находится в
[`examples/github_actions/ocr-review.yml`](https://github.com/alibaba/open-code-review/blob/main/examples/github_actions/ocr-review.yml).

### Что он делает

- Запускается по `pull_request_target` (`opened`) **и** по событиям
  `issue_comment`, тело которых начинается с `/open-code-review` или
  `@open-code-review`. Второй вариант позволяет ревьюерам повторно запускать
  OCR по запросу, оставив комментарий в PR. (`pull_request_target` используется
  вместо `pull_request`, чтобы секреты были доступны даже для PR из форков;
  OCR только читает diff и не выполняет код из PR.)
- Устанавливает OCR командой
  `npm install -g @alibaba-group/open-code-review`, записывает конфигурацию с
  аутентифицирует выбранный Codex или Claude runner, затем запускает основную команду в режиме диапазона
  веток.
- Разбирает объект JSON и публикует каждое замечание как встроенный комментарий
  ревью через GitHub Pull Request Review API. Комментарии без сведений о
  строках включаются в итоговый текст. При сбое пакетной публикации workflow
  публикует комментарии по одному и выводит статистику в итоговом комментарии.

### Установка

Поместите workflow в свой репозиторий:

```bash
mkdir -p .github/workflows
curl -o .github/workflows/ocr-review.yml \
  https://raw.githubusercontent.com/alibaba/open-code-review/main/examples/github_actions/ocr-review.yml
```

### Необходимые секреты

Задайте их в разделе **Settings → Secrets and variables → Actions**:

| Секрет | Обязательно | Описание |
|---|---|---|
| `GITHUB_TOKEN` | Авто | Токен GitHub для публикации PR review comments. |

Перед OCR добавьте шаг, который аутентифицирует выбранный установленный runner официально поддерживаемым CI-механизмом этого runner, затем запускайте OCR с `--runner codex` или `--runner claude`. Не придумывайте собственные имена runner secret для OCR.
### Настройка

Все следующие изменения вносятся в только что скопированный файл workflow
(`.github/workflows/ocr-review.yml`).

#### Фоновый контекст

`--background` — единственный флаг с наибольшим влиянием на результат; см.
[советы, применимые ко всем шаблонам](../#tips-that-apply-to-every-pattern).
Передайте заголовок PR (особенно эффективно, если заголовки следуют
семантическому соглашению вроде `feat(auth): add OAuth2 support`):

```yaml
- name: Run OCR review
  env:
    PR_TITLE: ${{ github.event.pull_request.title }}
    BASE_REF: ${{ github.base_ref }}
    HEAD_REF: ${{ github.head_ref }}
  run: |
    ocr review --runner codex \
      --background "$PR_TITLE" \
      --from "origin/$BASE_REF" \
      --to "origin/$HEAD_REF" \
      --format json --audience agent
```

Передавайте управляемые из PR значения через `env:`, а не подставляйте
`${{ }}` непосредственно в `run:`. GitHub текстово заменяет `${{ }}` *до*
разбора строки оболочкой, поэтому заголовок PR или имя ветки с метасимволами
оболочки может привести к выполнению команд на runner.

#### Пользовательские правила

Передайте файл правил проекта с помощью `--rule`:

```yaml
- name: Run OCR review
  env:
    BASE_REF: ${{ github.base_ref }}
    HEAD_REF: ${{ github.head_ref }}
  run: |
    ocr review --runner codex --rule ./my-rules.json \
      --from "origin/$BASE_REF" \
      --to "origin/$HEAD_REF"
```

Схема описана в разделе [Правила ревью](../../review-rules/).

#### Runner timeout

`--timeout <minutes>` ограничивает весь local runner process. Для крупных PR сужайте selection через review ranges, `--exclude` или path-limited scans, чтобы не расходовать quota выбранной runner quota:

```yaml
- name: Run OCR review
  env:
    BASE_REF: ${{ github.base_ref }}
    HEAD_REF: ${{ github.head_ref }}
  run: |
    ocr review --runner codex \
      --from "origin/$BASE_REF" \
      --to "origin/$HEAD_REF"
```

#### Шаблон запуска

Стандартный workflow запускается при **открытии** PR и по комментариям в PR,
начинающимся с `/open-code-review` или `@open-code-review`. Два распространённых
варианта изменения:

Запускать на большем числе событий жизненного цикла PR (например, повторять
ревью после отправки новых коммитов):

```yaml
on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]
```

Использовать другое ключевое слово в комментарии:

```yaml
if: |
  github.event_name == 'pull_request' ||
  (github.event_name == 'issue_comment'
    && github.event.issue.pull_request
    && startsWith(github.event.comment.body, '/review'))
```

Проверка `github.event.issue.pull_request` гарантирует, что комментарий
оставлен в PR, а не в обычной задаче.

#### Закрепление версии OCR

Стандартный workflow устанавливает последнюю опубликованную версию. Чтобы
закрепить конкретную:

```yaml
- name: Install OpenCodeReview
  run: npm install -g @alibaba-group/open-code-review@1.0.0
```

#### Публикация от имени GitHub App

По умолчанию комментарии ревью отправляет `github-actions[bot]`. Чтобы
публиковать их от имени брендированного бота вроде `OpenCodeReview Bot`,
замените `GITHUB_TOKEN` токеном установки GitHub App.

1. **Создайте приложение** в разделе *Settings → Developer settings → GitHub
   Apps → New GitHub App*. Отключите webhook: для этого сценария он не нужен.
   В разделе *Repository permissions* предоставьте:
   - **Pull requests**: Read and write;
   - **Contents**: Read-only (для получения diff);
   - **Metadata**: Read-only (обязательно).

2. **Создайте закрытый ключ** на странице настроек приложения и скачайте файл
   `.pem`. Запишите **App ID** с той же страницы.

3. **Установите приложение** в репозитории, которые OCR должен проверять.
   Installation ID указан в URL после установки, например
   `https://github.com/settings/installations/12345` → ID равен `12345`.

4. **Добавьте три секрета** в разделе *Settings → Secrets and variables →
   Actions*:

   | Секрет | Значение |
   |---|---|
   | `GITHUB_APP_ID` | App ID. |
   | `GITHUB_APP_PRIVATE_KEY` | Полное содержимое файла `.pem`, включая строки `-----BEGIN RSA PRIVATE KEY-----` и `-----END RSA PRIVATE KEY-----`. |
   | `GITHUB_APP_INSTALLATION_ID` | Installation ID. |

5. **Создайте токен и используйте его** на этапе публикации комментариев:

   ```yaml
   - name: Get GitHub App Token
     id: app-token
     uses: actions/create-github-app-token@v1
     with:
       app-id: ${{ secrets.GITHUB_APP_ID }}
       private-key: ${{ secrets.GITHUB_APP_PRIVATE_KEY }}

   - name: Post review comments to PR
     uses: actions/github-script@v7
     with:
       github-token: ${{ steps.app-token.outputs.token }}
       script: |
         # ...existing post script...
   ```

Теперь ревью будут публиковаться от имени приложения вместо
`github-actions[bot]`.

### Устранение неполадок

| Симптом | Причина / исправление |
|---|---|
| `Cannot find merge-base` | На этапе checkout использовалось неглубокое клонирование, но для ревью диапазона нужна полная история. Исходный workflow задаёт `fetch-depth: 0` для `actions/checkout`; сохраните эту настройку при редактировании файла. |
| `Failed to parse OCR output` | Выбранный runner не аутентифицирован в CI. Проверьте шаг входа Codex/Claude и перезапустите job. |
| Комментарии ревью попадают не на те строки | Обычно это означает, что diff изменился между началом ревью и публикацией комментариев. В этом случае скрипт публикации возвращается к обычному комментарию задачи, дополнительных действий не требуется. |

## GitLab CI

Готовый конвейер находится в
[`examples/gitlab_ci/.gitlab-ci.yml`](https://github.com/alibaba/open-code-review/blob/main/examples/gitlab_ci/.gitlab-ci.yml).

### Что он делает

- Запускается на события `merge_requests`.
- Работает в Node-образе, устанавливает OCR, аутентифицирует выбранный runner Codex или Claude, затем запускает основную команду в режиме MR diff с `--runner`.
- Разбирает JSON envelope и публикует каждое замечание как GitLab Discussion, с откатом к обычным MR notes, если inline-позиция недоступна.

### Установка

Поместите pipeline в корень репозитория:

```bash
curl -o .gitlab-ci.yml \
  https://raw.githubusercontent.com/alibaba/open-code-review/main/examples/gitlab_ci/.gitlab-ci.yml
```

Если `.gitlab-ci.yml` уже есть, положите рецепт в другой путь и подключите его:

```yaml
include:
  - local: 'ci/ocr-review.gitlab-ci.yml'
```

### Обязательные переменные CI/CD

Настройте в **Settings → CI/CD → Variables**:

| Переменная | Обязательна | Masked | Описание |
|---|---|---|---|
| `GITLAB_API_TOKEN` | Нет | Да | Project / personal / group access token с областью `api` для публикации комментариев. Необязателен; для MR из форков можно использовать встроенный `CI_JOB_TOKEN` как fallback. |

Перед OCR добавьте шаг pipeline, который аутентифицирует выбранный установленный runner официально поддерживаемым CI-механизмом этого runner, затем запускайте OCR с `--runner codex` или `--runner claude`. Не придумывайте собственные имена runner secret для OCR.

> **Быстрый совет по имени бота.** Для Project Access Token и Group Access
> Token рядом с обсуждениями MR отображается **имя** токена. Назовите токен
> `OpenCodeReview Bot`, чтобы быстро оформить ревьюера под брендом без
> дополнительной настройки. Это удобно, если вам не нужна более долговечная
> конфигурация сервисной учётной записи, описанная в разделе
> [Публикация от имени сервисной учётной записи](#post-under-a-service-account-identity).

### Настройка

Все следующие изменения вносятся в только что скопированный `.gitlab-ci.yml`.

#### Фоновый контекст

Передайте заголовок MR в `--background`. Это особенно полезно, если заголовки
следуют семантическому соглашению вроде `feat(auth): add OAuth2 support`:

```yaml
script:
  - |
    ocr review --runner codex \
      --background "$CI_MERGE_REQUEST_TITLE" \
      --from "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
      --to "${CI_COMMIT_SHA}" \
      --format json --audience agent
```

#### Пользовательские правила и runner timeout

Используйте те же флаги, что и в рецепте GitHub Actions: `--rule` для файла правил проекта и `--timeout` для ограничения процесса local runner:

```yaml
script:
  - |
    ocr review --runner codex --rule ./my-rules.json \
      --from "origin/$CI_MERGE_REQUEST_TARGET_BRANCH_NAME" \
      --to "${CI_COMMIT_SHA}"
```

Схема правил описана в разделе [Правила ревью](../../review-rules/).

#### Закрепление версии OCR

```yaml
script:
  - npm install -g @alibaba-group/open-code-review@1.0.0
```

#### Как избежать повторного ревью при каждой отправке

`only: [merge_requests]` запускается при **каждом** обновлении MR, что может
потратить много квоты подписки runner на долгоживущие MR. В GitLab нет встроенного
события «только при создании», поэтому рекомендуется обнаруживать существующие
заметки OCR перед запуском ревью и завершаться, если они найдены. Замените
вызов `ocr review --runner codex` обёрткой Python:

```python
import json, os, sys, urllib.request

GITLAB_URL = os.environ.get("CI_SERVER_URL", "https://gitlab.com")
PROJECT_ID = os.environ["CI_PROJECT_ID"]
MR_IID     = os.environ["CI_MERGE_REQUEST_IID"]
API_TOKEN  = os.environ["GITLAB_API_TOKEN"]

url = (
    f"{GITLAB_URL}/api/v4/projects/{PROJECT_ID}"
    f"/merge_requests/{MR_IID}/notes?per_page=100"
)
req = urllib.request.Request(url, headers={"PRIVATE-TOKEN": API_TOKEN})
with urllib.request.urlopen(req) as resp:
    notes = json.loads(resp.read().decode())

if any("OpenCodeReview" in n.get("body", "") for n in notes):
    print("OCR already reviewed this MR. Skipping to save tokens.")
    sys.exit(0)

# ...otherwise call `ocr review --runner codex ...` as usual and write the JSON to
# the file the posting step expects.
```

Чтобы принудительно выполнить повторное ревью, удалите предыдущие заметки OCR
из MR. При следующем запуске конвейер не найдёт заметок OCR и продолжит работу.

#### Локальный GitLab

Изменения кода не требуются. Скрипт публикации читает `CI_SERVER_URL` (GitLab
автоматически задаёт его на каждом runner), поэтому без дополнительной
настройки обращается к вашему экземпляру. Убедитесь, что `GITLAB_API_TOKEN`
выпущен вашим локальным экземпляром, а не `gitlab.com`.

#### Публикация от имени сервисной учётной записи {#post-under-a-service-account-identity}

По умолчанию обсуждения ревью публикуются от имени пользователя, которому
принадлежит `GITLAB_API_TOKEN`. Замените его сервисной учётной записью проекта,
чтобы использовать брендированное имя бота вроде `OpenCodeReview Bot`.

1. **Создайте сервисную учётную запись** в разделе
   *Project → Settings → Service Accounts → New service account*. Выбранное имя
   (например, `OpenCodeReview Bot`) будет отображаться рядом с обсуждениями MR.

2. **Пригласите её в проект** в разделе *Settings → Members → Invite member*.
   Найдите имя сервисной учётной записи и назначьте роль `Developer` или
   `Maintainer`: обе имеют права, необходимые для публикации обсуждений.

3. **Выпустите токен доступа** в разделе
   *Settings → Service Accounts → (нужная учётная запись) → Add new token*. Требуемая
   область: `api`. Немедленно скопируйте токен: GitLab показывает его только
   один раз.

4. **Замените значение токена** в разделе *Settings → CI/CD → Variables*:
   замените текущее значение `GITLAB_API_TOKEN` токеном сервисной учётной записи
   (имя переменной оставьте прежним).

Теперь обсуждения будут публиковаться от имени сервисной учётной записи, а не
пользователя, который изначально создал токен.

### Устранение неполадок

| Симптом | Причина / исправление |
|---|---|
| `Cannot find merge-base` | Runner использовал неглубокое клонирование. Исходный конвейер задаёт `GIT_DEPTH: 0`, чтобы принудительно получить полную копию; сохраните эту настройку при редактировании файла. |
| `API error 403` при публикации | У `GITLAB_API_TOKEN` нет области `api`, токен не принадлежит участнику проекта или, для локального GitLab, выпущен другим экземпляром. Перевыпустите токен с областью `api` и снова добавьте его в *Settings → CI/CD → Variables*. |
| `Failed to parse OCR output` | Выбранный runner не аутентифицирован в CI. Проверьте шаг входа Codex/Claude и перезапустите job. |
| Встроенные комментарии попадают не на те строки | GitLab требует точного совпадения SHA для встроенных обсуждений; скрипт публикации получает метаданные `versions`, чтобы использовать правильные `base_sha` / `start_sha` / `head_sha`. Если замечание всё равно не удаётся привязать, оно публикуется как обычная заметка MR. |

Конвейер записывает исходный JSON ревью в `/tmp/ocr-result.json`, а stderr — в
`/tmp/ocr-stderr.log`. Выведите их на отладочном этапе, чтобы проверить ответ
OCR:

```yaml
script:
  - cat /tmp/ocr-result.json
  - cat /tmp/ocr-stderr.log
```

## См. также

- [Справочник CLI](../cli-reference/#json) — формат вывода JSON, используемый обоими конвейерами; полезен при написании собственного скрипта CI с нуля.
- [Конфигурация](../../configuration/) — все переменные окружения и ключи конфигурации, поддерживаемые OCR.
