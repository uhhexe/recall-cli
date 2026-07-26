Unofficial demo. Not affiliated with Recall.ai. Built as a proposal.

# recall

A command line interface for the [Recall.ai](https://docs.recall.ai) meeting bot API. Send a bot
into a Zoom, Google Meet, or Microsoft Teams call, check on it, pull its transcript, and list what
it recorded, all from the terminal or a script.

Recall.ai does not publish an official CLI. This one was generated with an internal code
generation tool, then hand trimmed down to the seven commands below so it is easy to read end to
end in a few minutes.

## Quickstart

Three real commands. All three run with no API key, using `--dry-run` (shows the exact request
that would be sent) or the tool's spec derived mock server (returns realistic fixture responses).

```bash
# 1. Send a bot into a meeting. --transcribe turns on the meeting platform's
#    own live captions, no third party transcription key required.
recall bots create --meeting-url https://zoom.us/j/1234567890 --name "Sales Call Notetaker" --transcribe --dry-run

# 2. Check on a bot and get machine readable output.
recall bots get 550e8400-e29b-41d4-a716-446655440000 --json --dry-run

# 3. List every bot you have sent out, most recent first.
recall bots list --json --dry-run
```

Drop `--dry-run` once `RECALL_API_KEY` is set to a real key. Every command above is unchanged;
only the network call turns on.

## Install

Requires Go 1.26 or newer (see `go.mod`).

```bash
git clone <this-repo>
cd recall-cli
go build -o recall ./cmd/recall
./recall --help
```

Or run it without a permanent build:

```bash
go run ./cmd/recall --help
```

## Auth

Set `RECALL_API_KEY` to your Recall.ai API key. Recall's own docs say the `Token` prefix is
optional, so this CLI sends the key exactly as given:

```bash
export RECALL_API_KEY="your-key-here"
recall auth status
```

`recall auth status` reports whether a key is present and where it came from (environment
variable or the local config file at `~/.config/recall/config.toml`). It does not make a network
call, so it works with a fake key too. `recall auth set-token <token>` writes a key to that config
file if you would rather not export an environment variable every session.

## Regions

Recall.ai is deployed per region: `us-east-1`, `us-west-2`, `eu-central-1`, and `ap-northeast-1`.
Each is a separate account and dataset. Bots, recordings, and API keys created in one region are
not visible from another (see [docs.recall.ai/docs/regions](https://docs.recall.ai/docs/regions)).

```bash
recall bots list --region eu-central-1
# or
export RECALL_REGION=eu-central-1
recall bots list
```

Precedence: `--region` flag, then `$RECALL_REGION`, then `us-east-1` by default. An explicit
`$RECALL_BASE_URL` (used by this README's fixture tests, see below) always wins over both.

## Commands

| Command | What it does |
|---|---|
| `recall bots create --meeting-url <url> [--name <name>] [--join-at <time>] [--transcribe]` | Send a bot into a meeting, immediately or scheduled. |
| `recall bots list [--status <s>] [--platform <p>] [--meeting-url <url>]` | List bots, most recent first. |
| `recall bots get <id>` | Get one bot: status history, recordings, everything Recall knows about it. |
| `recall bots leave <id>` | Remove a bot from its call without deleting what it already recorded. |
| `recall bots transcript <id>` | Fetch the transcript for a bot's most recent recording, once it is ready. |
| `recall recordings [--bot-id <id>] [--status <s>]` | List recordings captured by your bots. |
| `recall auth status` / `recall auth set-token <token>` / `recall auth logout` | Check or manage your API key. |
| `recall doctor` | Health check: config, auth, region, and live connectivity in one command. |

Every command supports `--json` (structured output), `--dry-run` (print the request, send
nothing), `--select field1,field2` (trim JSON to named fields), `--compact` (trim to high value
fields automatically), and `--agent` (turns on `--json --compact --no-input --no-color --yes` at
once, the shape a script or an LLM agent wants). Run `recall <command> --help` for the full flag
list; `recall <command> [subcommand] --help` goes one level deeper.

`recall bots transcript` is the one command here that is not a single Recall.ai endpoint. Recall
does not expose "give me bot X's transcript" directly: a bot has recordings, a recording has a
transcript artifact once one was requested and finished processing, and that artifact carries a
download URL for the actual transcript JSON. This command chains those steps (`GET
/api/v1/bot/{id}/`, then a direct fetch of the returned download URL) behind one call, because
that is how people actually ask for the feature.

## What ran against fixtures, and what is still untested

No Recall.ai API key was available while building this CLI. Everything below was checked without
one; nothing below is a claim that any command works against a real Recall.ai account yet.

Verified without a key:

- **Structural correctness**: `go build`, `go vet`, `go test ./...`, and `govulncheck`, all clean.
- **Every command's `--dry-run` path**, by hand: each of the 7 commands above prints the exact
  method, URL (region correct), headers, and JSON body it would send, and exits 0. This is what
  the quickstart above runs.
- **Every command against the spec derived mock server**, via this generator's own `verify`
  tool (`cli-printing-press verify --dir . --spec ./spec.yaml`), which starts a local HTTP
  server that answers according to `spec.yaml` and drives every command's help, dry-run, and
  execute path against it. Current result: **PASS, 9/9, 100%**.
- **`recall doctor`'s connectivity check specifically did make one real, live, unauthenticated
  request** to `https://us-east-1.recall.ai/api/v1/bot/` while building this CLI, and got a real
  HTTP 401. That confirms the region URL scheme and host are correct. It does not confirm that
  any bot, recording, or transcript command returns correct data against a real account.
- **The OpenAPI spec this CLI's field names and endpoints are drawn from is Recall's own**,
  pulled from their published documentation at docs.recall.ai (66 endpoints; this CLI covers 5 of
  them: bot create, list, retrieve, leave call, and recording list).

Not verified, pending a real API key:

- Whether `bots create` actually starts a bot, and whether `--transcribe` actually produces a
  transcript once a call finishes.
- The exact shape of a real bot response's `recordings[]` once a call has actually happened
  (`bots get`, `bots list`).
- The exact JSON shape of a downloaded transcript. `bots transcript` parses the per-speaker
  `{participant, words: [...]}` shape documented at
  [docs.recall.ai/docs/download-schemas](https://docs.recall.ai/docs/download-schemas), and falls
  back to printing the raw payload under `raw_transcript` if that shape does not match. That
  fallback has not been exercised against a real download.
- Behavior in the `eu-central-1`, `us-west-2`, and `ap-northeast-1` regions specifically (only the
  URL construction is verified; no live call was made to a non-default region).

## Health check

```bash
recall doctor
recall doctor --json
```

Reports config file state, whether `RECALL_API_KEY` is set and where it came from, the resolved
region and base URL, and live reachability of that base URL. Safe to run with no key.

## Troubleshooting

- **"no credentials configured"**: `RECALL_API_KEY` is not set and no key is saved locally. Run
  `export RECALL_API_KEY="your-key-here"` or `recall auth set-token <token>`.
- **"unknown --region value"**: only `us-east-1`, `us-west-2`, `eu-central-1`, and
  `ap-northeast-1` are recognized. For a non-standard endpoint (a local mock server, for example),
  set `RECALL_BASE_URL` directly instead of `--region`.
- **HTTP 401 from `recall doctor`**: the host is reachable but the key is missing or wrong. This
  is expected with no key set. It is not a bug.
- **A command hangs**: `--timeout` defaults to 60s. Lower it with `--timeout 10s` if you are
  scripting against a slow network.

## Cookbook

```bash
# Schedule a bot instead of joining immediately (Recall recommends 10+ minutes ahead).
recall bots create --meeting-url https://meet.google.com/abc-defg-hij --join-at 2026-08-01T15:00:00Z

# Pull only the fields you need out of a bot.
recall bots get 550e8400-e29b-41d4-a716-446655440000 --json --select id,meeting_url

# Filter bots by platform and status.
recall bots list --platform zoom --status done --json

# See the transcript as plain text instead of JSON.
recall bots transcript 550e8400-e29b-41d4-a716-446655440000

# Pull recordings for one specific bot.
recall recordings --bot-id 550e8400-e29b-41d4-a716-446655440000 --json
```

## Known gaps

- **Auto pagination is not wired.** `recall bots list` and `recall recordings` return one page.
  Recall returns a `next` URL with each page; page manually by passing its `cursor` value via
  filter flags for now.
- **Only `meeting_captions` transcription is exposed** via `--transcribe`. Recall also supports
  its own streaming transcription and several third party providers (Deepgram, AssemblyAI, and
  others); those need their own provider config and were out of scope for this demo's 7 command
  hero flow.
- **No write path beyond create and leave.** `recall bots create --stdin` accepts a raw JSON body
  on stdin for advanced options (video layout, chat, realtime endpoints, and so on) that do not
  have their own flags yet.
- **This demo intentionally does not include** an offline sync and search store, a self-learning
  query cache, or an MCP server. Those are part of the code generator's standard scaffold; they
  were removed here to keep the command surface to the 7 commands above. See
  `.printing-press.json`'s `demo_scope` field for the one-line version.

## License

MIT. See [LICENSE](LICENSE). This is an independent, unofficial project. Recall.ai and the
Recall.ai name belong to their respective owner; this project is not affiliated with, endorsed
by, or sponsored by them.
