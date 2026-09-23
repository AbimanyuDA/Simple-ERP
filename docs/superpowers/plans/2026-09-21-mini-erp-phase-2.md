# Mini ERP Phase 2 MVP Implementation Plan

**Execution status:** completed through Checkpoint 4 on 2026-09-23. Evidence is recorded in `docs/checkpoints/` and `docs/RELEASE-CANDIDATE.md`. Production deployment remains excluded.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver one complete, secure, maintainable Mini ERP MVP through a single implementation phase with four internal checkpoints.

**Architecture:** `backend/` contains one Go modular monolith that applies practical Clean Architecture per domain and exposes a versioned REST API. `frontend/` contains one Next.js application and communicates with the backend only through REST; it never accesses PostgreSQL. One PostgreSQL database provides transaction boundaries for business state, stock ledger, AP/AR, idempotency, and audit; root Docker Compose provides the integrated local runtime. The checkpoints control implementation order and are not separate phases or projects.

**Tech Stack:** Go, REST/JSON, PostgreSQL, Next.js, TypeScript, Tailwind CSS, shadcn/ui, Docker Compose.

**Spec:** `docs/PHASE-1-DESIGN.md`, `docs/BUSINESS-RULES.md`, `docs/PROCESS-MAPPING.md`, `docs/ERD.md`, `docs/STATE-MACHINES.md`, `docs/API-CONTRACT.md`, and `docs/REVIEW-FINDINGS.md`.

## Global Constraints

- BPMN, business rules, state machines, ERD, API contract, and `PROCESS-MAPPING.md` are the source of truth.
- Stable BPMN task IDs and supporting action IDs must appear in use-case metadata, audit records, and automated-test names.
- Use one Go modular monolith, one Next.js application, one PostgreSQL database, and REST. Do not add microservices, a message broker, workflow engine, generic repository framework, CQRS framework, or dependency-injection framework.
- All Next.js source, styling, frontend tests, and browser E2E live under `frontend/`; all Go source, migrations, backend tests, and bootstrap tooling live under `backend/`. BPMN lives only in `docs/bpmn/`, design documentation only in `docs/`, and OpenAPI only in `api/openapi.yaml`.
- The frontend communicates with the backend only through REST and has no PostgreSQL driver or database credentials. Backend packages contain no frontend source. BPMN and documentation are never duplicated into source folders.
- Apply Clean Architecture pragmatically: domain rules have no HTTP/SQL dependency, application use cases own their ports, adapters translate HTTP/PostgreSQL, and wiring stays in `backend/cmd/api`.
- Maker-checker has no override. Stock reservation and negative stock are prohibited. Partial receipt, delivery, and payment are required.
- Purchase and sales invoices are consolidated after full fulfillment. Finance is AP/AR only; no GL or journal accounting.
- Posted stock and finance records are immutable. Corrections use mapped void/reversal commands with mandatory reason and audit.
- Stock movement, payment allocation, void/reversal, linked state/progress changes, and audit commit atomically in one database transaction.
- PR, PO, and SO close automatically when their fulfillment and payment conditions are satisfied. There is no manual close endpoint.
- Rejected PR/SO remain terminal and may only be copied to a new DRAFT document.
- Defaults are IDR, Asia/Jakarta, and one UI-default warehouse; the database supports multiple warehouses.
- Returns/refunds, transfer, tax engine, reservation, short-close, advanced cancellation, multi-company, external notification, GL, and production deployment remain out of scope.
- Do not execute any checkbox until this plan is approved. Once approved, checkpoints may proceed without user approval after every small task. Stop only for a changed business process, material design decision, security issue, or scope expansion.

---

## Delivery Model

Phase 2 is one continuous implementation phase:

| Internal checkpoint | Outcome | Depends on |
|---|---|---|
| 1. Application Foundation | Runnable secure application, access lifecycle, audit foundation, settings, and master data | Approved Phase 1 and this plan |
| 2. Core Operations | Complete Purchasing, Sales, and Inventory operations with stock ledger | Checkpoint 1 green |
| 3. Simple Finance | AP/AR, exact matching, partial payment, void/reversal, and automatic close | Checkpoint 2 green |
| 4. MVP Completion | Dashboard, reports, polished UI, full verification, and release candidate | Checkpoint 3 green |

