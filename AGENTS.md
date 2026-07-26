# recall: agent guide

Unofficial demo. Not affiliated with Recall.ai. This directory started as a `cli-printing-press`
generated CLI, then was hand trimmed to a 7 command hero flow for an outreach demo. It is not
published anywhere and will not be reprinted from the generator; treat it as a normal, standalone
Go repository, not generated output to be regenerated later.

## Local operating contract

Start by asking the CLI for current runtime truth instead of trusting a copied command list:

```bash
recall doctor --json
recall agent-context --pretty
recall <command> --help
```

Add `--agent` to command invocations for JSON, compact output, non-interactive defaults, no color,
and confirmation-safe scripting:

```bash
recall <command> --agent
```

Before running an unfamiliar command that may mutate remote state (`bots create`, `bots leave`),
inspect its help and prefer a dry run first:

```bash
recall <command> --help
recall <command> --dry-run --agent
```

Use `--yes --no-input` only after the target, arguments, and side effects are clear.

## No API key in this environment

If `RECALL_API_KEY` is not set, every read and mutate command still works with `--dry-run`: it
prints the exact request that would be sent and exits 0. Use that instead of guessing at behavior.
`recall doctor` and `recall auth status` both work with no key and report what is and is not
configured.

## Building

```bash
go build -o recall ./cmd/recall
go vet ./...
go test ./...
```

## Local customizations

This is a hand-edited repository now, not machine-generated output waiting on a future regen.
Edit files directly; there is no `.printing-press-patches/` convention to preserve here.

For install, auth, regions, the full command reference, and what has and has not been tested
against a real Recall.ai account, read `README.md`. `SKILL.md` is the terser agent-operations
version of the same material.
