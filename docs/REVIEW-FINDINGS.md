# Phase 1 Decision Record and Final Review

## Status

Phase 1 baseline was approved on 2026-09-21. All open decisions are locked below. The implementation structure is one Phase 2 with four internal checkpoints; implementation remains blocked until that revised plan is explicitly approved.

## Approved decisions

| ID | Final decision | Enforced in |
|---|---|---|
| OD-001 | Mandatory maker-checker. Creator cannot approve/reject their own transaction. No Admin override. | Business rules, DB checks, use-case authorization, tests |
| OD-002 | Zero quantity and price tolerance for three-way match. | P2P rules, match use case, tests |
| OD-003 | Partial receipt, delivery, and payment are supported. | BPMN loops, state machines, ERD, API |
| OD-004 | One consolidated sales invoice after full delivery. | O2C BPMN, finance ERD/API |
| OD-005 | One consolidated supplier invoice after all goods are received. | P2P BPMN, finance ERD/API |
| OD-006 | No stock reservation; revalidate stock immediately before inventory posting. | Inventory port, posting use cases, tests |
| OD-007 | Negative stock is prohibited. | Inventory domain invariant and DB transaction locking |
| OD-008 | Default currency `IDR`; default timezone `Asia/Jakarta`; store as organization settings where practical. | Organization settings and API presentation |
| OD-009 | Database supports multiple warehouses; MVP UI may preselect one default warehouse. | ERD, API filters, UI defaults |
| OD-010 | Finance is AP/AR invoice and payment tracking only. No GL or journal accounting. | Domain boundary and exclusions |

Additional approved decisions:

- Rejected Purchase Requests and Sales Orders are terminal and may be copied into a new `DRAFT` document.
- Invoice/payment void requires Admin, mandatory reason, and audit trail; no second approver.
- A payment must be voided before an invoice with active allocations can be voided. A replacement consolidated invoice may reference the prior VOID invoice; only one non-VOID invoice may exist per PO/SO.
- Administrative short-close of partial fulfillment is excluded.
- Password Reset and User Invitation use 24-hour, single-use, hashed tokens and manual secure link sharing because external notification integration is excluded.
- Browser authentication uses short-lived opaque access and rotating refresh sessions in secure HttpOnly cookies with CSRF protection; raw token values are never persisted or audited.
- The first Admin is created only by a guarded local bootstrap CLI; no bootstrap API or default credential exists.
- Basic cancellation is limited to explicitly mapped pre-posting/pre-fulfillment states. Automatic close is internal; no manual PR/PO/SO close endpoint exists.

## Explicitly out of scope for MVP

- Purchase returns / return to supplier
- Sales returns, refunds, and credit notes
- Warehouse transfers
- Tax engine
- Stock reservation and backorder allocation
- Administrative short-close and advanced cancellation
- General ledger and journal accounting
- External email, WhatsApp, SMS, or other notification integrations
- Multi-company tenancy, lot/serial/batch tracking, and multi-currency accounting

These capabilities cannot be improvised using negative documents or direct ledger edits. Each requires new BPMN, state-machine, data-model, API, and traceability review before implementation.

## BPMN semantic review

| Check | Final result |
|---|---|
| Level 0 capability coverage | Pass: Auth, Master Data, P2P, O2C, Inventory, Finance, Reporting, Audit |
| Level 1 process coverage | Pass: Access Lifecycle, P2P, O2C, Inventory Adjustment |
| Level 2 detailed coverage | Pass: Invitation/Reset, P2P, O2C, Inventory Adjustment |
| Pools represent participants | Pass |
| Lanes represent human/system responsibilities | Pass |
| Sequence flow crosses pool boundary | Pass: none |
| Message flow used only between pools | Pass |
| XOR splits have named outcomes and valid default | Pass |
| Loop merges are explicit XOR merges | Pass |
| Every path reaches an end event | Pass |
| User vs Service Task classification | Pass |
| Level 3 is implementation mapping, not technical BPMN | Pass |

Models remain engine-neutral and descriptive (`isExecutable="false"`). Go use cases enforce the same state transitions; no workflow engine is introduced.

## Responsibility review

- Admin owns approval, user management, master-data mutation, invoice/payment handling, and void actions.
- Operator owns operational entry, receipt, delivery, and stock count.
- User owns activation/reset password entry through one-time links.
- System owns deterministic validation, posting, token consumption, session revocation, ledger writes, and audit append.
- Inventory exclusively owns stock movement and balance projection.
- Finance exclusively owns AP/AR invoices, allocations, and payments.
- Reporting is read-only; Audit records facts and never decides transitions.

No duplicate business ownership remains.

## State transition review

- `REJECTED` is terminal for Purchase Request and Sales Order.
- Copy-to-draft creates a new document with a new ID/number and `copied_from_id`; it does not reopen the rejected aggregate.
- Partial receipt/delivery/payment states are retained.
- Posted stock/finance records are immutable; correction creates linked reversal records.
- Invoice/payment void stores reason, actor, time, and reversal reference.
- Short-close is absent from BPMN and API by design.
- All inventory posting revalidates stock/source/version in its transaction.
- Revoke, expiry, cancel, copy, void, and reversal operations have stable supporting action IDs, audit expectations, and test families in `PROCESS-MAPPING.md`.

## Architecture review

The backend remains a modular monolith using Clean Architecture per domain. Domain entities have no HTTP/SQL dependencies; application packages own ports; adapters implement them; `cmd/api` is the only composition root. Generic repositories, a global domain model, workflow runtime, message broker, microservices, and accounting framework remain excluded.

Migrations create warehouse master data before `organization_settings.default_warehouse_id`. Production deployment is deliberately excluded from Phase 2 and requires a separate approval.

## Final Phase 1 sign-off

- [x] OD-001 through OD-010 resolved.
- [x] Additional state/void/scope decisions recorded.
- [x] Password Reset and Invitation modeled and mapped.
- [x] BPMN hierarchy updated and structurally validated.
- [x] State machines, ERD, API, and traceability updated.
- [x] Stable BPMN IDs checked for uniqueness and mapping coverage.
- [x] Final traceability coverage report produced.
- [ ] Phase 2 implementation plan reviewed and approved.