The team records checkpoint results in the plan/CI evidence and continues when all acceptance criteria are green. A checkpoint failure is fixed within Phase 2; it does not create a new phase.

## Shared Implementation Loop

Every business capability follows the same small TDD loop:

1. Read the mapped BPMN/business-rule/state/API rows and confirm the stable trace ID.
2. Write a failing domain or application test named with `T-<TRACE-ID>-<NN>`.
3. Add the smallest migration/repository behavior required and prove it with PostgreSQL integration tests.
4. Implement the domain transition and application use case, then the REST adapter and OpenAPI contract.
5. Add the permission-aware UI state and a focused browser journey.
6. Assert authorization, audit contents, idempotency/concurrency behavior, and transaction rollback.
7. Update `PROCESS-MAPPING.md` with actual implementation paths without changing stable IDs.

Code reviews and commits should be small enough to revert safely, but they do not require a user approval gate unless one of the stop conditions above is reached.

## Security Baseline

- Passwords: Argon2id with unique salt; initial parameters 64 MiB, 3 iterations, parallelism 1, 32-byte output; benchmark on the target host and never reduce below the approved minimum.
- Sessions: opaque 256-bit access/refresh tokens. Store hashes only. Access TTL is 15 minutes; refresh idle TTL is 7 days and absolute TTL is 30 days.
- Cookies: `__Host-erp_access` and `__Host-erp_refresh`, `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`, no `Domain`; never browser storage.
- Rotation: refresh rotates both tokens. Reuse of an old refresh token revokes the entire token family. Logout revokes the current session; password reset and user deactivation revoke all user sessions.
- CSRF: per-session synchronizer token returned in a response body and required as `X-CSRF-Token` for unsafe methods, plus exact trusted-origin/Fetch Metadata validation and JSON-only mutation bodies.
- Public limits: login 5 attempts/15 minutes per email+IP; invitation/reset consumption 5/token fingerprint and 10/IP per 15 minutes; refresh 30/5 minutes per session+IP; authenticated default 120/minute/user.
- Invitation/reset tokens: at least 256 random bits, hashed at rest, one use, 24-hour expiry, raw value shown once, and redacted from DB, audit, application/access logs, telemetry, and analytics.
- First Admin: local `bootstrap-admin` CLI only, password from TTY/stdin, database advisory lock, refusal when any user exists, no default credential, and no HTTP route.

## Migration and Tooling Order

Migration filenames may be refined during implementation, but dependency order may not change:

1. identity/RBAC, sessions/action tokens, audit, idempotency, sequences;
2. units, products, suppliers, customers, warehouses, plus default warehouse seed;
3. organization settings and its foreign key to the existing warehouse;
4. purchase, sales, inventory documents and stock ledger;
5. supplier/customer invoices, payments, allocations, and reporting indexes.

Phase 2 also creates and maintains:

- `.env.example` containing names and safe examples only—never secrets;
- migration up/down tooling and an empty-database plus upgrade-path verification command;
- RBAC/master seed tooling separate from the guarded Admin bootstrap;
- `api/openapi.yaml` as the executable API contract;
- backend/frontend lint, format, typecheck, unit, integration, race, E2E, and traceability commands;
- CI that starts PostgreSQL, migrates from empty, runs the full verification set, and validates BPMN XML/mapping coverage.

---

### Checkpoint 1: Application Foundation

**Outcome:** Developers can start the stack locally, bootstrap exactly one initial Admin, authenticate securely, manage users/settings/master data, and inspect foundational audit records.

**Affected areas:** root tooling and CI; `backend/cmd`, `backend/internal/{auth,masterdata,audit,platform}`; initial migrations; `api/openapi.yaml`; `frontend/` auth, Users, Settings, Master Data, tests, and E2E.

