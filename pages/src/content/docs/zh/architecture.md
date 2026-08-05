---
title: 架构
sidebar:
  order: 8
---

本文说明 `ocr review` 和 `ocr scan` 在本地 runner 模型下的实际运行方式。OCR 负责确定性地选择文件、渲染 runner prompt、为本次 review 或 scan 启动一个已认证的 Codex 或 Claude 本地进程、校验结构化结果，并写入可复现的 session。

## 流水线

1. 校验参数并选择 `--runner codex` 或 `--runner claude`。
2. 为 review 构建 Git diff selection，或为 scan 构建 `--path` selection。
3. 应用 binary、`--exclude`、支持文件类型、内置噪声路径和规则匹配过滤。
4. 把保留文件写成 manifest，并把它作为 `reviewed_files` 覆盖率契约。
5. 调用一个本地 runner 进程。
6. 校验 JSON、路径、行号、评论结构和覆盖率，然后写入 text/JSON 输出与 session。

`ocr review --preview` 只运行前面的选择和过滤步骤，不调用 runner，因此不需要 `--runner` 或 runner 认证。

## 本地 runner 调用

```bash
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

非 preview 的 review 和 scan 必须指定 `--runner`。支持值是 `codex` 和 `claude`。OCR 会预检所选 CLI；CLI 不存在或未认证时会在开始前失败。`--runner-model <name>` 只是在当前运行中向所选 runner 传递模型提示。

OCR 每条命令只启动一个本地 runner 进程。范围由 selection 参数控制，进程时长由 `--timeout <minutes>` 限制。

## Session、coverage 与 resume

每次运行写入：

```text
~/.opencodereview/sessions/<encoded-repo-path>/<session-id>.jsonl
```

Session 记录 selection、prompt metadata、runner result、validation warnings、coverage status 和最终 comments。`--resume <session-id>` 会复用保存的 session state，使中断运行按同一 selection 和 coverage accounting 继续。

## 源码地图

| 关注点 | 文件 |
|---|---|
| 命令分发 | `cmd/opencodereview/main.go` |
| 参数解析 | `cmd/opencodereview/flags.go` |
| Review/scan 编排 | `internal/agent/` |
| Preview / 过滤 | `internal/agent/preview.go` |
| Diff 加载 | `internal/diff/git.go` |
| Runner 适配 | `internal/runner/` |
| Session 写入 | `internal/session/persist.go` |
