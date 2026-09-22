# REST API Contract Proposal

Base path: `/api/v1`. JSON uses `snake_case`. Login, refresh, invitation activation, and password-reset consumption are public token endpoints; all other endpoints require the approved opaque cookie session. Authentication credentials are never stored in browser `localStorage` or `sessionStorage`. This document defines the contract shape; the future canonical machine-readable contract will be `api/openapi.yaml` during Phase 2.

## Conventions

### Success

```json
{
  "data": {
    "id": "018f...",
    "status": "SUBMITTED",
    "version": 2
  }
}
```

Collections use `page` and `per_page` (default 20, maximum 100) with `meta.total`, `meta.page`, `meta.per_page`, and `meta.total_pages`. Supported common filters are `status`, `q`, `created_at[after]`, `created_at[before]`, and `sort`.

### Error

```json
{
  "error": {
    "code": "state_conflict",
    "message": "Purchase request must be SUBMITTED to approve",
    "details": [
      { "field": "status", "code": "invalid_transition", "message": "Current status is DRAFT" }
    ],
    "request_id": "req_01..."
  }
}
```

| Status | Use |
|---:|---|
| 200 | read/update/action succeeded |
| 201 | resource created; includes `Location` |
| 204 | deactivation/void command with no representation when chosen |
| 400 | malformed request |
| 401 | missing/invalid authentication |
| 403 | missing capability or maker-checker violation |
| 404 | resource not found or not visible |
| 409 | state/version/idempotency/duplicate conflict |
| 422 | semantic validation failure |
| 429 | authentication or API rate limit exceeded; includes `Retry-After` |
| 500 | unexpected error with no internal detail exposed |

Mutating requests carry `Idempotency-Key`; updates/transitions carry `If-Match: "<version>"`. Cookie-authenticated unsafe methods also carry `X-CSRF-Token` and `Content-Type: application/json`. The response exposes `ETag: "<new-version>"`.

## Authentication security contract

- Password hashing: Argon2id, unique 16-byte-or-larger salt, 64 MiB memory, 3 iterations, parallelism 1, 32-byte output. Deployment benchmarking may increase these parameters but may not reduce them below the documented OWASP minimum.
- Session model: opaque 256-bit access and refresh tokens. Only their hashes are persisted. Access expires in 15 minutes; refresh has a 7-day idle and 30-day absolute expiry.
- Cookie transport: `__Host-erp_access` and `__Host-erp_refresh` use `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`, and no `Domain`. Production requires HTTPS.
- Rotation and revocation: every refresh rotates both tokens. Reuse of a rotated refresh token revokes the token family. Logout revokes the current session; password reset and user deactivation revoke all sessions for the user.
- CSRF: `GET /auth/csrf` returns a per-session synchronizer token in the response body. Every unsafe cookie-authenticated request must send it through `X-CSRF-Token`; the API also validates the exact configured `Origin` with `Referer` fallback and rejects cross-site Fetch Metadata.
- Public endpoint limits: login is limited to 5 attempts per 15 minutes per normalized email and IP; invitation/reset consumption to 5 attempts per 15 minutes per token fingerprint and 10 per IP; refresh to 30 attempts per 5 minutes per session and IP. Authenticated API traffic defaults to 120 requests per minute per user. Limits return `429` with `Retry-After`.
- Invitation/reset links: tokens contain at least 256 random bits, expire after 24 hours, are single-use, and are stored only as hashes. Raw values are returned once and are redacted from database fields, audit payloads, structured/application/access logs, error telemetry, and URLs recorded by analytics. Token pages use `Referrer-Policy: no-referrer`.
- First Admin: created only through the local `bootstrap-admin` CLI. It has no HTTP route, accepts the password interactively or via stdin (never a command argument or environment variable), locks bootstrap transactionally, and refuses to run once any user exists.

Security decisions are based on the OWASP [Password Storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html), [Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html), [CSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html), and [Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html) guidance. The project uses stronger Argon2id settings than the published minimum and must benchmark them on its actual deployment host.

## Authentication and users

