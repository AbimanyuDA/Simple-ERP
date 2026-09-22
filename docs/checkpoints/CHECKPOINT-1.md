# Checkpoint 1 — Application Foundation

Status: **PASSED** on 2026-09-22. Phase 2 may proceed to Checkpoint 2.

## Delivered

- Standalone Go modular monolith and Next.js application under the approved `backend/` and `frontend/` roots.
- PostgreSQL migrations for users, sessions, one-time action tokens, audit, idempotency, document numbering, master data, warehouses, organization settings, and token sharing lifecycle.
- Guarded first-Admin CLI with stdin-only password, advisory lock, refusal after the first user, and audit.
- Argon2id credentials, opaque hash-at-rest access/refresh tokens, approved TTLs, rotation, reuse-family revocation, logout, CSRF synchronizer cookie/header, exact-origin check, and login rate limiting.
- Invitation and password-reset create/share/consume/revoke/expire-on-access flows. Raw tokens are returned once and are not persisted or audited.
- Admin RBAC for users, settings, and master-data mutations; user deactivation and password reset revoke sessions.
- Responsive application shell, login, activation/reset pages, Users, Settings, and Master Data workspaces.
- Root Compose/Make/CI tooling and initial OpenAPI 3.1 contract.

## Verification evidence

| Gate | Result |
|---|---|
| Go unit tests | PASS |
| Go race detector | PASS |
| PostgreSQL integration tests | PASS |
| Concurrent bootstrap | PASS; one success only |
| Empty migration up | PASS; version 4 clean |
| Full down/up on isolated database | PASS; version 4 clean after reapply |
| Frontend unit/component tests | PASS |
| TypeScript typecheck and ESLint | PASS |
| Next.js production build | PASS |
| Desktop and mobile login E2E | PASS |
| Backend/frontend Docker image build | PASS |
| Integrated health/UI smoke | PASS |
| OpenAPI structural validation | PASS |
| BPMN/mapping ID coverage | PASS; 39 BPMN tasks + 21 supporting actions |
| Secret-pattern and tracked-env scan | PASS; one harmless type-signature match reviewed |

## Migration and security notes

- Settings migration follows warehouse creation and references the seeded `MAIN` warehouse.
- Migration rollback was tested only on a dedicated disposable database; the development database was not destroyed.
- A failed initial E2E revealed a build-time Next.js rewrite pointed at container-local `localhost`. The build now receives `http://backend:8080`; the unchanged E2E then passed on desktop and mobile.
- No live secret or personal environment file is tracked. Local Compose credentials are explicitly development-only defaults.

## Traceability implementation roots

- Auth domain/application: `backend/internal/auth/domain`, `backend/internal/auth/application`
- Auth adapters: `backend/internal/auth/adapter/http`, `backend/internal/auth/adapter/postgres`
- Master data: `backend/internal/masterdata`
- UI: `frontend/features/auth`, `frontend/features/users`, `frontend/features/settings`, `frontend/features/master-data`
- Tests: colocated Go tests, `backend/test/integration`, `frontend/tests`, `frontend/e2e/access-lifecycle.spec.ts`

## Observations carried forward

- OpenAPI currently covers Checkpoint 1 and will be extended with each subsequent vertical slice.
- Audit records are written and testable; the searchable audit explorer is intentionally delivered in Checkpoint 4.
- Runtime rate limiting is process-local, appropriate for the approved single-instance MVP modular monolith. Horizontal scaling would require a shared limiter and is outside MVP.
