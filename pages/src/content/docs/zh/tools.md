---
title: 工具
sidebar:
  order: 9
---

Review 和 scan 的模型工作委托给所选本地 runner：Codex CLI 或 Claude Code CLI。OCR 负责确定性选择、校验、覆盖率和 session 输出。

## Runner 选择

```bash
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

非 preview 的 `ocr review` 和 `ocr scan` 必须指定 `--runner`。Preview 是只读预检，不会调用 runner。`--runner-model <name>` 仅作为本次运行传给所选 runner 的模型提示。

## 权限与范围

OCR 为一次 review 或 scan 启动一个已认证的本地 runner 进程。嵌入式 prompt 要求 runner 只读查看仓库并返回严格 JSON。OCR 没有用于新增、删除或重定义模型工具的 CLI 开关。

- Codex 通过已安装的 Codex CLI 运行。
- Claude 通过已安装的 Claude Code CLI 运行，并按只读 `Read`、`Glob`、`Grep` 权限预期工作。
- 登录、API token、quota、retry 和模型侧工具行为属于所选 runner CLI。

`--timeout <minutes>` 限制整体 runner 进程。运行过大时，请用 review range、`--exclude` 或 scan `--path` 缩小 selection。

## 支持的定制

| 需求 | 使用 |
|---|---|
| 选择 CLI | `--runner codex` 或 `--runner claude` |
| 单次模型提示 | `--runner-model <name>` |
| 限制时长 | `--timeout <minutes>` |
| 添加背景 | `--background` / `--background-file` |
| 改变规则 | `--rule <file>` |
| 缩小范围 | `--exclude`、review range、scan `--path` |

修改 runner prompt、JSON schema 或权限策略需要改源码并重新构建；这些不是运行时配置参数。
