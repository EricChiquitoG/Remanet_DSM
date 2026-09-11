# Electric-motor demonstrator AAS instances V2

This folder contains the schema-validated Product DT and remanufacturer capability AAS Environment JSON instances for the electric-motor use case.

## Deterministic scope

- Product DTs: all 22 records in `dataset_V2/motor_batches/batch_000.csv`.
- Capability AASs: the first five records in `dataset_V2/motor_providers/motor_providers.csv`, following the same demonstrator-fixture rule as the cutting-tool V2 generation.
- Owner/custodian enrichment: `dataset_V2/motor_owners/motor_owners.csv`.
- Source archive SHA-256: `b17e80f87c3a103b5ff83bde5e9eb8e08e39e4c78f3dc5157740533f1ea6f311`.

## Product DT instances

All eight submodels from `samples/aas/extensions/ProductChainStatus_Submodel_Design_v0.2/Product-DT-Generic-V0.8_ProductChainStatus.json` are retained as instances. A project-specific `ElectricMotorConditionAssessment` submodel preserves component identities, predicted assessments, normalized status, and real assessment evidence. ProductChainStatus references this current assessment without embedding workflow or process history.

| Serial | Batch | Condition | File |
|---:|---|---|---|
| 317829 | B137 | Poor | `Product-DT-ElectricMotor-317829.json` |
| 612942 | B514 | Poor | `Product-DT-ElectricMotor-612942.json` |
| 101299 | B827 | Poor | `Product-DT-ElectricMotor-101299.json` |
| 344000 | B006 | Poor | `Product-DT-ElectricMotor-344000.json` |
| 174388 | B473 | Poor | `Product-DT-ElectricMotor-174388.json` |
| 873938 | B470 | Poor | `Product-DT-ElectricMotor-873938.json` |
| 603266 | B026 | Poor | `Product-DT-ElectricMotor-603266.json` |
| 825072 | B187 | Poor | `Product-DT-ElectricMotor-825072.json` |
| 657593 | B435 | Poor | `Product-DT-ElectricMotor-657593.json` |
| 532184 | B453 | Poor | `Product-DT-ElectricMotor-532184.json` |
| 888445 | B314 | Poor | `Product-DT-ElectricMotor-888445.json` |
| 599675 | B891 | Poor | `Product-DT-ElectricMotor-599675.json` |
| 630361 | B792 | Poor | `Product-DT-ElectricMotor-630361.json` |
| 757659 | B180 | Poor | `Product-DT-ElectricMotor-757659.json` |
| 394387 | B054 | Poor | `Product-DT-ElectricMotor-394387.json` |
| 848137 | B337 | Poor | `Product-DT-ElectricMotor-848137.json` |
| 890647 | B854 | Poor | `Product-DT-ElectricMotor-890647.json` |
| 925471 | B883 | Poor | `Product-DT-ElectricMotor-925471.json` |
| 211335 | B396 | Poor | `Product-DT-ElectricMotor-211335.json` |
| 654453 | B565 | Poor | `Product-DT-ElectricMotor-654453.json` |
| 953855 | B145 | Poor | `Product-DT-ElectricMotor-953855.json` |
| 015342 | B314 | Poor | `Product-DT-ElectricMotor-015342.json` |

## Remanufacturer capability instances

Each supplied operation is represented by an IDTA 02020 `CapabilityContainer`. Missing operations are omitted. Provider identity, location, equipment, source specifications, energy, and cost are modeled as capability properties. Units and currencies are not invented.

| Provider | Name | Country | Capabilities | File |
|---:|---|:---:|---|---|
| 0 | ToolTechnologyProvider-000 | CY | Assembly Testing, Reassembly | `CapabilityDescription-ElectricMotorProvider-000.json` |
| 1 | ToolTechnologyProvider-001 | NL | Assembly Testing, Reassembly, Reuse, Assembly for Manufacturing, Grinding, Component Replacement, Motor Redesign, Dimensional Testing, Turning, Lamination Punching and Stacking, Inspection | `CapabilityDescription-ElectricMotorProvider-001.json` |
| 2 | ToolTechnologyProvider-002 | DK | Assembly Testing, Reuse, Assembly for Manufacturing, Grinding, Component Replacement, Motor Redesign, Lamination Punching and Stacking, Inspection, Assembly Inspection | `CapabilityDescription-ElectricMotorProvider-002.json` |
| 3 | ToolTechnologyProvider-003 | BG | Reassembly, Assembly for Manufacturing, Component Replacement, Dimensional Testing, Turning, Assembly Inspection, Disassembly | `CapabilityDescription-ElectricMotorProvider-003.json` |
| 4 | ToolTechnologyProvider-004 | FR | Assembly Testing, Reassembly, Component Replacement, Dimensional Testing | `CapabilityDescription-ElectricMotorProvider-004.json` |

## Mapping decisions

- `IdOwner` is legal ownership; `CurrentHolderID` is custody. Organization coordinates remain under owner/actor profiles and are not treated as physical product location.
- Product status is `InUse`, lifecycle `MiddleOfLife`, chain stage `InUse`, and circular strategy `None`.
- Source component conditions map as `Good -> Good`, `Mid -> Fair`, and `Bad -> Poor`; the worst predicted component condition becomes the ProductChainStatus condition.
- Nominal performance, mass fractions, component materials, and dimensions are placed in Technical Data. Component assessment evidence remains in the current condition submodel.
- Provider equipment dictionaries are normalized to readable labels and also preserved losslessly as compact JSON in `EquipmentSpecifications`.

## Validation

- Official AAS JSON schema 3.1.2: PASS for 27/27 generated environments.
- Unique AAS, asset, and submodel IDs; local reference resolution; instance kinds; ProductChainStatus rules; IDTA template concept retention; source coverage; and missing-sentinel checks: PASS.
- MDTS readiness: PASS as standalone AAS Environment JSON documents under the repository's pinned validation model.

See `generation-validation-report.json` and `generation-mapping.json` for machine-readable evidence.

## Rerun

`node scripts/generate-electric-motor-aas-v2.mjs`
