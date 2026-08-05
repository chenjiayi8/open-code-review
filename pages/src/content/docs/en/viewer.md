---
title: Session Viewer
sidebar:
  order: 10
---

`ocr viewer` is a small embedded HTTP server that renders past review
sessions in a browser-friendly UI. No external dependencies — sessions
are read directly from the JSONL files OCR writes to disk during every
review.

## Launching

```bash
ocr viewer                  # binds localhost:5483
ocr viewer --addr :3000     # bind to all interfaces on port 3000
ocr viewer --addr 0.0.0.0:8080   # bind on all interfaces
```

The default address is `localhost:5483`. The server holds the foreground
— `Ctrl+C` stops it. Sessions are scanned lazily from
`~/.opencodereview/sessions/` on each request, so a review running in
another terminal shows up the moment its JSONL file appears.

> **DNS-rebinding protection.** The viewer checks the `Host` header
> against a loopback allowlist (`localhost`, `127.0.0.1`, `::1`). A
> concrete bind host (e.g. `--addr 192.168.1.10:5483`) is added
> automatically, but **wildcard** binds (`:3000`, `0.0.0.0`, `::`) are
> not — reaching the UI from a LAN IP or hostname then returns
> `forbidden host`. To expose a wildcard bind, set
> `OCR_VIEWER_ALLOWED_HOSTS` to a comma-separated list of allowed
> hostnames (e.g. `OCR_VIEWER_ALLOWED_HOSTS=box.local,192.168.1.10`).

## Three pages

The viewer has three URLs:

| URL | What you see |
|---|---|
| `/` | List of all repositories that have sessions on disk. |
| `/r/{repo}` | List of sessions for one repository, newest first. |
| `/r/{repo}/{sessionID}` | Full detail for a single session. |

`{repo}` is a path-encoded string (separators `/` and `\` replaced with
`-`, colons replaced with `_` — the same encoding used to name the
on-disk directories). You don't usually type this — you click through.

### `/` — Repository list

For each repo with at least one session you see the repo path, the
total session count, and the most recent activity timestamp.

### `/r/{repo}` — Session list for one repo

For each session: ID (a UUID), branch name (when OCR was able to
detect it), review mode, model, file count, duration, and a started-at
timestamp.

### `/r/{repo}/{sessionID}` — Session detail

The detail page is the interesting one. It shows:

1. **Header** — diff range, model, branch, total tokens, run duration.
2. **Selection and result groups** — manifest entries, runner invocation metadata, validation warnings, coverage status, and final comments.

The detail page is organized around the local runner process and the deterministic validation that follows it. Use coverage and warnings to see which selected files were reviewed, skipped, or returned incomplete by the runner.

## What's in a task card

Expand the runner or validation record to inspect:

- a **header row** — request number, model badge, a token badge
  (`P:` prompt / `C:` completion, plus `CR:` / `CW:` cache read/write
  when present), a duration badge, and an error badge when the round
  failed;
- **Response** — the raw assistant response, including any reasoning /
  `thinking` blocks;
- **Validation** — coverage, path, line-range, and JSON-shape warnings.

The full runner prompt and raw response are intentionally kept in the JSONL transcript; inspect the saved session when you need exact evidence.

## Use cases

The viewer is designed around three workflows:

### "Why did the model say that?"

Open a comment in your terminal output, locate the session in the viewer, and inspect the runner response plus OCR validation records. For the exact prompt and raw runner output, open the corresponding JSONL session records.

### "Why was this file silent?"

A file with **no comments** is successful only when it appears in `reviewed_files` and has no validation warning. If the file is absent from coverage or the runner failed, surface it as a warning.

### "What did compression keep / drop?"

Use the runner response, coverage records, and validation warnings to debug missing context or incomplete output. The JSONL transcript is the source of truth for the selected manifest and raw runner result.

## Storage layout on disk

The viewer reads from:

```
~/.opencodereview/sessions/
└── <path-encoded-repo-path>/
    └── <session-id>.jsonl
```

Each line in the JSONL file is one event:

```json
{"type": "llm_request", "filePath": "src/foo.go", "taskType": "runner", "request_no": 1, "messages": [{"role": "user", "content": "Review this diff…"}], "timestamp": "2026-06-02T10:15:23Z"}
{"type": "llm_response", "filePath": "src/foo.go", "taskType": "runner", "model": "claude-sonnet-4-6", "content": "Found 2 issues…", "duration_ms": 8421, "usage": {"prompt_tokens": 12450, "completion_tokens": 320}}
{"type": "tool_call", "filePath": "src/foo.go", "tool_name": "file_read", "arguments": "{\"file_path\":\"src/foo.go\",\"start_line\":1,\"end_line\":50}", "result": "File: src/foo.go (Total lines: 220)\nIS_TRUNCATED: false\nLINE_RANGE: 1-50\n1|package foo…", "ok": true, "duration_ms": 14}
```

Lines are append-only — a partial JSONL means a session was killed
mid-run, and the viewer renders what it has.

To free disk space, delete entire session files; the viewer regenerates
its index on the next request.

## Privacy

The JSONL transcripts contain **everything** sent to and received from
the LLM, including any code that was in the diff. They live entirely on
your machine inside `~/.opencodereview/`. OCR does not upload them
anywhere.

If your reviews include code you wouldn't want stored long-term,
either:

- delete the session files periodically, or
- redirect `--audience agent --format json` output to a transient pipe
  in CI and run with a temporary `HOME` so the JSONL never persists.

The OpenTelemetry exporter is a separate concern — see
[Telemetry](../telemetry/) for how to keep prompt content out of
exported traces.

## When the viewer is not the right tool

- For programmatic post-processing (CI, dashboards), use
  `ocr review --runner codex --format json --audience agent`. The viewer renders for
  humans, not machines.
- For grepping across many sessions, use `jq` on the JSONL files
  directly. There's no search box in the UI yet.

## See Also

- [Architecture](../architecture/) — what those five task types
  actually do under the hood.
- [Tools](../tools/) — runner selection, permissions, and customization boundaries.