| Method | Path | Trace ID / capability | Result |
|---|---|---|---|
| POST | `/auth/login` | public | access + rotated refresh token |
| POST | `/auth/refresh` | valid refresh | rotated token pair |
| POST | `/auth/logout` | authenticated | revoke current access/refresh session and clear cookies |
| GET | `/auth/csrf` | authenticated | return per-session CSRF token; no credential value |
| POST | `/auth/activate-invitation` | valid one-time token | set password; consume invitation; `INVITED → ACTIVE` |
| POST | `/auth/reset-password` | valid one-time token | replace password; consume reset; revoke all sessions |
| GET | `/me` | authenticated | identity and effective permissions |
| GET | `/users` | `users.read` | paginated users |
| POST | `/user-invitations` | `users.manage` | create INVITED user and return raw invitation link once |
| POST | `/user-invitations/{id}/mark-shared` | `users.manage` | record manual secure sharing |
| POST | `/user-invitations/{id}/revoke` | ACT-AUTH-INVITE-REVOKE | `users.manage`; revoke unused invitation |
| POST | `/users/{id}/password-resets` | `users.manage` | create reset request and return raw link once |
| POST | `/password-resets/{id}/mark-shared` | `users.manage` | record manual secure sharing |
| POST | `/password-resets/{id}/revoke` | ACT-AUTH-RESET-REVOKE | `users.manage`; revoke unused reset |
| GET | `/users/{id}` | `users.read` | user detail |
| PATCH | `/users/{id}` | `users.manage` | update allowed profile/roles |
| POST | `/users/{id}/activate` | `users.manage` | set ACTIVE |
| POST | `/users/{id}/deactivate` | `users.manage` | set INACTIVE and revoke sessions |
| POST | `/users/{id}/revoke-sessions` | ACT-AUTH-SESSION-REVOKE | `users.manage`; revoke every active session for the user |
| GET | `/settings/organization` | `settings.read` | currency, timezone, and default warehouse |
| PATCH | `/settings/organization` | `settings.manage` | change currency, timezone, or default warehouse; maker-checker is not an API setting |

The raw invitation/reset token is never returned by list/detail endpoints and never appears in audit or logs. Both public consumption endpoints use a generic `invalid_or_expired_token` error. Expiry is evaluated on access; the first observation after the 24-hour deadline atomically records `ACT-AUTH-INVITE-EXPIRE` or `ACT-AUTH-RESET-EXPIRE` without revealing the token.

## Master data

The same REST pattern applies to `/products`, `/suppliers`, `/customers`, `/warehouses`, and `/units`:

| Method | Path | Capability | Notes |
|---|---|---|---|
| GET | `/{resource}` | `masterdata.read` | paginate/filter/search |
| POST | `/{resource}` | `masterdata.manage` | create ACTIVE record |
| GET | `/{resource}/{id}` | `masterdata.read` | detail |
| PATCH | `/{resource}/{id}` | `masterdata.manage` | update with `If-Match` |
| POST | `/{resource}/{id}/activate` | `masterdata.manage` | lifecycle command |
| POST | `/{resource}/{id}/deactivate` | `masterdata.manage` | reject when business invariant requires active record |

## Purchasing

### Purchase requests

| Method | Path | BPMN ID | Capability / transition |
|---|---|---|---|
| GET | `/purchase-requests` | — | `purchase.read` |
| POST | `/purchase-requests` | PUR-PR-001 | `purchase.create`; `[*] → DRAFT` |
| GET | `/purchase-requests/{id}` | — | `purchase.read` |
| PATCH | `/purchase-requests/{id}` | PUR-PR-001 | `purchase.edit`; DRAFT only |
| POST | `/purchase-requests/{id}/submit` | PUR-PR-002 | `purchase.submit`; DRAFT → SUBMITTED |
| POST | `/purchase-requests/{id}/approve` | PUR-PR-003 | `purchase.approve`; SUBMITTED → APPROVED |
| POST | `/purchase-requests/{id}/reject` | PUR-PR-003 | `purchase.approve`; SUBMITTED → REJECTED; body requires reason |
| POST | `/purchase-requests/{id}/cancel` | ACT-PUR-PR-CANCEL | `purchase.edit`; DRAFT → CANCELLED; reason required |
| POST | `/purchase-requests/{id}/copy` | ACT-PUR-PR-COPY | `purchase.create`; terminal REJECTED → new independent DRAFT |
| POST | `/purchase-requests/{id}/purchase-order` | PUR-PO-001 | `purchase.create`; APPROVED → CONVERTED_TO_PO and create PO DRAFT |

### Purchase orders and receiving

