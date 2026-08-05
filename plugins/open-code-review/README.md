# Coding agent plugins

Open Code Review ships platform-specific integrations for Claude Code, Codex, and Cursor. All integrations call the local `ocr` CLI and require Git 2.41 or later.

Install OCR first:

```bash
npm install -g @alibaba-group/open-code-review
```

Authenticate the local subscription runner you want OCR to use. The installed runner CLI owns authentication; OCR does not configure or use runner credentials.

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` is required for review/scan execution, while `--runner-model <name>` is optional per run. Read-only/preflight commands such as `--preview` can run without invoking a runner. CI authentication is separate from local subscription login; authenticate Codex or Claude inside CI instead of storing OCR credentials.

## Claude Code

Run these commands inside Claude Code:

```text
/plugin marketplace add alibaba/open-code-review
/plugin install open-code-review@open-code-review
```

This installs the `/open-code-review:review` and `/open-code-review:delegate-review` slash commands. See the [Claude Code guide](https://open-codereview.ai/docs/claude-code) for manual installation, usage, and behavior.

## Codex

Add this repository as a Codex marketplace, then start Codex:

```bash
codex plugin marketplace add alibaba/open-code-review
codex
```

Open `/plugins`, install and enable **Open Code Review**, then start a new task. The plugin exposes callable review skills backed by the local `ocr` CLI.

## Cursor

This repository includes a Cursor plugin manifest at [`.cursor-plugin/plugin.json`](.cursor-plugin/plugin.json). For a local manual installation, copy the entire `plugins/open-code-review/` directory to:

```text
~/.cursor/plugins/local/open-code-review/
```

Verify that the manifest is located at `~/.cursor/plugins/local/open-code-review/.cursor-plugin/plugin.json`, then restart Cursor or run **Developer: Reload Window**.