| Concern | Required coverage |
|---|---|
| BPMN / action IDs | AUTH-USR-001–005, AUTH-PWD-001–005, CAP-AUTH-001, CAP-MDM-001, CAP-AUD-001; all `ACT-AUTH-*` mappings |
| Requirements | BR-GEN-001–010/013/015, BR-AUTH-001–012, master-data lifecycle rules |
| Database | Identity/RBAC → session/token/audit → master data/warehouse → organization settings, in that order |
| Go | Config, transaction boundary, auth/session, bootstrap CLI, RBAC, audit appender, organization settings, master-data use cases |
| REST / OpenAPI | Auth, CSRF, invitation/reset/revoke, session revocation, users, settings, products, units, suppliers, customers, warehouses |
| Next.js UI | Login/logout, activate/reset, user management, organization settings, responsive master-data CRUD/lifecycle |
| Authorization | Default-deny permission middleware; only Admin manages users/settings/master data; Operator receives read access only |
| Audit | Bootstrap, login/session revocation, invitation/reset lifecycle, user and master-data mutation; no secret values |
| Tests | Domain, API, PostgreSQL integration, permission/audit, UI, E2E access lifecycle, migration, bootstrap race/refusal, traceability |

**Work sequence**

- [ ] Establish Compose, environment loading, Go/Next.js skeletons, migration/seed commands, OpenAPI validation, Make targets, and CI verification.
- [ ] Prove migration ordering from an empty database and a down/up cycle; verify warehouse exists before organization settings references it.
- [ ] Implement one-time Admin bootstrap with advisory-lock concurrency tests and zero secret leakage.
- [ ] Implement Argon2id credentials, opaque rotating sessions, secure cookies, CSRF/origin checks, rate limits, logout, reset/deactivation revocation, and refresh-reuse detection.
- [ ] Implement invitation/reset create-share-consume-revoke-expire flows with stable trace IDs and generic public errors.
- [ ] Implement RBAC, audit foundation, organization settings, and master-data lifecycle using practical domain/application/adapter boundaries.
- [ ] Build accessible responsive UI states, including loading, empty, validation, conflict, forbidden, and session-expired handling.

**Acceptance criteria**

- [ ] `docker compose up` starts web, API, and PostgreSQL from documented non-secret configuration.
- [ ] A fresh database migrates successfully; rollback/reapply succeeds; no migration references the default warehouse before it exists.
- [ ] Bootstrap succeeds exactly once under concurrent attempts, takes no password through argv/env, and exposes no HTTP route.
- [ ] Cookie, CSRF, expiry, rotation, reuse, rate-limit, and immediate-revocation tests pass; no raw credential/token appears in persistence or captured logs/audit.
- [ ] Admin/Operator permission boundaries and all invitation/reset state transitions pass domain, API, integration, authorization, audit, UI, and E2E tests.
- [ ] Every mapped Auth/action ID appears in use-case metadata, audit assertions, tests, OpenAPI, and traceability checks.

---

### Checkpoint 2: Core Operations

**Outcome:** Users can complete Purchasing, Sales, and Inventory operations through fulfillment while every stock change remains source-linked, atomic, and non-negative.

**Affected areas:** `backend/internal/{purchasing,sales,inventory}` and their migrations/adapters; `frontend/` Purchasing, Sales, Inventory, tests, and E2E; operational OpenAPI routes.

| Concern | Required coverage |
|---|---|
| BPMN / action IDs | PUR-PR-001–003, PUR-PO-001/002, PUR-GR-001/002, SAL-SO-001–005, SAL-DLV-001/002, INV-ADJ-001–005; mapped cancel/copy/reversal actions |
| Requirements | BR-GEN-001–015 as applicable, BR-P2P-001–007/012, BR-O2C-001–006/010, BR-INV-001–008 |
| Database | PR/PO/receipt, SO/delivery, adjustment, immutable stock movement lines, rebuildable balances, source/idempotency uniqueness |
| Go | State machines, maker-checker, copy/cancel, partial fulfillment, inventory posting/reversal, optimistic locking and idempotency |
| REST / OpenAPI | Draft CRUD plus submit/review/issue/post/cancel/copy/reverse actions and stock reads |
| Next.js UI | Purchasing, Sales, Inventory workspaces; approval inbox; partial quantity forms; conflict and stock feedback |
| Authorization | Operator operational commands; Admin approvals/reversals; creator cannot approve or reject own document |
| Audit | Every transition plus source/reversal IDs, quantity changes, previous/new state, actor, and stable trace ID |
| Tests | State/domain, API, PostgreSQL, maker-checker, rollback, optimistic-lock, idempotency, concurrent stock, UI/E2E, traceability |

**Work sequence**