| Method | Path | BPMN ID | Capability / transition |
|---|---|---|---|
| GET | `/purchase-orders` | — | `purchase.read` |
| GET | `/purchase-orders/{id}` | — | `purchase.read` |
| PATCH | `/purchase-orders/{id}` | PUR-PO-001 | `purchase.edit`; DRAFT only |
| POST | `/purchase-orders/{id}/issue` | PUR-PO-002 | `purchase.issue`; DRAFT → ISSUED |
| POST | `/purchase-orders/{id}/cancel` | ACT-PUR-PO-CANCEL | `purchase.issue`; DRAFT/ISSUED before receipt → CANCELLED; reason required |
| POST | `/purchase-orders/{id}/goods-receipts` | PUR-GR-001 | `goods_receipt.create`; creates DRAFT receipt |
| POST | `/goods-receipts/{id}/cancel` | ACT-PUR-GR-CANCEL | `goods_receipt.create`; DRAFT → CANCELLED; reason required |
| POST | `/goods-receipts/{id}/post` | PUR-GR-002 | `goods_receipt.post`; DRAFT → POSTED, stock movement |
| POST | `/goods-receipts/{id}/reverse` | ACT-PUR-GR-REVERSE | `goods_receipt.reverse`; POSTED → REVERSED through paired movement; reason required |

### Supplier invoices and payments

| Method | Path | BPMN ID | Capability / transition |
|---|---|---|---|
| POST | `/supplier-invoices` | PUR-INV-001 | `finance.manage`; create MATCH_PENDING |
| POST | `/supplier-invoices/{id}/match` | PUR-INV-002 | `finance.manage`; validate 3-way match |
| POST | `/supplier-invoices/{id}/resolve-match` | PUR-INV-003 | `finance.manage`; exception → pending |
| POST | `/supplier-invoices/{id}/post` | PUR-INV-002 | internal or `finance.manage`; MATCH_PENDING → POSTED when matched |
| POST | `/supplier-invoices/{id}/void` | ACT-FIN-SINV-VOID | `finance.manage`; POSTED with no active allocation → VOID via reversal |
| POST | `/supplier-payments` | PUR-PAY-001 | `finance.manage`; create DRAFT |
| POST | `/supplier-payments/{id}/post` | PUR-PAY-002 | `finance.manage`; DRAFT → POSTED; allocations applied |
| POST | `/supplier-payments/{id}/void` | ACT-FIN-SPAY-VOID | `finance.manage`; POSTED → VOID via reversal |

Invoice and payment void bodies require `{ "reason": "..." }`. The actor must be Admin; a second approver is not required. Payment void restores invoice outstanding/status and atomically reopens automatically closed PR/PO/SO when full-payment conditions no longer hold. An invoice with active allocations returns `409 dependent_payment_exists`; linked payments must be voided first. A replacement consolidated invoice can then be created with an internal `replaces_invoice_id` link. The same rule applies to supplier and customer documents.

Every mapped `/cancel` or `/reverse` body also requires `{ "reason": "..." }`. Basic cancel is accepted only from the states listed in `STATE-MACHINES.md`; reversal is a correction command that preserves the original posted document and creates linked inverse effects atomically.
Goods-receipt and delivery reversal returns `409 downstream_document_exists` while a related non-VOID invoice exists. Every inverse stock posting revalidates non-negative projected stock.

## Sales

### Sales orders and delivery

| Method | Path | BPMN ID | Capability / transition |
|---|---|---|---|
| GET | `/sales-orders` | — | `sales.read` |
| POST | `/sales-orders` | SAL-SO-001 | `sales.create`; create DRAFT |
| GET | `/sales-orders/{id}` | — | `sales.read` |
| PATCH | `/sales-orders/{id}` | SAL-SO-001 | `sales.edit`; DRAFT only |
| POST | `/sales-orders/{id}/submit` | SAL-SO-002 | `sales.submit`; DRAFT → SUBMITTED |
| POST | `/sales-orders/{id}/validate-stock` | SAL-SO-003 | internal validation, no arbitrary transition |
| POST | `/sales-orders/{id}/approve` | SAL-SO-004 | `sales.approve`; SUBMITTED → APPROVED after stock recheck |
| POST | `/sales-orders/{id}/reject` | SAL-SO-004 | `sales.approve`; SUBMITTED → REJECTED |
| POST | `/sales-orders/{id}/cancel` | ACT-SAL-SO-CANCEL | `sales.edit`; DRAFT/APPROVED before delivery → CANCELLED; reason required |
| POST | `/sales-orders/{id}/copy` | ACT-SAL-SO-COPY | `sales.create`; terminal REJECTED → new independent DRAFT |
| POST | `/sales-orders/{id}/confirm` | SAL-SO-005 | internal notification marker; APPROVED only |
| POST | `/sales-orders/{id}/deliveries` | SAL-DLV-001 | `delivery.create`; create DRAFT |
| POST | `/deliveries/{id}/cancel` | ACT-SAL-DLV-CANCEL | `delivery.create`; DRAFT → CANCELLED; reason required |
| POST | `/deliveries/{id}/post` | SAL-DLV-002 | `delivery.post`; stock decrease and order progress |
| POST | `/deliveries/{id}/reverse` | ACT-SAL-DLV-REVERSE | `delivery.reverse`; POSTED → REVERSED through paired movement; reason required |

