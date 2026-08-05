---
title: Session Viewer
sidebar:
  order: 10
---

`ocr viewer` serves saved review sessions from `~/.opencodereview/sessions/` in a browser-friendly UI. It reads local JSONL session files only; it does not contact GitHub, GitLab, Codex, or Claude.

## Start the viewer

```bash
ocr viewer
```

The command prints a local URL and keeps running until you stop it. Sessions are scanned lazily from disk on each request, so a review that is still running can appear as a partial session.

## Pages

| Route | Purpose |
|---|---|
| `/` | List repositories that have saved sessions. |
| `/r/{repo}` | List sessions for one repository, newest first. |
| `/r/{repo}/{sessionID}` | Show one saved session. |

The session detail view is based on the actual persisted session records:

- `session_start` records provide repo, branch, mode, range, model, and start time.
- `review_item_done`, `review_item_reused`, and `review_item_failed` records provide per-file completion, reused prior results, failures, and comment counts.
- The final `session_end` record may embed `run_manifest`, which is the authoritative selected/completed/reused/failed/waived coverage summary for review sessions that support manifests.

Treat the viewer as a session/coverage browser. If a run is partial or interrupted, prefer the `session_end` manifest when present; otherwise use the item records as legacy checkpoint evidence.

## Common checks

### A file has no comments

Check whether the file appears as completed/reused in the session items or as completed/reused in `run_manifest`. A selected file listed as failed, or missing from a manifest-backed `session_end`, should be treated as incomplete rather than clean.

### A run was interrupted

A JSONL without `session_end` is partial. Use `ocr review --resume <session-id>` or inspect item records to see which files already have checkpoints.

### Programmatic output

For CI and dashboards, prefer `ocr review --runner codex --format json --audience agent` or `ocr session show --json <session-id>`. The viewer renders local session evidence for humans.

## Storage layout

```text
~/.opencodereview/sessions/
  <encoded-repo-path>/
    <session-id>.jsonl
```

Representative JSONL records look like:

```json
{"type":"review_item_done","filePath":"src/foo.go","comments":[{"path":"src/foo.go","content":"..."}]}
{"type":"review_item_failed","filePath":"src/bar.go","error":"runner timeout"}
{"type":"session_end","run_manifest":{"selected_count":2,"completed_count":1,"failed_count":1}}
```

Delete whole session files to free disk space. The viewer rebuilds its index from the files that remain.

## See Also

- [Architecture](../architecture/) — runner selection, manifest coverage, and sessions.
- [Tools](../tools/) — runner flags and customization boundaries.
