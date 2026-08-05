---
title: QuickStart
sidebar:
  order: 3
---

几分钟内完成第一次代码审查。

## 前置条件

- **Git ≥ 2.41**
- **Node.js ≥ 18**
- **已用本地订阅登录的 Codex CLI 或 Claude Code CLI**

## 第 1 步 — 安装 CLI

```bash
npm install -g @alibaba-group/open-code-review
ocr version
```

## 第 2 步 — 登录本地 runner

OCR 将 LLM 工作交给已安装的本地 CLI。订阅认证由本地 CLI 负责；OCR 不保存 runner 凭据。

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`ocr review` 与 `ocr scan` 必须传入 `--runner`。需要为单次运行覆盖 runner 默认模型时，可选使用 `--runner-model <name>`。`--preview` 等只读/预检命令不会调用 runner，可在登录前运行。

## 第 3 步 — 运行第一次审查

```bash
cd path/to/your-repo
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr scan --runner claude --path internal/agent
```

如果 OCR 报告预检认证失败，请先运行所选 runner 的登录命令再重试。CI 认证与本地订阅登录相互独立；在 CI 任务中登录 Codex 或 Claude，而不要为 OCR 配置供应商密钥。

## 参见

- [安装](../installation/)
- [配置](../configuration/)
- [CLI 参考](../cli-reference/)
- [评审规则](../review-rules/)
