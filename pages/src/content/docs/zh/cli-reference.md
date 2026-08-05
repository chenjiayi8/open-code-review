---
title: CLI Reference
sidebar:
  order: 6
---

本页说明使用本地订阅 runner 的审查命令。

## 认证

OCR 不管理 runner 凭据或供应商连接设置。先登录已安装的 runner CLI：

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

## 命令摘要

| 命令 | 别名 | 作用 |
|---|---|---|
| `ocr review` | `ocr r` | 审查工作区、commit 或 ref 范围。 |
| `ocr scan` | `ocr s` | 不依赖 Git diff，审查完整文件。 |
| `ocr rules check <file>` | — | 显示文件命中的评审规则。 |
| `ocr session list` | — | 列出保存的审查会话。 |
| `ocr session show <id>` | — | 查看一个保存的会话。 |
| `ocr session comments <id>` | — | 输出保存会话中的评论。 |
| `ocr viewer` | — | 启动本地会话查看器。 |
| `ocr version` | — | 输出版本与构建信息。 |

## 必需 runner 参数

非 preview 的 `ocr review` 和 `ocr scan` 必须传入 `--runner codex` 或 `--runner claude`。`--runner-model <name>` 可选，只传给当前选择的 runner。OCR 会预检所选 runner；如果 CLI 不存在或未登录，会在开始审查前失败。

`ocr review --preview` 与 `ocr scan --preview` 等只读/预检命令只检查文件和规则，不调用 runner。

## `ocr review` 示例

```bash
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr review --runner codex --runner-model gpt-5-codex --format json
ocr review --preview
```

常用参数包括 `--repo`、`--runner`、`--runner-model`、`--from`、`--to`、`--commit`、`--resume`、`--format`、`--audience`、`--background`、`--background-file`、`--timeout`、`--rule`、`--exclude` 与 `--max-git-procs`。

## `ocr scan` 示例

```bash
ocr scan --runner claude
ocr scan --runner claude --path internal/agent
ocr scan --runner claude --runner-model sonnet --format json
ocr scan --preview --path internal/agent
```

`ocr scan` 审查完整文件。它接受共享的输出、runner、超时、规则、排除和恢复参数，也支持 `--path`、`--no-plan`、`--no-dedup`、`--no-summary` 等扫描开关。

## 本地订阅与 CI

本地登录状态属于该机器上的 Codex 或 Claude CLI。CI 任务必须在 CI 内认证所选 runner；不要添加 OCR 供应商凭据设置。