### Customer invoices and receipts

| Method | Path | BPMN ID | Capability / transition |
|---|---|---|---|
| POST | `/sales-orders/{id}/sales-invoice` | SAL-INV-001 | `finance.manage` or internal; create and post consolidated invoice once after full delivery |
| POST | `/sales-invoices/{id}/void` | ACT-FIN-CINV-VOID | `finance.manage`; POSTED with no active allocation → VOID via reversal |
| POST | `/customer-payments` | SAL-PAY-001 | `finance.manage`; create DRAFT |
| POST | `/customer-payments/{id}/post` | SAL-PAY-002 | `finance.manage`; allocate payment |
| POST | `/customer-payments/{id}/void` | ACT-FIN-CPAY-VOID | `finance.manage`; reverse allocation |

`SAL-SO-006` is an internal atomic follow-on of final customer-payment posting. There is no manual close endpoint.

## Inventory

| Method | Path | BPMN ID | Capability / transition |
|---|---|---|---|
| GET | `/inventory/balances` | — | `inventory.read`; warehouse/product filters |
| GET | `/inventory/movements` | — | `inventory.read`; immutable ledger |
| GET | `/inventory/movements/{id}` | — | `inventory.read` |
| POST | `/inventory-adjustments` | INV-ADJ-001 | `inventory.adjust`; create DRAFT with snapshot |
| PATCH | `/inventory-adjustments/{id}` | INV-ADJ-001 | `inventory.adjust`; DRAFT only |
| POST | `/inventory-adjustments/{id}/submit` | INV-ADJ-002 | DRAFT → SUBMITTED |
| POST | `/inventory-adjustments/{id}/approve` | INV-ADJ-003 | `inventory.approve`; SUBMITTED → APPROVED after snapshot check |
| POST | `/inventory-adjustments/{id}/reject` | INV-ADJ-003 | SUBMITTED → REJECTED |
| POST | `/inventory-adjustments/{id}/cancel` | ACT-INV-ADJ-CANCEL | `inventory.adjust`; DRAFT → CANCELLED; reason required |
| POST | `/inventory-adjustments/{id}/post` | INV-ADJ-004 | internal/atomic follow-on; APPROVED → POSTED or COMPLETED |
| POST | `/inventory-adjustments/{id}/reverse` | ACT-INV-ADJ-REVERSE | `inventory.approve`; COMPLETED → REVERSED using inverse movement; reason required |

## Finance, reporting, and audit reads

| Method | Path | Capability | Notes |
|---|---|---|---|
| GET | `/supplier-invoices` | `finance.read` | filter status/due date/supplier |
| GET | `/supplier-payments` | `finance.read` | filter date/supplier |
| GET | `/sales-invoices` | `finance.read` | filter status/due date/customer |
| GET | `/customer-payments` | `finance.read` | filter date/customer |
| GET | `/dashboard/summary` | `dashboard.read` | scoped KPI aggregation |
| GET | `/reports/purchases` | `reports.read` | period/supplier/status |
| GET | `/reports/sales` | `reports.read` | period/customer/status |
| GET | `/reports/inventory` | `reports.read` | balance and movement summary |
| GET | `/reports/receivables` | `reports.read` | aging buckets |
| GET | `/reports/payables` | `reports.read` | aging buckets |
| GET | `/audit-logs` | `audit.read` | entity/actor/action/date filters |

## Command body examples

Approve/reject:

```json
{ "comment": "Approved within budget" }
```

```json
{ "reason": "Supplier quotation is missing" }
```

Payment creation:

```json
{
  "party_id": "018f...",
  "payment_date": "2026-09-21",
  "method": "BANK_TRANSFER",
  "reference": "TRX-8891",
  "currency": "IDR",
  "allocations": [
    { "invoice_id": "018f...", "amount": "1250000.00" }
  ]
}
```

## Standard domain error codes

`validation_error`, `not_found`, `forbidden`, `state_conflict`, `version_conflict`, `duplicate_document`, `idempotency_conflict`, `insufficient_stock`, `quantity_exceeds_remaining`, `match_exception`, `payment_exceeds_outstanding`, `dependent_payment_exists`, `downstream_document_exists`, `separation_of_duties_violation`, `rate_limit_exceeded`, `csrf_invalid`, and `internal_error`.

Every approval endpoint returns `403 separation_of_duties_violation` when the authenticated actor is the document creator. Every stock-posting endpoint revalidates source quantities, stock version, duplicate source movement, and sufficient on-hand quantity in the same transaction before writing the ledger.
