# OpenDelve

**Compliance as code for AI agents.**

GitHub-native compliance bootstrap kit for regulated AI workflows. Scaffolds a working ISO 9001 quality system in your repo, generates SOPs, attaches evidence, signs approvals, and emits audit-grade artifacts. AI agents read the same machine-readable policies and use them to verify your software stays compliant.

> **Status:** v0 — under active development. First release ships when the demo path is end-to-end green in CI.

## Quick start (target experience)

```bash
brew install lloydarmbrust/tap/opendelve

opendelve init my-quality --pack iso9001
cd my-quality

opendelve sop add temperature-sensitive-shipment-release
opendelve evidence add ./temp-log.csv --schema temperature-log
opendelve approve request SOP-0001
opendelve approve sign APR-2026-0001 \
  --name "Jane Doe" \
  --meaning "I approve this SOP for controlled operational use"
opendelve audit packet build SOP-0001
```

Output: a regulator-grade audit packet, signed approval certificate, and full git-bound trace in under two minutes from `brew install`.

## What ships in v1

- 1 deep pack: ISO 9001
- 3 starter scaffolds: SOC 2, ISO 13485, PCI DSS (each prints "v0.2 — watch the repo")
- Single binary, multi-arch (darwin arm64/amd64, linux amd64/arm64, windows amd64)
- Audit packet PDF (Maroto-rendered)
- JSON-first agent contract + bundled Claude Code skill (`opendelve.skill.md`)
- Zero telemetry, MIT licensed, SBOM-published, Sigstore-signed binaries

## Documentation

The full design doc lives at `docs/design/v1.md` (in-progress). Spec and rationale are tracked there.

## License

MIT — see [LICENSE](./LICENSE).
