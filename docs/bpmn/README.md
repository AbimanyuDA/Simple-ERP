# BPMN Architecture

## Hierarchy

| Level | File | Meaning |
|---|---|---|
| 0 | `l0-enterprise-landscape.bpmn` | ERP capability landscape |
| 1 | `l1-core-processes.bpmn` | phase view of Access Lifecycle, P2P, O2C, and Inventory Adjustment |
| 2 | `l2-access-lifecycle.bpmn` | detailed User Invitation and Password Reset workflow |
| 2 | `l2-procure-to-pay.bpmn` | detailed P2P workflow |
| 2 | `l2-order-to-cash.bpmn` | detailed O2C workflow |
| 2 | `l2-inventory-adjustment.bpmn` | detailed adjustment workflow |
| 3 | `../PROCESS-MAPPING.md` | implementation traceability matrix |

## Modeling policy

- BPMN 2.0 descriptive models; no workflow engine binding in Phase 1.
- One pool per participant. External Supplier and Customer are black-box pools.
- Lanes communicate responsibility, not system architecture.
- Sequence flow remains inside a process/pool.
- Message flow exists only between participants.
- XOR gateways express exclusive decisions and explicit merge points.
- No parallel gateway is used because the approved baseline has no mandatory concurrent branches.
- Human activity is a User Task; deterministic application automation is a Service Task.
- Technical calls, database tables, and source files belong in Level 3 mapping, not in BPMN lanes.
- Initialization, secondary lifecycle, and correction commands such as bootstrap, revoke, expiry, basic cancel, copy, void, and reversal remain sourced from the appropriate Level 0 capability. They use stable `ACT-*` IDs in Level 3 and are not inserted as artificial tasks into the Level 2 happy paths.

## Change control

1. Change BPMN first and allocate stable task IDs.
2. Update `BUSINESS-RULES.md`, `STATE-MACHINES.md`, and `PROCESS-MAPPING.md` in the same change.
3. Update API/use case/database/test artifacts after design approval.
4. Never reuse a retired task ID.
5. A code transition without BPMN and mapping source is a release blocker.

## Validation

The files must be well-formed XML and visually reviewed in a BPMN 2.0-compatible modeler. Camunda-specific executable lint is not applicable while models remain engine-neutral and `isExecutable="false"`. If Camunda is adopted, add engine extensions and require zero-error/zero-warning `c8ctl bpmn lint` before deployment.
