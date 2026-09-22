# Checkpoint 3 — Simple Finance

Status: **PASSED** on 2026-09-22. Phase 2 may proceed to Checkpoint 4.

## Delivered

- AP supplier invoice recording only after a PO is fully received, with one active consolidated invoice per PO.
- Zero-tolerance three-way quantity/price matching, explicit `MATCH_EXCEPTION`, correction back to `MATCH_PENDING`, and posting after a clean match.
- AR sales invoice generation only after complete delivery, with one active consolidated invoice per Sales Order.
- Supplier and customer payment drafts with allocation-sum, positive amount, IDR/currency, payment-date, and outstanding validation.
- Partial payment, overpayment prevention, atomic outstanding/status updates, and immutable posted records.
- Automatic `PO RECEIVED → CLOSED` plus `PR CONVERTED_TO_PO → COMPLETED`, and `SO DELIVERED → COMPLETED`, only after final payment.
- Admin-only invoice/payment void with mandatory reason and audit. Payment void restores outstanding and automatically reopens source documents.
- Invoice void rejection while active allocations exist; linked replacement invoice only after the original is VOID.
- Receipt/delivery reversal is blocked while a downstream non-VOID invoice exists.
- AP/AR Finance workspace showing outstanding, match/post/void actions, partial payment capture, and immutable correction guidance.
- Finance REST contract and stable trace IDs in OpenAPI.

## Verification evidence

| Gate | Result |
|---|---|
| Go domain/unit tests | PASS; exact match, partial/overpayment, reversal, date/currency/allocation validation |
| Go vet and race detector | PASS |
| Full P2P PostgreSQL integration | PASS; early invoice, mismatch/resolution, partial/final pay, close/reopen, allocation guard, void/replacement |
| Full O2C PostgreSQL integration | PASS; early invoice, consolidated invoice, partial/final receipt, complete/reopen, void/replacement |
| Downstream stock reversal guards | PASS for active AP and AR invoices |
| AP/AR reconciliation assertions | PASS; outstanding values and source states verified after post/void |
| Migration 6 empty up/down/up | PASS; `6 → 5 → 6`, clean each time |
| Frontend component tests | PASS |
| TypeScript, ESLint, production build | PASS |
| Admin and Operator finance browser E2E | PASS on desktop and mobile |
| Docker backend/frontend build | PASS |
| OpenAPI structural validation | PASS |
| BPMN/stable-ID coverage | PASS; 39 tasks + 21 supporting actions |

## Migration and state alignment

- Migration 6 adds supplier/customer invoices, invoice lines, payments, allocations, partial unique active-invoice constraints, and outstanding indexes.
- It also normalizes the earlier provisional database labels to the approved state-machine vocabulary: `FULLY_RECEIVED → RECEIVED` and `FULLY_DELIVERED → DELIVERED`; paid sources use `CLOSED`/`COMPLETED` exactly as documented.
- The down migration safely restores the Checkpoint 2 labels only on the isolated rollback path.

## Transaction and security notes

- Payment posting/void, allocations, outstanding, document status, source close/reopen, and audit commit in one database transaction.
- Finance routes are default-deny to Operator and require Admin after authenticated session, CSRF, and idempotency middleware.
- Posted invoice/payment mutations have command routes only; there are no edit/delete or manual close/reopen endpoints.
- Finance remains AP/AR tracking only. No journal, chart of accounts, double-entry ledger, or tax engine was introduced.

## Traceability implementation roots

- Go: `backend/internal/finance/{domain,application,adapter}`
- Database: `backend/migrations/000006_finance.*.sql`
- UI: `frontend/features/finance`, `frontend/app/finance/page.tsx`
- Integration/E2E: `backend/test/integration/finance_test.go`, `frontend/e2e/finance.spec.ts`
