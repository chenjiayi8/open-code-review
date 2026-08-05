---
title: セッションビューア
sidebar:
  order: 10
---

`ocr viewer` は `~/.opencodereview/sessions/` に保存された review sessions を browser-friendly UI で表示します。読み取るのは local JSONL session files だけで、GitHub、GitLab、Codex、Claude には接続しません。

## ビューアを起動

```bash
ocr viewer
```

コマンドは local URL を表示し、停止するまで動き続けます。Sessions は request ごとに disk から lazy scan されるため、実行中の review は partial session として見えることがあります。

## Pages

| Route | Purpose |
|---|---|
| `/` | saved sessions を持つ repositories を一覧表示。 |
| `/r/{repo}` | 1 repository の sessions を新しい順に一覧表示。 |
| `/r/{repo}/{sessionID}` | 1 つの saved session を表示。 |

Session detail は実際に永続化された records に基づきます。

- `session_start` records は repo、branch、mode、range、model、start time を提供します。
- `review_item_done`、`review_item_reused`、`review_item_failed` records は file ごとの completion、reused prior results、failures、comment counts を提供します。
- 最終 `session_end` record には `run_manifest` が埋め込まれることがあります。manifest 対応 review sessions では、これが selected/completed/reused/failed/waived coverage summary の authoritative source です。

Viewer は session/coverage browser として扱ってください。Partial または interrupted run では、存在する場合は `session_end` manifest を優先し、なければ item records を legacy checkpoint evidence として使います。

## Common checks

### コメントがないファイル

その file が session items で completed/reused として表示されるか、`run_manifest` で completed/reused に計上されているか確認します。Selected file が failed として表示される、または manifest-backed `session_end` から欠けている場合は、clean ではなく incomplete として扱います。

### Run が中断された

`session_end` のない JSONL は partial です。`ocr review --resume <session-id>` を使うか、item records を見て checkpoint 済み files を確認します。

### Programmatic output

CI や dashboards には `ocr review --runner codex --format json --audience agent` または `ocr session show --json <session-id>` を優先してください。Viewer は humans 向けに local session evidence を render します。

## Storage layout

```text
~/.opencodereview/sessions/
  <encoded-repo-path>/
    <session-id>.jsonl
```

Representative JSONL records:

```json
{"type":"review_item_done","filePath":"src/foo.go","comments":[{"path":"src/foo.go","content":"..."}]}
{"type":"review_item_failed","filePath":"src/bar.go","error":"runner timeout"}
{"type":"session_end","run_manifest":{"schema_version":"ocr.run-manifest/v1","run_id":"session-123","operation":"review","terminal_state":"partial","coverage":{"selected":[{"item_id":"...","path":"src/foo.go","fingerprint":"..."},{"item_id":"...","path":"src/bar.go","fingerprint":"..."}],"completed":[{"item_id":"...","path":"src/foo.go","fingerprint":"..."}],"reused":[],"failed":[{"item_id":"...","path":"src/bar.go","fingerprint":"...","classification":"timeout","reason":"runner timeout"}],"waived":[]},"elapsed_ms":8421}}
```

Disk space を空けるには session files 全体を削除してください。Viewer は残った files から index を再構築します。

## See Also

- [Architecture](../architecture/) — runner selection、manifest coverage、sessions。
- [Tools](../tools/) — runner flags と customization boundaries。