- [ ] Implement PR and SO draft/submit/review/cancel/copy workflows with terminal rejection and maker-checker denial.
- [ ] Implement PR-to-PO conversion, PO issue, partial goods receipts, and source-linked positive stock posting.
- [ ] Implement stock validation without reservation, partial deliveries, and row-locked negative-stock-safe posting.
- [ ] Implement inventory count/adjustment approval, zero-difference completion, posting, cancellation, and inverse-movement reversal.
- [ ] Build responsive operational UI with clear remaining quantities, current warehouse stock, disabled/hidden unauthorized actions, and server-side enforcement.
- [ ] Reconcile stock balances from immutable movement lines and verify a failed command leaves neither partial business writes nor a false success-audit row; security failure logging remains separate and redacted.

**Acceptance criteria**

- [ ] Happy, rejection/copy, cancellation, partial/final fulfillment, stale-version, duplicate-request, and reversal scenarios pass, including downstream-invoice and negative-result reversal rejection.
- [ ] Same-creator approval/rejection always returns the mapped separation-of-duties error with no write.
- [ ] Concurrent delivery/negative adjustment cannot make stock negative; receipt/delivery/adjustment and ledger/projection/audit commit or roll back together.
- [ ] No reservation table or implicit reserved quantity exists; stock is revalidated immediately before posting.
- [ ] Domain, API, integration, authorization, audit, UI, E2E, race/concurrency, and traceability suites for the checkpoint are green.

---

### Checkpoint 3: Simple Finance

**Outcome:** AP/AR invoices and payments complete P2P/O2C without introducing accounting-ledger scope.

**Affected areas:** `backend/internal/finance`, finance migrations/adapters, P2P/O2C orchestration, `frontend/` Finance UI/tests/E2E, and OpenAPI.

| Concern | Required coverage |
|---|---|
| BPMN / action IDs | PUR-INV-001–003, PUR-PAY-001/002, PUR-CLOSE-001, SAL-INV-001, SAL-PAY-001/002, SAL-SO-006; all `ACT-FIN-*-VOID` |
| Requirements | BR-P2P-008–011, BR-O2C-007–009, BR-FIN-001–005, BR-GEN-004/005/010/015 |
| Database | One consolidated invoice per PO/SO, invoice lines, payments, allocations, outstanding amounts, void/reversal references |
| Go | Exact three-way match, exception resolution, partial allocation, void/reversal, and automatic PR/PO/SO close |
| REST / OpenAPI | Supplier/customer invoice and payment actions; no manual close endpoint |
| Next.js UI | Match result/exception, payable/receivable detail, partial payment, mandatory-reason void, completion status |
| Authorization | Admin-only finance mutation/void; report/read capabilities remain separate |
| Audit | Match evidence, allocation deltas, outstanding before/after, reason, reversal reference, automatic close trace IDs |
| Tests | Exact-match, consolidation, partial/overpayment, void/reversal rollback, auto-close, auth/audit, UI/E2E, traceability |

**Work sequence**

- [ ] Implement one consolidated supplier invoice only after full receipt and enforce zero-tolerance quantity/price matching.
- [ ] Implement supplier partial payments, outstanding balance, invoice/payment void, and atomic automatic PO/PR close after full payment.
- [ ] Implement one consolidated sales invoice only after full delivery, customer partial payments, outstanding balance, and void/reversal.
- [ ] Trigger SO completion atomically from final customer-payment posting; do not expose manual close.
- [ ] Build Finance UI that makes posted immutability, outstanding amount, allocation history, match exception, and reversal provenance explicit.

**Acceptance criteria**

- [ ] Early or duplicate consolidated invoice creation fails; any quantity/price difference produces a match exception.
- [ ] Partial payments update outstanding correctly; overpayment, currency mismatch, invalid date, edit/delete of posted records, and unauthorized void fail without partial writes.
- [ ] Payment void requires Admin/reason, restores invoice outstanding/status, and reopens auto-closed PR/PO/SO when needed; invoice void rejects active allocations, records reversal/audit, and permits only a linked replacement invoice.
- [ ] PR/PO/SO close automatically only after all approved fulfillment and payment conditions are satisfied; no short-close/manual-close route exists.
- [ ] Full P2P and O2C domain, API, integration, authorization, audit, UI, E2E, race/concurrency, and traceability suites are green.

---

### Checkpoint 4: MVP Completion

**Outcome:** The complete MVP is understandable for UMKM users, observable, regression-tested, and ready for release-candidate review—not production deployment.

