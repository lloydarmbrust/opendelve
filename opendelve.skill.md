---
name: opendelve
description: Use this skill when working in a repository managed by OpenDelve (presence of `opendelve.yaml` at repo root). OpenDelve is a compliance-as-code CLI for regulated AI workflows. Treat OpenDelve as the deterministic authority on what evidence, approvals, and policies apply to any subject in this repo. Never invent compliance state — always ask OpenDelve via its read commands and act on the structured JSON answer. Read the bundled `AGENTS.md` for the full safe-command allow-list.
---

# OpenDelve Skill

This skill helps an AI agent (Claude Code, Cursor, Codex, Copilot) operate
correctly inside an OpenDelve-managed repository. The repo is the source of
truth. OpenDelve is the deterministic authority. You ask, it answers.

## When to use this skill

Trigger this skill when ANY of the following is true:

- The current working directory contains `opendelve.yaml` at its root
- The user mentions "compliance", "SOP", "audit packet", "evidence", "approval", "ISO 9001", "SOC 2", "ISO 13485", or "PCI" in the context of this repo
- The user is about to commit changes that touch `policies/`, `sops/`, `workflows/`, `evidence/`, `approvals/`, `audit-packets/`, `evidence-schemas/`, or `controls/`
- The user asks "what's the state of this SOP / shipment / release?"

## Core rule

**Never decide compliance.** Ask OpenDelve. Act on the structured answer.

Wrong:
> "I think this SOP needs an approval from the QA manager."

Right:
> Run `opendelve verify SOP-0001 --json` and read `requiredApprovals` from the
> response. Present those exact roles to the user.

## Allow-listed read commands (run freely)

These are side-effect-free and emit a JSON decision envelope on stdout.

```bash
opendelve trace <subject> --json       # full audit trace for a subject
opendelve verify <subject> --json      # presence + schema validity check
opendelve explain <subject> --json     # agent-facing decision contract
opendelve schemas list --json          # enumerate evidence schemas
opendelve packs list --json            # enumerate installed packs
opendelve schema validate <file> --schema <name> --json
```

## Allow-listed write commands (require explicit user authorization)

Always pass `--dry-run` first. Show the user the would-be output. Only invoke
the real write after explicit per-invocation human authorization.

```bash
opendelve evidence add <file> --schema <s> --dry-run    # then without --dry-run after confirmation
opendelve approve request <subject> --dry-run
opendelve sop add <slug> --dry-run
opendelve pack add <pack> --dry-run
```

## Hard-blocked commands (humans only)

You must never invoke these:

- `opendelve approve sign` — signing is a deliberate human act
- merging protected branches
- bypassing `blockingConditions` declared on an SOP or workflow
- editing approval JSON receipts after they are written
- inventing or fabricating evidence (only files declared by the user with a matching schema are valid)

## How to read the decision envelope

Every read command emits this JSON on stdout (full schema:
`schemas/decision.schema.json`):

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

Decision tree:
- `status: "ok"` → proceed with your plan's next step
- `status: "needs_evidence"` → run `evidence add` for each entry in `missingEvidence` (one at a time, with user authorization per the write rules above)
- `status: "needs_approval"` → tell the user which approvals are missing; do NOT call `approve sign` yourself
- `status: "blocked"` → STOP. Report the `blockedActions` to the user. Do not work around the block.

## How to read the error envelope

When the command fails, OpenDelve emits an error envelope on stdout (full
schema: `schemas/error.schema.json`):

```json
{
  "schemaVersion": "v1",
  "kind": "error",
  "code": "git_not_initialized",
  "message": "this directory is not a git repository",
  "hint": "run git init to initialize a repo, then try again",
  "timestamp": "2026-05-08T12:00:00Z"
}
```

Distinguish success vs error by the top-level `kind: "error"` field.

## Exit codes

- `0` — success (`status: "ok"`)
- `2` — non-error decision requiring action (`status: "needs_*" | "blocked"`); JSON `status` field disambiguates
- `1` — error envelope returned

## Worked example

User says: "I need to release shipment-042. What's the state?"

```bash
$ opendelve verify shipment-042 --json
{
  "schemaVersion": "v1",
  "status": "needs_evidence",
  "subject": "shipment-042",
  "policyRefs": ["SOP-0007"],
  "requiredEvidence": ["temperature-log", "shipment-label"],
  "missingEvidence": ["shipment-label"],
  "requiredApprovals": ["qa_manager", "ops_manager"],
  "missingApprovals": ["qa_manager", "ops_manager"],
  "blockedActions": ["release"],
  "allowedNext": ["evidence add", "approve request"]
}
```

You report to the user:
> "Shipment-042 is blocked from release. Missing: shipment-label evidence, plus
> approvals from qa_manager and ops_manager. Per OpenDelve's allow-list, I can
> attach the shipment-label evidence and request the approvals; signing has to
> be done by a human. Want me to add the evidence file? (Path the user provides
> goes through `opendelve evidence add` — I'll show you the dry-run first.)"

## Reference

- Full agent contract: `AGENTS.md`
- Schemas: `schemas/*.schema.json`
- Repository: https://github.com/opendelve/opendelve
