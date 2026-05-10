---
schemaVersion: v1
id: SOP-0001
title: Temperature-Sensitive Shipment Release
version: 1.0.0
status: draft
owners:
  - qa_manager
  - ops_manager
effectiveDate: null
reviewCycleDays: 365
requiredApprovals:
  - role: qa_manager
    signaturesRequired: 1
  - role: ops_manager
    signaturesRequired: 1
evidenceRequirements:
  - schema: temperature-log
    required: true
blockingConditions:
  - missing_required_evidence
  - temperature_out_of_range
  - required_training_incomplete
workflowBindings:
  - WF-0001
controlMappings:
  - framework: iso9001
    control: "8.5.1"
  - framework: iso9001
    control: "8.5.4"
trainingRequirements: []
signatureRequirements:
  meaning: "I approve this shipment for controlled release."
  reauthRequired: false
auditPacketOutputs:
  - approval_certificate_pdf
  - evidence_manifest_json
  - packet_pdf
---

# Purpose

Release temperature-sensitive shipments (e.g., peptide compounds, cold-chain
pharmaceuticals, biologics) only when complete temperature evidence and the
required quality and operations approvals are on file.

# Scope

Applies to all outbound shipments labeled as temperature-sensitive on the
internal product catalog or whose product class falls under cold-chain
handling per the storage-conditions matrix.

# Responsibilities

- **QA Manager** — verifies temperature evidence is complete, in range, and
  attributable to the correct shipment lot. Signs first.
- **Ops Manager** — verifies the trained operator signed the picking record
  and the carrier's manifest is attached. Signs second.
- **Trained Operator** — attaches the temperature log file at pick time
  using `opendelve evidence add <file> --schema temperature-log`.

# Procedure

1. **Attach temperature log.** The operator runs:
   ```
   opendelve evidence add ./shipment-NNN-temp-log.csv --schema temperature-log --subject shipment-NNN
   ```
   The temperature log must cover the full storage-to-pack window and contain
   readings at no less than 30-minute intervals.
2. **Validate temperature range.** The system runs the `temperature-log`
   schema validator. Min and max must be within the product's labeled range
   (default: 2°C to 8°C unless the product specifies otherwise).
3. **Confirm trained operator.** The Ops Manager verifies the operator has
   a current training record (TR-NNNN with status: effective) and that the
   training covers cold-chain handling.
4. **Request QA approval.**
   ```
   opendelve approve request shipment-NNN
   ```
5. **QA Manager signs first** with meaning "I approve this shipment for
   controlled release":
   ```
   opendelve approve sign APR-2026-NNNN --name "Jane Doe" --meaning "..."
   ```
6. **Ops Manager signs second** with the same meaning text.
7. **Release shipment.** The carrier picks up. The system emits an audit
   packet:
   ```
   opendelve audit packet build shipment-NNN
   ```

# Blocking conditions

The following conditions block release. The CLI will refuse to advance the
state and the agent decision envelope will return `status: blocked`:

- **missing_required_evidence** — no `temperature-log` evidence is attached
- **temperature_out_of_range** — any reading in the log exceeds the product's
  labeled range
- **required_training_incomplete** — the picking operator has no current
  training record covering cold-chain handling

# References

- ISO 9001:2015 § 8.5.1 — Control of production and service provision
- ISO 9001:2015 § 8.5.4 — Preservation
- WF-0001 — Shipment release workflow
- temperature-log evidence schema (`evidence-schemas/temperature-log.yaml`)
