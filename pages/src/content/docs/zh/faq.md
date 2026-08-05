---
title: FAQ
sidebar:
  order: 14
---

常见错误、意外与“这应该这样吗？”的问题。若你的问题不在此处，开一个带运行步骤
与完整输出的 [GitHub issue](https://github.com/alibaba/open-code-review/issues)。

## 配置与启动

### `runner subscription authentication required`

OCR 的 review 和 scan 命令使用已安装的本地订阅 runner。非 preview 运行必须传入 `--runner codex` 或 `--runner claude`，并先登录对应 CLI：

```bash
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
ocr scan --runner claude --path internal/agent
```

如果 OCR 报告需要 runner 订阅认证，说明所选 runner 缺失、未登录，或在当前环境不可用。安装/登录该 runner 后重试。不要发明 OCR 自有的 runner 凭据变量；请使用所选 runner 的官方登录流程认证。

### Preview 可用但 review 失败

`ocr review --preview` 是只读命令，不调用 runner。完整 review 仍可能因为缺少 `--runner` 或所选 Codex/Claude CLI 未认证而失败。登录后用 `--runner codex` 或 `--runner claude` 重新运行。

### Runner 返回认证失败

登录、订阅、额度或权限错误来自所选 Codex 或 Claude CLI。请在同一个 shell 或 CI job 中重新认证该工具后重试 OCR。在 CI 中，先用 CI 平台支持的 secret/OIDC/device-flow 机制认证 runner，再运行 `ocr review --runner ...`。

### `not a git repository`

`ocr review --runner codex` 对当前目录运行 `git diff`（以及对 untracked 文件的 `git ls-files`）。
若你不在 Git 工作树内，它会提前退出。要么 `cd` 进仓库，要么传 `--repo /path/to/repo`。

### 本地 runner 报告工具、额度或超时错误

OCR 现在把模型执行交给已登录的 Codex 或 Claude CLI。若 runner 报告工具、额度、重试或超时问题，请先确认 runner 登录状态，再用 `--path`、`--exclude` 或更小的 diff 缩小范围；如果确实需要更长时间，可提高 `--timeout <minutes>`。

## 过滤与规则

### 我的文件没被评审

运行 `ocr review --preview`（无 LLM 成本）。输出列出每个候选文件及其被保留或
丢弃的**原因**：

```
src/foo.go              modified
src/foo_test.go         modified  (excluded: user_exclude)
node_modules/lib.js     added     (excluded: default_path)
imgs/logo.png           binary    (excluded: unsupported_ext)
```

五种排除原因对应[文件过滤](../review-rules/#how-files-are-filtered)中的门：

| 原因 | 修复 |
|---|---|
| `binary` | 无需处理——二进制文件无可评审文本。 |
| `user_exclude` | 从你的 `exclude` 列表移除该模式。 |
| `unsupported_ext` | 把扩展名加入你的 `include` 列表以绕过白名单门。 |
| `default_path` | 把文件加入 `include`——那会覆盖内置测试文件排除模式。 |
| `deleted` | 无需处理——没有新内容可评审。 |

### 我的自定义规则没触发

运行 `ocr rules check <file-path>`。它会完整打印匹配的**层**与 **glob 模式**：

```
File: src/api/UserHandler.go
Source: Project (.opencodereview/rule.json)
Pattern: src/api/**/*.go
Rule: …
```

若层不对（如期望项目规则却显示 "System built-in"），多半是**声明顺序**问题——
首条匹配模式生效。把更具体的规则在 `rules` 数组里前移，或修正 glob。

### 花括号展开不工作

`bmatcuk/doublestar/v4` 支持 `{ts,tsx}` 花括号。若不匹配，检查多余空格——
`{ts, tsx}` 带空格会静默地无法匹配 `tsx`。

## 评审

### 某文件显示零评论——它真的被评审了吗？

打开[会话查看器](../viewer/)（`ocr viewer`）并找到对应 session。零评论文件只有在出现在 `reviewed_files` 且没有 validation warning 时才表示干净评审。如果 coverage 中缺失，请检查它是评审前被过滤，还是本地 runner 返回了不完整输出。

### 评论的 `start_line: 0` 和 `end_line: 0`

OCR 无法把评论锚定到 diff 中的精确行。两个常见原因：

- 模型改写了 `existing_code` 而非从 diff 原样复制。模型被告知不要这样做，但偶尔
  仍会如此。
- diff 有异常格式（CRLF、tab/空格混用）破坏了滑动窗口匹配。

评论仍是真实的——只是没被自动放置。多数 agent 集成（SKILL、Claude Code
plugin）读 `existing_code` 字段并自行在文件中定位。

### Token threshold exceeded

```
[ocr] WARNING: prompt tokens (94000) exceed 80% of max_tokens(58888) for src/big.sql
```

该文件的初始 prompt（规则 + diff + change-files 列表）在模型能响应之前就已超过
`MAX_TOKENS = 58888` 的 80 %。OCR 跳过该文件并继续——JSON 模式下你也会在
`warnings` 中看到。

缓解：

- 若是自动生成的，把文件加入 `exclude` 列表。
- 把大重构拆成更小的 commit。
- 对一系列小 commit 用 `--commit` 模式，而非一次性工作区模式评审。

### plan 阶段花了很久而文件很小

先运行 `ocr review --preview`。若文件的 `lines.changed` 超过
`PLAN_MODE_LINE_THRESHOLD`（默认 **50**），plan 阶段会运行。这是有意为之——大
diff 能从 plan 中受益。要为单次评审跳过它，用更小 diff 运行，或临时编辑内嵌模板
（高级；需要修改内嵌 runner prompt 并重新构建 OCR）。


### 本地 runner 超时或未覆盖所有文件

OCR 以 runner 返回的 `reviewed_files` 作为覆盖证据。缺失的文件会在 session 输出中标记为未完成。遇到超时时，请缩小选择范围或提高 `--timeout <minutes>`。

### 一些文件未完成；运行仍可能产生结果

OCR 会保留已成功验证的评论，并在 session/JSON 输出中记录未覆盖或失败的文件。检查 `warnings` 和 session 明细即可知道哪些路径需要重跑。

### CI 运行比本地慢得多

CI 环境通常需要重新安装并认证所选 runner，且没有本地缓存。确认 CI job 在运行 OCR 前已登录 Codex 或 Claude；若 runner 报告 quota/timeout，请缩小范围或使用 `--timeout <minutes>`。

## 输出与集成

### `--audience agent` 仍有进度行

确认你看到的不是 **stderr**。进度消息偶尔会到 stderr（警告、错误）。`--audience
agent` 保证的干净 stdout 是*对解析器友好的*——要屏蔽一切，重定向：
`ocr review --runner codex --audience agent 2>/dev/null`（先认证所选 runner）。

### JSON 输出是 `{ "files_reviewed": 0, "comments": [] }`

工作区没有合格文件。这是有意为之——显式形状让调用方区分“无可评审内容”与“已评审
文件中无发现”。零评论的正常评审产出的是普通空数组 `[]`。

### 会话 JSONL 在哪？

```
~/.opencodereview/sessions/<path-encoded-repo-path>/<session-id>.jsonl
```

仓库路径通过把 `/` 和 `\` 替换为 `-`、`:` 替换为 `_` 编码
（如 `/Users/foo/my-repo` → `Users-foo-my-repo`）。用 `ocr viewer` 浏览会话。
删除该目录清除历史；OCR 在下次运行时重新生成编码路径。

## 性能与成本

### 怎么知道哪些 token 花了多少？

启用遥测：

```bash
ocr config set telemetry.enabled true
ocr config set telemetry.exporter console
codex login                 # or: claude auth login --claudeai
ocr review --runner codex
```

LLM 调用没有自己的 span——它们记为 metric。关注 `ocr.llm.tokens_used`
（counter，标 `model` + `type`）、`ocr.llm.requests_total`（counter，标 `model`
+ `status`）、`ocr.llm.request_duration_seconds`（histogram，标 `model`）。
console exporter 会内联打印这些聚合。如需仪表盘，切换到 OTLP exporter 并发到你的
metrics 体系——见[遥测](../telemetry/)。

### 如何减少 runner 工作量？

- 添加 `include`/`exclude` 规则，避免评审无关文件。
- 对大型改动使用更小的 commit、范围或 `--path`。
- 传 `--background`，让 runner 一开始就获得必要业务上下文。

## 隐私与安全

### OCR 会把我的代码发到别处吗？

OCR 把你的 **diff**（及可选 read-tool 片段）交给所选且已认证的本地 runner。其余任何内容都不
离开你的机器——会话 JSONL 与规则文件仅存于本地。

若启用遥测，`content_logging` 标志已接入配置层但目前**不**控制任何代码路径——
无论该标志值如何，prompt 与响应内容绝不导出到你的 collector。请视为保留位。生产
环境保持 `false`。详情见[遥测](../telemetry/#content-logging)。

### 我能在发给 LLM 前脱敏 secret 吗？

非内置功能。推荐工作流：

1. 不要把 secret 提交到仓库（常规规则）。
2. 把已知含敏感信息的文件加入 `exclude`。
3. 用 `git diff --no-textconv` 过滤器或 pre-commit 脱敏，使 secret 不进入 diff。

“脱敏规则”功能在路线图上；关注
[issue 跟踪器](https://github.com/alibaba/open-code-review/issues)。

## 杂项

### changelog 在哪？

[GitHub Releases](https://github.com/alibaba/open-code-review/releases)
——每个 release 都附带从 Conventional Commits 生成的 notes。

### OCR 支持非 Git VCS 吗？

不支持。diff provider 通过 shell 调用 `git`。SVN / Mercurial 等需要新的 provider；Hg 支持的
issue 已在[此](https://github.com/alibaba/open-code-review/issues)开放。

### 为什么二进制叫 `opencodereview` 而 CLI 是 `ocr`？

release 中发布的静态二进制以项目命名（`opencodereview`）；NPM wrapper 为了便于使用而安装为 `ocr`。从源码构建得到 `dist/opencodereview`——复制为 `$PATH` 上的
`ocr`。

### 如何卸载？

```bash
npm uninstall -g @alibaba-group/open-code-review        # NPM install
sudo rm /usr/local/bin/ocr                              # binary install
rm -rf ~/.opencodereview                                # all state
```

OCR 不在 `~/.opencodereview` 之外写入（NPM 下载二进制除外），因此删除该目录即可
清除历史、配置与每用户规则。

## 另见

- [配置](../configuration/)——runner 选择与 config key。
- [评审规则](../review-rules/)——文件过滤器与规则解析链。
- [会话查看器](../viewer/)——查看历史评审会话。
- [遥测](../telemetry/)——token 用量与 LLM 指标。
