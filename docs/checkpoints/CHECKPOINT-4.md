# Checkpoint 4 — MVP Completion

Status: **PASSED** on 2026-09-23. Phase 2 implementation is complete and stops at release-candidate review.

## Delivered

- Reconciled dashboard metrics for approvals, active operations, zero stock, AP, AR, and overdue exposure.
- Admin purchase, sales, inventory, AP, and AR reports using purpose-built read queries.
- Admin audit explorer with actor, action, entity, state transition, stable trace ID, request ID, and secret-safe fields.
- Immutable audit-entry database trigger and reporting indexes in migration 7.
- Authenticated per-user rate limit of 120 requests/minute with `429` and `Retry-After`.
- Responsive dashboard/report layout for desktop and mobile, including the grid shrink fix discovered by E2E.
- Strict BPMN semantic lint for all Level 0–2 documents and root verification integration.
- OpenAPI coverage for dashboard, reports, audit, and idempotent operational commands.
- Go build image pinned to 1.26.6 and `golang.org/x/text` 0.39.0 after vulnerability review.

## Verification evidence

| Gate | Result |
|---|---|
| Go unit/application tests | PASS |
| Go vet and race detector | PASS |
| PostgreSQL integration | PASS, including reconciled reporting and immutable/redacted audit |
| Frontend component tests | PASS: 6 files, 8 tests |
| TypeScript, ESLint, production build | PASS |
| Browser E2E | PASS: 10/10 across desktop and mobile; no skipped tests |
| Docker integrated build | PASS |
| Migration 7 | PASS: fresh `→ 7`, rollback `7 → 6`, reapply `6 → 7` |
| Backup/restore rehearsal | PASS: restored disposable database at schema `7:false` |
| Stock/AP/AR reconciliation | PASS: 0/0/0 mismatches; 0 negative balances |
| npm production dependency audit | PASS: 0 vulnerabilities |
| Go vulnerability scan | PASS under pinned Go 1.26.6: 0 reachable vulnerabilities |
| Secret/env/database-driver scan | PASS: only `.env.example` tracked; no frontend PostgreSQL package |
| OpenAPI lint | PASS |
| BPMN semantic lint | PASS |
| Stable trace verification | PASS: 39 BPMN tasks + 21 supporting actions |

## Security and scope notes

- Dashboard is authenticated for both roles; reports, audit, and finance remain Admin-only server-side.
- Read queries are parameterized, return defined fields only, and never expose password/session/token hashes or audit metadata blobs.
- Security headers, secure session cookies, CSRF/origin enforcement, RBAC, idempotency, guarded state transitions, maker-checker, and row-locked stock checks remain active.
- The process-local rate limiter is appropriate for the approved single-instance MVP and must be replaced with a shared limiter before horizontal scaling.
- No production deployment or out-of-scope accounting/logistics capability was introduced.

## Traceability implementation roots

- Go: `backend/internal/reporting/{application,adapter}` and `backend/internal/auth/adapter/http/middleware.go`
- Database: `backend/migrations/000007_reporting_indexes.*.sql`
- UI: `frontend/features/dashboard`, `frontend/app/page.tsx`
- Integration/E2E: `backend/test/integration/reporting_test.go`, `frontend/e2e/dashboard.spec.ts`
- Contracts: `api/openapi.yaml`, `.bpmnlintrc`, `scripts/verify-traceability.sh`
