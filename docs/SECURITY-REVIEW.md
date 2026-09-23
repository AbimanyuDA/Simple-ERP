# Phase 2 Security Review

Review date: 2026-09-23. Result: **PASS for local release-candidate review; no unresolved critical or high finding.**

## Reviewed controls

| Area | Evidence / outcome |
|---|---|
| Authentication | Argon2id credentials; opaque hashed sessions; access/refresh rotation; reuse and reset/deactivation revocation tests pass |
| Cookies and CSRF | `__Host-*`, Secure, HttpOnly, SameSite Lax; synchronizer token and exact trusted-origin checks on unsafe authenticated requests |
| Authorization / IDOR | Principal derived from session, default protected route groups, Admin-only finance/reports/audit/users/settings/master data; Operator denial E2E passes |
| State integrity | Domain transitions, maker-checker, optimistic checks, transaction boundaries, partial-fulfillment limits, and negative-stock guards are server-enforced |
| Replay / abuse | Idempotency keys on operational/finance mutations; public endpoint limits plus authenticated 120/minute/user limit |
| Injection / mass assignment | Parameterized pgx SQL and explicit request DTOs/command switches; no generic model binding or dynamic table name from unvalidated input |
| Secret leakage | Raw action/session tokens are not persisted or audited; audit read model excludes metadata/hash/token fields; repository secret-pattern scan passed |
| Browser hardening | `nosniff`, frame denial, no-referrer, restrictive API CSP, JSON mutation expectations, server timeouts |
| Dependencies | npm production audit: 0; govulncheck under Go 1.26.6 and x/text 0.39.0: 0 reachable vulnerabilities |
| Data integrity | Stock and audit ledgers reject update/delete; AP/AR and stock reconciliation return zero mismatches |

## Findings resolved during review

1. Go 1.26.1 and x/text 0.38.0 produced reachable vulnerability findings. The build image is now pinned to Go 1.26.6 and x/text to 0.39.0; the repeat scan is clean.
2. Dashboard mobile layout inherited an oversized grid minimum, causing an overlaid hit target. `.main` now has `min-width: 0`; desktop/mobile E2E passes.

## Residual risks / production prerequisites

- Rate limits and session/application runtime are single-instance. Horizontal scaling needs a shared limiter and an explicit deployment architecture review.
- TLS termination, WAF/reverse-proxy policy, secret manager, centralized logs/alerts, backup retention, disaster recovery objectives, and penetration testing are deployment concerns not configured in this repository.
- CSP is intentionally API-focused. A production web CSP and transport headers must be finalized at the edge with the deployment plan.
- Reports currently use bounded MVP queries without pagination/export; production data-volume limits must be measured before go-live.