**Affected areas:** backend reporting/audit read models, `frontend/` Dashboard/Reports/Audit UI and test suites, accessibility/responsiveness, CI gates, documentation, and release evidence.

| Concern | Required coverage |
|---|---|
| BPMN / capability IDs | CAP-RPT-001, CAP-AUD-001 and full Level 0–2 regression; all 39 BPMN task IDs and supporting action IDs |
| Requirements | Dashboard/report/audit scope plus every approved business/security rule and explicit exclusion |
| Database | Read-only reporting indexes/views; no duplicate business-state source |
| Go | Purpose-built read queries only; no generic reporting framework or state mutation |
| REST / OpenAPI | Dashboard, purchase/sales/inventory/AP/AR reports, audit explorer; contract fully validated |
| Next.js UI | Consistent sidebar, responsive forms/tables, clear statuses, accessible focus/labels/errors, actionable dashboard |
| Authorization | Scoped Operator dashboard; Admin reports/audit; no sensitive audit/credential exposure |
| Audit | Searchable immutable record with actor/system, action, entity, states, trace ID, request ID, and redacted snapshots |
| Tests | Full unit/API/integration/auth/audit/UI/E2E/race/security/migration/traceability regression |

**Work sequence**

- [ ] Implement actionable dashboard metrics, operational reports, AP/AR aging, and source-linked audit explorer with purpose-built read queries.
- [ ] Complete responsive/accessibility/error-state polish across desktop and mobile without imitating a generic template dashboard.
- [ ] Run security review for authorization bypass, IDOR, CSRF, cookie/session handling, token/log leakage, injection, mass assignment, rate limits, and security headers.
- [ ] Verify migrations on empty and representative upgraded databases, including rollback where safe and backup/restore rehearsal for release data.
- [ ] Validate all BPMN XML, stable ID uniqueness, Level 2-to-mapping coverage, supporting action coverage, OpenAPI routes, audit metadata, and test-family presence.
- [ ] Produce release-candidate evidence: versions, checksums/build output, known limitations, test results, migration result, and unresolved risks.

**Acceptance criteria**

- [ ] Dashboard, reports, and audit explorer show authorized, reconciled data and remain usable at supported mobile/desktop widths.
- [ ] Full backend tests pass with race detection; frontend lint/typecheck/unit tests and critical E2E journeys pass; no flaky test is silently retried or skipped.
- [ ] Security review has no unresolved critical/high finding, no raw secret/token leakage, and all state-changing routes enforce authentication, authorization, CSRF, idempotency, and state/version rules as applicable.
- [ ] Migration verification, stock-ledger reconciliation, AP/AR outstanding reconciliation, BPMN XML validation, OpenAPI validation, and traceability coverage are green.
- [ ] Out-of-scope capabilities are absent and documented; release-candidate review explicitly states that production deployment requires a new plan and approval.

---

## Verification Gate Used by Every Checkpoint

A checkpoint cannot be marked complete until all applicable categories pass:

1. domain/application unit tests;
2. REST API contract and handler tests;
3. PostgreSQL repository/transaction/migration integration tests;
4. authorization and maker-checker tests;
5. audit completeness and secret-redaction tests;
6. Next.js component/accessibility/responsive tests;
7. browser E2E happy and critical failure journeys;
8. BPMN/stable-ID/PROCESS-MAPPING traceability checks;
9. `go test -race` for affected backend packages and concurrency scenarios.

The exact commands are exposed through stable Make targets such as `make lint`, `make typecheck`, `make test`, `make test-race`, `make test-integration`, `make test-e2e`, `make migrate-verify`, `make openapi-verify`, and `make traceability-verify`; CI invokes the same targets.

## Phase 2 Approval and Stop Conditions

This file preserves the approved implementation plan. Completion evidence, rather than retroactive checkbox edits, is recorded in the four checkpoint reports.

After approval, work proceeds across the four checkpoints without requesting approval for each small task. Stop and request direction only when:

1. BPMN/business process or an approved state transition must change;
2. a material architecture/security decision differs from this plan;
3. a vulnerability cannot be safely resolved inside the approved scope;
4. an excluded capability or production deployment becomes necessary.

Production infrastructure, hosting, domain/TLS setup, live secrets, observability operations, data import, backup policy, and go-live execution require a separate deployment plan and explicit approval.
