---
title: Configuration
sidebar:
  order: 5
---

OCR 不再保存供应商端点、模型或 API 密钥。审查执行由本地订阅支持：OCR 调用已安装的本地 runner，认证归该 runner 管理。

## 本地订阅 runner

先登录要让 OCR 使用的 CLI：

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`--runner` 支持 `codex` 与 `claude`。`ocr review` 和 `ocr scan` 必须传入该参数，因为 OCR 需要知道由哪个本地订阅 CLI 执行 LLM 工作。`--runner-model <name>` 是可选参数，只影响当前调用。

## 只读与预检行为

`ocr review --preview` 等只读/预检命令只检查文件和规则，不调用 runner。需要 runner 的命令会先预检所选 CLI；如果 CLI 缺失或未登录，OCR 会在开始审查前失败。请运行对应登录命令后重试。

## CI 认证

本地订阅登录不会自动带到 CI。CI 工作流必须安装 OCR，并在任务内认证所选 Codex 或 Claude runner。不要向 OCR 添加供应商端点或 API 密钥配置命令；CI runner 认证应使用 CI 平台支持的机制。
