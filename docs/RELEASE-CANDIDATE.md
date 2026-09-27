# Mini ERP MVP Release Candidate Report

Date: 2026-09-27
Status: **TA BASELINE CANDIDATE VERIFIED LOCALLY; PRODUCTION READINESS NOT ASSESSED**
Production readiness: **NOT APPROVED**; deployment requires a separate plan and approval.

## Implemented capability

- Secure authentication, session lifecycle, invitation/password reset, Admin/Operator authorization, users, settings, and master data.
- Procure-to-Pay from PR through PO, partial receipt, consolidated supplier invoice, partial payment, and automatic completion.
- Order-to-Cash from SO through partial delivery, consolidated sales invoice, partial payment, and automatic completion.
- Maker-checker approvals, terminal rejection with copy-to-draft, guarded cancellation, and reasoned reversal/void flows.
- Multi-warehouse database model with one default-warehouse MVP UI.
- Immutable source-linked stock movement ledger, negative-stock prevention, balances, and inventory adjustments.
- AP/AR tracking only, dashboard, operational/finance reports, and immutable searchable audit trail.

## Final verification

The table below records an earlier release-candidate run and is not current evidence for this remediation branch. Re-run every applicable gate after migration 9 before restoring release-candidate status.

| Verification | Result |
|---|---|
| Backend unit, vet, race | PASS |
| PostgreSQL integration | PASS |
| Frontend unit/component | PASS: 8/8 |
| Frontend typecheck/lint/build | PASS |
| Browser E2E desktop/mobile | PASS: 10/10 |
| Migration/rollback | PASS: fresh v7 and `7 → 6 → 7` |
| Backup/restore | PASS on disposable restored database |
| Stock ledger reconciliation | PASS: 0 mismatch, 0 negative balance |
| AP reconciliation | PASS: 0 mismatch |
| AR reconciliation | PASS: 0 mismatch |
| Security/dependency review | PASS; no unresolved critical/high finding |
| BPMN semantic validation | PASS |
| OpenAPI validation | PASS |
| Traceability validation | HISTORICAL PASS: 39/39 tasks and 21/21 supporting actions; current mapping has 22 supporting actions and requires a fresh run |

Detailed evidence is in `docs/checkpoints/CHECKPOINT-1.md` through `CHECKPOINT-4.md`, `docs/SECURITY-REVIEW.md`, and `docs/TRACEABILITY-COVERAGE.md`.

### Remediation verification (2026-09-27)

These checks used an isolated local Docker Compose project and support the TA
study baseline only:

| Verification | Result |
|---|---|
| Backend unit and vet | PASS |
| Backend race detector | SKIPPED: Windows Go environment has no CGO support |
| PostgreSQL integration | PASS against a disposable PostgreSQL database |
| Frontend unit/component | PASS: 16/16 |
| Frontend typecheck, lint, and production build | PASS |
| Browser E2E desktop | PASS: 6/6 using local Chrome |
| Browser E2E mobile | NOT RUN |
| Fresh migration | PASS through migration 9 |
| Migration rollback | NOT RUN |
| BPMN XML and traceability validation | PASS: 39/39 tasks and 22/22 supporting actions |
| Backup, restore, and production operations | NOT ASSESSED |

## Known limitations

- One organization and one default warehouse in the UI; database supports multiple warehouses.
- No stock reservation; stock is revalidated at posting.
- No administrative short-close; partial fulfillment remains open until fully fulfilled.
- Reports/audit are Admin-only and use simple bounded tables without pagination/export.
- In-process rate limiting assumes one backend instance.
- Invitation/reset delivery is manual copy/share; no email or external notification integration.
- Production operations, observability, TLS/edge controls, retention policy, and disaster-recovery automation are not included.

## Explicitly out of scope

Purchase returns, sales returns/refunds, warehouse transfers, tax engine, stock reservation, advanced cancellation, short-close, general ledger, journal accounting, full accounting, multi-company, external notification integrations, and production deployment remain absent.

## Review focus

The recommended next action is stakeholder acceptance review of workflows, terminology, and UI ergonomics using the local run instructions in `README.md`. Any material business-process change must return to BPMN/business-rule/state/API design before implementation.
