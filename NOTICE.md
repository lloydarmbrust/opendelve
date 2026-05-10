# NOTICE

OpenDelve is licensed under the MIT License (see `LICENSE`).

This file records third-party content harvested or referenced by OpenDelve's
compliance packs. Entries are added as content is incorporated. Each entry
identifies the source, its license, and the OpenDelve files that derive from
or were inspired by it.

## Harvest sources (license-audited 2026-05-08)

The following sources were verified license-compatible with MIT redistribution:

| Source | License | OpenDelve usage |
|--------|---------|-----------------|
| [strongdm/comply](https://github.com/strongdm/comply) | Apache-2.0 | (pending) SOC 2 starter scaffold inspiration |
| [GSTT-CSC/QMS-Template](https://github.com/GSTT-CSC/QMS-Template) | Apache-2.0 | (pending) ISO 13485 starter structure |
| [opencontrol/schemas](https://github.com/opencontrol/schemas) | CC0 1.0 (public domain) | (pending) control-mapping conventions |
| [opencontrol/compliance-masonry](https://github.com/opencontrol/compliance-masonry) | CC0 1.0 (public domain) | (pending) authoring patterns |
| [ComplianceAsCode/content](https://github.com/ComplianceAsCode/content) | BSD-3-Clause | (pending) PCI DSS starter scaffold inspiration |

## Excluded sources

The following sources are NOT used as harvest material because their licenses
are incompatible with MIT redistribution:

| Source | License | Reason |
|--------|---------|--------|
| [openregulatory/templates](https://github.com/openregulatory/templates) | CC-BY-NC-SA 4.0 | NonCommercial clause incompatible with MIT |

OpenRegulatory may inform our authoring style (general structure, common
sections), but no text is copied into OpenDelve packs.

## ISO standards text

ISO 9001:2015, ISO 13485:2016, and other ISO standards are copyright ISO and
not redistributable. OpenDelve packs do NOT copy or paraphrase ISO standard
text. Pack content references control identifiers and clause numbers only;
the substantive procedural text is original to OpenDelve and represents a
plausible operational implementation, not an authoritative interpretation
of the standard.

## How attribution is recorded

When a file in `packs/` derives non-trivially from an Apache-2.0 or
BSD-3-Clause source, that file gets a frontmatter or top-comment
attribution like:

```yaml
attribution:
  source: github.com/strongdm/comply
  license: Apache-2.0
  derivation: adapted-from
  notes: Initial structure for access-review SOP; substantive content rewritten.
```

This file (`NOTICE.md`) gets a one-line entry summarizing the harvest in the
table above.
