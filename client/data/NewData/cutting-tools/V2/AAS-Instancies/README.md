# Cutting-tool demonstrator AAS instances V2

This folder contains the schema-validated AAS Environment JSON instances for the HTS/Wolco cutting-tool demonstrator flow.

## Deterministic demonstrator scope

- Product DTs: all 14 records in `dataset_V2/tool_batches/batch_000.csv`.
- Remanufacturer capability AASs: the first five records in `dataset_V2/tool_providers/tool_providers.csv`, matching the five-provider demonstrator fixture used by the prior version.
- Owner enrichment: `dataset_V2/tool_owners/tool_owners.csv`.
- Source archive SHA-256: `7865413bf6dc85fc020af3caa5df853567b50cfab42c0fde0e59630b3b93e929`.

## Product DT instances

The Product DTs are instantiated from `samples/aas/extensions/ProductChainStatus_Submodel_Design_v0.2/Product-DT-Generic-V0.8_ProductChainStatus.json`. All eight template submodels are retained, populated as instances, and augmented with the project-specific current `CuttingToolConditionAssessment` submodel referenced by `ProductChainStatus`.

| Serial | Batch | Owner | File |
|---:|---|---:|---|
| 901945 | B364 | 0 | `Product-DT-CuttingTool-901945.json` |
| 441728 | B692 | 0 | `Product-DT-CuttingTool-441728.json` |
| 479023 | B311 | 0 | `Product-DT-CuttingTool-479023.json` |
| 139438 | B881 | 0 | `Product-DT-CuttingTool-139438.json` |
| 461005 | B046 | 0 | `Product-DT-CuttingTool-461005.json` |
| 247102 | B964 | 0 | `Product-DT-CuttingTool-247102.json` |
| 562399 | B920 | 0 | `Product-DT-CuttingTool-562399.json` |
| 329635 | B462 | 0 | `Product-DT-CuttingTool-329635.json` |
| 492804 | B599 | 0 | `Product-DT-CuttingTool-492804.json` |
| 885070 | B863 | 0 | `Product-DT-CuttingTool-885070.json` |
| 432926 | B033 | 0 | `Product-DT-CuttingTool-432926.json` |
| 315794 | B074 | 0 | `Product-DT-CuttingTool-315794.json` |
| 969165 | B394 | 0 | `Product-DT-CuttingTool-969165.json` |
| 957311 | B116 | 0 | `Product-DT-CuttingTool-957311.json` |

## Remanufacturer capability instances

The capability files retain the IDTA 02020 Capability Description 1.0 structure and concept descriptions. Each supplied operation becomes one `CapabilityContainer`. Missing operations are omitted. Provider identity, location, equipment, supported coating materials/types, energy, and cost are represented as capability properties; units and currencies are not invented because the dataset does not provide them.

| Provider | Name | Country | Capabilities | File |
|---:|---|:---:|---|---|
| 0 | ToolTechnologyProvider-000 | NL | Recoating, Testing, Regrinding, Inspection | `CapabilityDescription-ToolProvider-000.json` |
| 1 | ToolTechnologyProvider-001 | SK | Recoating | `CapabilityDescription-ToolProvider-001.json` |
| 2 | ToolTechnologyProvider-002 | FR | Recoating, Regrinding, Inspection | `CapabilityDescription-ToolProvider-002.json` |
| 3 | ToolTechnologyProvider-003 | RO | Recoating, Testing | `CapabilityDescription-ToolProvider-003.json` |
| 4 | ToolTechnologyProvider-004 | LT | Recoating, Regrinding, Inspection | `CapabilityDescription-ToolProvider-004.json` |

## Mapping decisions

- `IdOwner` is legal ownership; `CurrentHolderID` is custody/responsibility. Owner or holder coordinates are organization coordinates and are not copied into `CurrentLocation`.
- Product status follows the supplied Product Chain Status cutting-tool design: `InUse`, `MiddleOfLife`, chain stage `InUse`, and circular strategy `None`.
- Changing dimensions, resharpening count, self-assessment, and wear/damage are stored in `CuttingToolConditionAssessment`, keeping Product Chain Status as a current-state projection.
- Energy, cost, wear, geometry, and cutting parameters preserve source values. No undocumented units or currency are asserted.
- Capability equipment dictionaries are normalized to readable equipment labels. Recoating material categories and coating types are retained as RemaNet extension properties inside the IDTA `PropertySet` pattern.

## Validation

- Official AAS JSON schema 3.1.2: PASS for 19/19 generated environments.
- ProductChainStatus semantic ID and banned process/snapshot-field checks: PASS.
- IDTA 02020 Capability Description semantic ID and template concept-description retention: PASS.
- AAS-to-submodel reference resolution, unique global IDs, instance kinds, source serial/batch/provider coverage, and missing-sentinel checks: PASS.
- MDTS readiness: PASS as standalone AAS Environment JSON documents accepted by the repository's pinned AAS validation model.

Detailed machine-readable evidence is in `generation-validation-report.json` and `generation-mapping.json`.

## Rerun

`node scripts/generate-cutting-tool-aas-v2.mjs`
