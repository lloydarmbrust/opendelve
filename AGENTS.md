# Agent rules for OpenDelve

This file declares the safe-command allow-list and decision contract for AI
agents (Claude Code, Cursor, Codex, Copilot, etc.) operating in or alongside
an OpenDelve-managed repository.

The repo is the source of truth. OpenDelve is the deterministic authority on
applicability, required evidence, required approvals, and next safe action.

Agents do not "decide compliance." Agents ask OpenDelve the deterministic
question and act on the structured answer.

## Allow-listed read commands (safe for any agent)

These commands have no side effects and emit a JSON decision envelope
(see `schemas/decision.schema.json`) on stdout. Agents may run them freely.

- `opendelve trace <subject> --json` — full audit trace for a subject
- `opendelve verify <subject> --json` — presence + schema validity check
- `opendelve explain <subject> --json` — agent-facing decision contract
- `opendelve schema validate <file> --schema <name> --json` — validate any file against a named schema
- `opendelve schemas list --json` — enumerate available evidence schemas
- `opendelve packs list --json` — enumerate installed packs

## Allow-listed write commands (require explicit user authorization)

These commands mutate repo state. Agents MUST obtain explicit per-invocation
human authorization before running them. Agents SHOULD pass `--dry-run`
first to preview the change and present it for human confirmation.

- `opendelve evidence add <file> --schema <schema> [--dry-run]` — attach evidence (manifest only by default; never raw)
- `opendelve approve request <subject> [--dry-run]` — request a human approval
- `opendelve sop add <slug> [--dry-run]` — create a new SOP file
- `opendelve pack add <pack> [--dry-run]` — install an additional compliance pack

## Hard-blocked for agents (humans only)

These commands MUST NOT be invoked by an agent under any circumstance.

- `opendelve approve sign` — signing is a deliberate human act; signature meaning cannot be delegated
- merging protected branches
- marking training records complete on behalf of a person
- inventing or fabricating evidence (only files declared by the user with a matching schema are valid)
- bypassing `blockingConditions` declared on an SOP or workflow
- editing approval JSON receipts after they are written

## Decision envelope (what agents read)

Every read command emits this JSON shape on stdout:

```json
{
  "schemaVersion": "v1",
  "status": "ok" | "needs_approval" | "needs_evidence" | "blocked",
  "subject": "<id>",
  "policyRefs": ["..."],
  "requiredEvidence": ["..."],
  "missingEvidence": ["..."],
  "requiredApprovals": ["..."],
  "missingApprovals": ["..."],
  "blockedActions": ["..."],
  "allowedNext": ["..."],
  "git": { "commit": "...", "branch": "...", "dirty": false },
  "timestamp": "2026-05-08T12:00:00Z"
}
```

- `status: "ok"` — the agent may proceed with subsequent steps in its plan
- `status: "needs_*" | "blocked"` — the agent MUST escalate to a human or take only an `allowedNext` action
- `blockedActions` — actions the agent must not take while the subject is in this state

When the command fails, OpenDelve emits an error envelope (see
`schemas/error.schema.json`) with `kind: "error"`, a stable `code`, a
`message`, and an optional `hint`. Agents distinguish success vs failure by
the presence of a top-level `kind: "error"` field.

## Exit codes

- `0` — success (`status: "ok"`)
- `2` — non-error decision requiring action (`status: "needs_*" | "blocked"`); JSON `status` field disambiguates
- `1` — error envelope returned

## Never assume

- Never assume a schema name. Use `opendelve schemas list` first.
- Never assume an approval can be skipped. Read `requiredApprovals` from the decision envelope.
- Never assume raw evidence may be embedded in a packet. The PHI policy is opt-in via `--include-raw` with explicit user confirmation.

## Reference

Full schema definitions: `schemas/*.schema.json`
Documentation: https://github.com/lloydarmbrust/opendelve
