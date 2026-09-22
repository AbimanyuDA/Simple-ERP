# Proposed Repository Structure

Struktur ini adalah baseline Phase 1 dan rencana Phase 2 yang telah disetujui. `Simple-ERP` adalah Git repository mandiri; seluruh source dan tooling proyek berada di bawah root ini dan tidak bergantung pada repository induk atau proyek tetangga.

```text
Simple-ERP/
├── frontend/                        # seluruh Next.js application
│   ├── app/                         # routes dan layouts
│   ├── components/                  # shared UI dan shadcn/ui
│   ├── features/                    # auth, purchasing, sales, inventory, finance
│   ├── lib/                         # REST client dan frontend utilities
│   ├── tests/                       # frontend unit/component tests
│   ├── e2e/                         # browser E2E tests
│   ├── public/
│   ├── package.json
│   └── tsconfig.json
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   │   └── main.go              # composition root only
│   │   ├── migrate/                  # migration CLI
│   │   └── bootstrap-admin/          # one-time local first-Admin CLI
│   ├── internal/
│   │   ├── auth/
│   │   ├── masterdata/
│   │   ├── purchasing/
│   │   ├── inventory/
│   │   ├── sales/
│   │   ├── finance/
│   │   ├── reporting/
│   │   ├── audit/
│   │   └── platform/                # technical details shared by adapters
│   ├── migrations/
│   ├── test/
│   │   ├── acceptance/
│   │   ├── integration/
│   │   └── fixtures/
│   ├── go.mod
│   └── go.sum
├── api/
│   └── openapi.yaml
├── docs/
│   ├── bpmn/
│   │   ├── l0-enterprise-landscape.bpmn
│   │   ├── l1-core-processes.bpmn
│   │   ├── l2-access-lifecycle.bpmn
│   │   ├── l2-procure-to-pay.bpmn
│   │   ├── l2-order-to-cash.bpmn
│   │   └── l2-inventory-adjustment.bpmn
│   ├── PHASE-1-DESIGN.md
│   ├── BUSINESS-RULES.md
│   ├── PROCESS-MAPPING.md
│   ├── ERD.md
│   ├── STATE-MACHINES.md
│   ├── API-CONTRACT.md
│   ├── REVIEW-FINDINGS.md
│   ├── TRACEABILITY-COVERAGE.md
│   └── superpowers/plans/2026-09-21-mini-erp-phase-2.md
├── compose.yaml                     # Phase 2+
├── .env.example                     # documented non-secret configuration
├── .github/workflows/verify.yml     # lint, tests, race, migration, traceability
├── Makefile                         # Phase 2+
└── README.md
```

`frontend/` berkomunikasi dengan `backend/` hanya melalui REST `/api/v1`. Frontend tidak memiliki driver/database credential dan tidak mengakses PostgreSQL. Backend tidak menyimpan komponen, styling, atau source Next.js. BPMN hanya berada di `docs/bpmn/`, sedangkan dokumen desain hanya berada di `docs/`; tidak ada salinan di folder source code.

`compose.yaml` dan `Makefile` di root menjadi entry point terpadu untuk frontend, backend, dan PostgreSQL. Root tidak menampung business source code; hanya file koordinasi bersama seperti `README.md`, `.gitignore`, `.env.example`, Compose, Makefile, dan konfigurasi CI.

The Auth domain additionally owns `UserActionToken` for invitation/password reset and opaque access/refresh sessions. Raw access, refresh, CSRF, invitation, and reset tokens never cross into persistence or audit models. The bootstrap command imports Auth application use cases; it does not bypass the domain or expose an HTTP route.

## Backend module template

```text
internal/<domain>/
├── domain/
│   ├── entity.go                    # aggregate/entity and invariants
│   ├── state.go                     # allowed transitions
│   ├── value.go                     # domain value objects
│   └── error.go                     # framework-independent errors
├── application/
│   ├── command/                     # mutating use cases, one operation/file
│   ├── query/                       # read use cases
│   └── port/                        # interfaces required by use cases
└── adapter/
    ├── http/                        # request DTO, handler, presenter, routes
    └── postgres/                    # repository and row mapping
```

Use case names mirror BPMN actions, for example:

```text
internal/purchasing/application/command/
├── create_purchase_request.go       # PUR-PR-001
├── submit_purchase_request.go       # PUR-PR-002
├── review_purchase_request.go       # PUR-PR-003
└── convert_purchase_request.go      # PUR-PO-001
```

## Dependency rules

Allowed:

```text
cmd/api → adapter → application → domain
adapter/postgres → application ports + domain
process application use case → another module's narrow application port
```

Forbidden:

```text
domain → application/adapter/platform
application → HTTP router/sql driver/PostgreSQL DTO
module adapter → another module's tables or concrete repository
frontend → database
```

## Cross-module transaction pattern

For operations such as posting a goods receipt, the Purchasing application use case owns orchestration and depends on narrow ports such as `StockMovementPoster`, `PurchaseRepository`, `AuditAppender`, and `UnitOfWork`. Concrete PostgreSQL adapters are wired once in `cmd/api`. Interfaces are defined by the consuming application package, not in a global `interfaces` package.

No generic repository, base entity, event bus, CQRS framework, or dependency-injection framework is introduced in the MVP. Read-only dashboard queries may use purpose-built SQL read models because they do not contain business state transitions.

Keep this template practical: a domain may combine small files/packages until their responsibilities genuinely diverge. Clean Architecture means dependency direction and testable business rules, not one folder or interface per noun.

## Test placement

- `backend/internal/<domain>/domain/*_test.go`: state transition and invariant unit tests without database.
- `backend/internal/<domain>/application/*_test.go`: use case tests with small fake ports; test IDs match `PROCESS-MAPPING.md`.
- `backend/internal/<domain>/adapter/http/*_test.go`: request/response, auth, status codes.
- `backend/internal/<domain>/adapter/postgres/*_test.go`: repository integration tests against PostgreSQL.
- `backend/test/acceptance`: BPMN scenario tests spanning API and database.
- `frontend/tests`: component, accessibility, state, and REST-client tests.
- `frontend/e2e`: browser journeys against the integrated Compose stack.

## Clean Architecture diagnostic target

The future implementation must score 10/10 on these checks: business rules testable without DB/server, all dependencies inward, database swappable without domain changes, use cases independent of HTTP, framework confined to adapters, module graph acyclic, and concrete wiring only in `cmd/api`.
