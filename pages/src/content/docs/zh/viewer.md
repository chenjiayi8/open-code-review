---
title: 会话查看器
sidebar:
  order: 10
---

`ocr viewer` 以浏览器 UI 展示 `~/.opencodereview/sessions/` 中保存的 review session。它只读取本地 JSONL session 文件，不会访问 GitHub、GitLab、Codex 或 Claude。

## 启动查看器

```bash
ocr viewer
```

命令会打印一个本地 URL，并持续运行直到你停止它。Session 会在每次请求时从磁盘惰性扫描，因此仍在运行的 review 可能显示为 partial session。

## 页面

| 路由 | 用途 |
|---|---|
| `/` | 列出有 session 的仓库。 |
| `/r/{repo}` | 按时间倒序列出某仓库的 session。 |
| `/r/{repo}/{sessionID}` | 展示一个保存的 session。 |

Session 详情基于实际持久化记录：

- `session_start` 记录提供 repo、branch、mode、range、model 和 start time。
- `review_item_done`、`review_item_reused`、`review_item_failed` 记录提供每个文件的完成、复用、失败和评论数量。
- 最终 `session_end` 记录可能包含 `run_manifest`；对支持 manifest 的 review session，这是 selected/completed/reused/failed/waived coverage summary 的权威来源。

请把 viewer 当作 session/coverage 浏览器。若运行是 partial 或 interrupted，优先使用存在的 `session_end` manifest；否则用 item records 作为 legacy checkpoint evidence。

## 常见检查

### 某文件没有评论

检查该文件是否在 session items 中显示为 completed/reused，或在 `run_manifest` 中计入 completed/reused。若 selected 文件显示 failed，或在 manifest-backed `session_end` 中缺失，应视为 incomplete，而不是 clean。

### 运行被中断

没有 `session_end` 的 JSONL 是 partial。使用 `ocr review --resume <session-id>`，或查看 item records 确认哪些文件已有 checkpoint。

### 程序化输出

CI 和 dashboard 请优先使用 `ocr review --runner codex --format json --audience agent` 或 `ocr session show --json <session-id>`。Viewer 面向人工渲染本地 session evidence。

## 存储位置

```text
~/.opencodereview/sessions/
  <encoded-repo-path>/
    <session-id>.jsonl
```

代表性 JSONL 记录如下：

```json
{"type":"review_item_done","filePath":"src/foo.go","comments":[{"path":"src/foo.go","content":"..."}]}
{"type":"review_item_failed","filePath":"src/bar.go","error":"runner timeout"}
{"type":"session_end","run_manifest":{"schema_version":"ocr.run-manifest/v1","run_id":"session-123","operation":"review","terminal_state":"partial","coverage":{"selected":[{"item_id":"...","path":"src/foo.go","fingerprint":"..."},{"item_id":"...","path":"src/bar.go","fingerprint":"..."}],"completed":[{"item_id":"...","path":"src/foo.go","fingerprint":"..."}],"reused":[],"failed":[{"item_id":"...","path":"src/bar.go","fingerprint":"...","classification":"timeout","reason":"runner timeout"}],"waived":[]},"elapsed_ms":8421}}
```

如需释放磁盘空间，删除整个 session 文件即可。Viewer 会从剩余文件重建索引。

## 另见

- [架构](../architecture/)——runner 选择、manifest coverage 与 session。
- [工具](../tools/)——runner 参数和定制边界。
