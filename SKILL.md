---
name: recall
description: "Unofficial demo CLI for Recall.ai. Send bots into Zoom, Google Meet, and Microsoft Teams calls to record and transcribe them. Not affiliated with Recall.ai."
author: "ühh"
license: "MIT"
argument-hint: "<command> [args]"
allowed-tools: "Read Bash"
metadata:
  openclaw:
    requires:
      bins:
        - recall
---

# recall: unofficial Recall.ai demo CLI

Unofficial demo. Not affiliated with Recall.ai. This skill drives the `recall` binary built from
this repository. It is not published anywhere; there is no package manager install.

## Prerequisites: build the CLI

This is a local-only demo repository, not a published package. Build it from source:

```bash
cd <path-to-this-repo>
go build -o recall ./cmd/recall
```

Verify: `./recall --version`. Put the resulting binary on `$PATH`, or invoke it by its built path
for the rest of this skill.

## Command reference

**bots**: send bots into meetings, check on them, and pull what they captured.

- `recall bots create --meeting-url <url> [--name <name>] [--join-at <time>] [--transcribe]`: send a bot into a meeting.
- `recall bots list [--status <s>] [--platform <p>] [--meeting-url <url>]`: list bots, most recent first.
- `recall bots get <id>`: get a single bot by ID, including its recordings and status history.
- `recall bots leave <id>`: remove a bot from its call without deleting its recording.
- `recall bots transcript <id>`: fetch the transcript for a bot's most recent recording, if one is ready. Not a single Recall endpoint: chains `GET /api/v1/bot/{id}/` with a direct fetch of the transcript download URL Recall returns.

**recordings**: list the recordings your bots have captured.

- `recall recordings [--bot-id <id>] [--status <s>]`: list recordings, most recent first.

**auth**: check or manage the API key.

- `recall auth status`: report whether `RECALL_API_KEY` is set and where it came from. No network call.
- `recall auth set-token <token>`: save a key to the local config file.
- `recall auth logout`: clear stored credentials.

**doctor**: `recall doctor` runs a health check: config, auth, resolved region, and live
connectivity to Recall.ai's base URL, in one command. Safe with no key.

## Auth setup

```bash
export RECALL_API_KEY="<your-key>"
recall auth status
recall doctor
```

Recall's own documentation states the `Token` prefix on the `Authorization` header is optional
and may be omitted; this CLI sends the raw key value. To persist a key instead of exporting it
every session: `recall auth set-token <token>` (writes to `~/.config/recall/config.toml`).

## Regions

Recall.ai deploys one full stack per region: `us-east-1` (default), `us-west-2`, `eu-central-1`,
`ap-northeast-1`. A key, bot, or recording created in one region does not exist in another.

```bash
recall bots list --region eu-central-1
# or
RECALL_REGION=eu-central-1 recall bots list
```

Precedence: `--region` flag, then `$RECALL_REGION`, then `us-east-1`. `$RECALL_BASE_URL`, if set,
always wins over both (used for pointing the CLI at a local mock server during testing).

## Agent mode

Add `--agent` to any command. Expands to `--json --compact --no-input --no-color --yes`.

- Pipeable: JSON on stdout, errors on stderr.
- Filterable: `--select` keeps a named subset of fields, for example
  `recall bots list --agent --select id,meeting_url,status_changes`.
- Previewable: `--dry-run` prints the exact request (method, URL, headers, body) and exits 0
  without sending anything. Works with no API key at all.
- Non-interactive: never prompts; every input is a flag or a positional argument.

### Response envelope

Read commands (`bots list`, `bots get`, `recordings`) wrap output in a small envelope when piped
or run with `--json`:

```json
{
  "meta": {"source": "live"},
  "results": <data>
}
```

`source` is always `"live"`: this demo CLI has no local data store, so every read hits the API (or
the mock server under `--dry-run`/`verify`). Parse `.results` for the data.

## Paths and state

- `--home <dir>` relocates config, data, state, and cache for one invocation; `RECALL_HOME`
  relocates them for a session or a fleet.
- Resolution order: per-kind env var (`RECALL_CONFIG_DIR`, etc.), `--home`, `RECALL_HOME`, XDG
  vars, then platform defaults.
- Credentials saved via `auth set-token` live in `credentials.toml` under the data dir, not in
  `config.toml`. Run `recall doctor` to see the resolved paths and credential source.

## Output delivery

Every command accepts `--deliver <sink>`. Output goes to the named sink in addition to (or
instead of) stdout.

| Sink | Effect |
|------|--------|
| `stdout` | Default: write to stdout only |
| `file:<path>` | Atomically write output to `<path>` (temp file plus rename) |
| `webhook:<url>` | POST the output body to the URL |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Usage error (wrong arguments) |
| 3 | Resource not found |
| 4 | Authentication required |
| 5 | API error (upstream issue) |
| 7 | Rate limited (wait and retry) |
| 10 | Config error, including an unrecognized `--region` value |

## Direct use

1. Check the binary exists: `recall --version`. If not found, build it (see Prerequisites above).
2. Match the request to a command from the Command Reference above.
3. Execute with `--agent` for scripted or agent-driven use:
   ```bash
   recall <command> [subcommand] [args] --agent
   ```
4. With no `RECALL_API_KEY` available, add `--dry-run` instead of guessing at behavior: it shows
   the exact request the live call would make, with zero network risk.

## What this demo does not include

Not a limitation to work around, an intentional scope cut for a 7 command outreach demo: no
offline sync and search store, no self-teaching query cache, no MCP server, no pagination beyond
one page. See the README's Known Gaps section for the full list and why.
