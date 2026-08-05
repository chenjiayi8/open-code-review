---
title: QuickStart
sidebar:
  order: 3
---

几分钟内完成第一次代码审查。

## 前置条件

- **Git ≥ 2.41**
- **Node.js ≥ 18**
- **已用支持的原生方式认证的 Codex CLI 或 Claude Code CLI**

## 第 1 步 — 安装 CLI

```bash
npm install -g @alibaba-group/open-code-review
ocr version
```

## 第 2 步 — 认证本地 runner

OCR 将 LLM 工作交给已安装的本地 CLI。认证由该 CLI 支持的原生方式（已有登录或 API token）负责；OCR 不保存 runner 凭据。

```bash
codex login                 # or use the runner's supported API-token auth
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

`ocr review` 与 `ocr scan` 必须传入 `--runner`。需要为单次运行覆盖 runner 默认模型时，可选使用 `--runner-model <name>`。`--preview` 等只读/预检命令不会调用 runner，可在认证前运行。

## 第 3 步 — 运行第一次审查

```bash
cd path/to/your-repo
ocr review --runner codex
ocr review --runner codex --from main --to feature-branch
ocr review --runner codex --commit abc123
ocr scan --runner claude --path internal/agent
```

如果 OCR 报告 runner 认证失败，请先用所选 CLI 的原生登录或 API token 机制完成认证再重试。CI 认证与本地状态相互独立；在 CI 任务中认证 Codex 或 Claude，而不要为 OCR 配置供应商密钥。

## 参见

- [安装](../installation/)
- [配置](../configuration/)
- [CLI 参考](../cli-reference/)
- [评审规则](../review-rules/)
