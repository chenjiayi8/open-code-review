# OpenCode integration

This integration exposes OpenCodeReview as native tools and slash commands in [OpenCode](https://opencode.ai/).

It registers:

- `ocr_review` — review workspace changes, one commit, or a ref range and return structured JSON findings.
- `ocr_health` — show the installed OCR version and run runner-free preview/scope validation.
- `/ocr-review` and `/ocr-health` — convenient prompts that invoke the tools.

## Prerequisites

Install OCR and authenticate a local runner first using that CLI's supported native login or API-token mechanism. OCR does not configure or use runner credentials; Codex or Claude owns authentication.

```bash
npm install -g @alibaba-group/open-code-review
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

## Install globally

```bash
mkdir -p ~/.config/opencode/plugins
curl -fsSL \
  https://raw.githubusercontent.com/alibaba/open-code-review/main/plugins/open-code-review/opencode/open-code-review.ts \
  -o ~/.config/opencode/plugins/open-code-review.ts
```

Restart OpenCode after installation.

## Install for one project

Run this from the project root:

```bash
mkdir -p .opencode/plugins
curl -fsSL \
  https://raw.githubusercontent.com/alibaba/open-code-review/main/plugins/open-code-review/opencode/open-code-review.ts \
  -o .opencode/plugins/open-code-review.ts
```

Commit the plugin file if the integration should be shared with the project.

## Usage

Use the registered commands:

```text
/ocr-review current workspace; focus on authentication regressions
/ocr-review compare main to feature/auth-refresh
/ocr-health
```

`ocr_review` must select `--runner codex` or `--runner claude` for LLM work. `--runner-model <name>` is optional per run. Set `preview` to `true` to inspect which files would be reviewed without invoking a runner.

## Behavior and safety

- Reviews use `--audience agent` and JSON output.
- The process is launched with an argument array and `shell: false`.
- Reviews have a 15-minute overall timeout and a 10 MiB output limit.
- Cancelling the OpenCode tool terminates the OCR process.
- Runner authentication remains in the installed Codex or Claude CLI.
- CI authentication is separate from local runner authentication.
- Workspace mode includes staged, unstaged, and untracked files.

## Development

```bash
cd plugins/open-code-review/opencode
npm install
npm run check
```
