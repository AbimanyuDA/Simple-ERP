# Checkpoint 2 — Core Operations

Status: **PASSED** on 2026-09-22. Phase 2 may proceed to Checkpoint 3.

## Delivered

- Purchase Request lifecycle with optimistic version checks, maker-checker approval/rejection, terminal rejection copy, cancellation, PR-to-PO conversion, and PO issue/cancel.
- Partial goods receipts with remaining-quantity validation and atomic positive stock ledger posting.
- Sales Order lifecycle with stock validation, maker-checker approval/rejection, terminal rejection copy, cancellation/confirmation, and partial delivery.
- Row-locked delivery posting that revalidates stock and cannot produce negative balances under concurrent requests.
- Inventory adjustment snapshot, submit/review, zero-difference completion, posting, cancellation, and inverse-movement reversal.
- Immutable source-linked `stock_movements` with rebuildable `stock_balances`; all corrections use inverse entries.
- Reusable operational idempotency middleware backed by `idempotency_keys`; matching replay returns the original status/body while payload mismatch is rejected.
- Responsive Purchasing, Sales, and Inventory workspaces plus desktop/mobile operational E2E coverage.
- OpenAPI routes and traceability coverage for Purchasing, Sales, Inventory, and their supporting actions.

## Verification evidence

| Gate | Result |
|---|---|
| Go unit tests | PASS |
| Go vet and race detector | PASS |
| PostgreSQL operational integration tests | PASS |
| Maker-checker | PASS; creator approval/rejection denied |
| Concurrent stock posting | PASS; only one competing delivery posts, balance remains non-negative |
| Ledger immutability/source linkage | PASS; update/delete trigger rejects mutation |
| Idempotency replay/payload conflict | PASS in middleware unit and PostgreSQL integration tests |
| Migration 5 empty up/down/up | PASS; `5 → 4 → 5`, clean each time |
| Frontend unit/component tests | PASS |
| TypeScript typecheck, ESLint, production build | PASS |
| Desktop/mobile browser E2E | PASS; 4 journeys |
| Backend/frontend Docker build | PASS |
| OpenAPI structural validation | PASS |
| BPMN/mapping ID coverage | PASS; 39 BPMN tasks + 21 supporting actions |

## Transaction, inventory, and security notes

- Receipt, delivery, adjustment, stock balance, immutable movement, business progress, and success audit are committed in one PostgreSQL transaction.
- Stock reservation is absent. Delivery and negative adjustment paths lock and revalidate the current stock immediately before posting.
- Reversal requires an Admin at the HTTP boundary, a mandatory reason, a linked inverse movement, and a success audit entry. Negative-result reversal is rejected.
- Operational mutations require `Idempotency-Key` after authentication. Keys are scoped per actor, bounded in length, and tied to a method/path/body hash.
- The failed initial operations E2E was caused by assuming a fixture PR survived the final integration test. The browser journey now creates its own draft through the UI before asserting state, avoiding test-order coupling.

## Traceability implementation roots

- Purchasing: `backend/internal/purchasing`, `frontend/features/purchasing`
- Sales: `backend/internal/sales`, `frontend/features/sales`
- Inventory: `backend/internal/inventory`, `frontend/features/inventory`
- Idempotency: `backend/internal/platform/idempotency`
- Migration: `backend/migrations/000005_operations.*.sql`
- Integration/E2E: `backend/test/integration/operations_test.go`, `frontend/e2e/operations.spec.ts`

## Observations carried forward

- Downstream-invoice reversal rejection becomes executable only after Checkpoint 3 introduces consolidated invoices; Checkpoint 3 must add the cross-document guards and tests before its gate closes.
- Finance completion will change FULLY_RECEIVED/FULLY_DELIVERED into COMPLETED only after full invoice payment; no manual or administrative short-close route exists.
