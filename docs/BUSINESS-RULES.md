# Actors, Domains, and Business Rules

## 1. Actors

| Actor | Participant type | Responsibilities | Explicit limits |
|---|---|---|---|
| Admin | Internal human | Manage users and master data; approve/reject transactions; record invoices and payments; access reports | Cannot approve/reject a document they created; cannot alter posted ledger entries |
| Operator | Internal human | Create operational drafts; submit transactions; receive goods; create sales and delivery documents; perform stock count | Cannot approve, post payment, manage users, or edit posted documents |
| User | Internal human | Activate an invited account and set a new password from a valid one-time link | Cannot create or distribute invitation/reset links unless also an Admin |
| System | Internal automation | Validate rules; generate document numbers; post ledgers; calculate totals; write audit trail; close linked documents | Cannot invent business approval; only acts after a valid user command/event |
| Supplier | External participant | Receive purchase order; deliver goods; issue supplier invoice | No direct application access in MVP |
| Customer | External participant | Place order; receive confirmation/goods/invoice; make payment | No direct application access in MVP |

The implemented MVP uses two bundled roles, `ADMIN` and `OPERATOR`. Route middleware and handlers enforce authentication and the Admin-only commands; a separate persisted capability/permission model is not implemented.

## 2. Domain ownership

| Domain | Owns | Does not own |
|---|---|---|
| Auth | Users, credentials, opaque access/refresh sessions, roles, permissions, CSRF tokens | Business approvals |
| Master Data | Products, units, suppliers, customers, warehouses | Stock quantity and transactions |
| Purchasing | Purchase requests, purchase orders, goods receipts | Stock ledger and supplier payment ledger |
| Inventory | Stock movements, stock balance projection, stock adjustments | Product commercial data and sales orders |
| Sales | Sales orders and deliveries | Stock ledger and customer payment ledger |
| Finance | Supplier/customer invoices, payments, payable/receivable status | Full general ledger/accounting in MVP |
| Reporting | Read models and dashboard metrics | Mutation of business documents |
| Audit | Append-only critical operation history | Domain state decisions |

Modules are separated by Go packages and HTTP surfaces, but their PostgreSQL stores share one schema. Multi-entity transactions intentionally update related tables directly—for example receipt/delivery posting updates stock, and payment posting closes or reopens source documents. This is the implemented modular-monolith boundary; cross-module application ports are a future architecture option, not current evidence.

## 3. Global business rules

| ID | Rule |
|---|---|
| BR-GEN-001 | Every transactional document has immutable UUID, human-readable number, status, version, creator, timestamps, and optional notes. |
| BR-GEN-002 | A command is authorized before domain validation, and both occur before persistence. |
| BR-GEN-003 | Invalid state transition returns conflict and performs no partial write. |
| BR-GEN-004 | Posted stock/finance records are immutable; correction uses linked reversal. |
| BR-GEN-005 | Critical command writes actor, action, entity, entity ID, timestamp, previous state, new state, stable trace ID (BPMN task ID or supporting action ID), request ID, and a mandatory reason where applicable. Full JSON before/after snapshots are outside the current MVP baseline. |
| BR-GEN-006 | All totals are calculated server-side from quantity, unit price, tax, and discount inputs. Client totals are informational only. |
| BR-GEN-007 | Disabled users cannot authenticate or start new sessions; historical records retain their actor reference. |
| BR-GEN-008 | Inactive master data cannot be selected for a new document but remains visible on historical documents. |
| BR-GEN-009 | A transition uses optimistic locking; stale `version` is rejected. |
| BR-GEN-010 | A repeated command with the same idempotency key returns the original result and never posts twice. |
| BR-GEN-011 | Maker-checker is mandatory: `created_by` cannot equal `reviewed_by`; there is no self-approval override in MVP. |
| BR-GEN-012 | Before every inventory posting, the system revalidates source quantities, ledger idempotency, stock projection version, and—when quantity decreases—sufficient on-hand stock. |
| BR-GEN-013 | Organization settings default to currency `IDR`, timezone `Asia/Jakarta`, and one default warehouse. Maker-checker is a fixed business invariant, not a configurable setting. |
| BR-GEN-014 | Basic cancellation is allowed only in the explicitly listed pre-posting/pre-fulfillment states, requires a reason, and writes audit. Cancellation after stock or finance posting is not allowed; only the separately mapped correction reversal/void may be used. |
| BR-GEN-015 | A stock movement, payment allocation, void/reversal, state transition, balance/progress update, and its audit record commit atomically in one database transaction whenever they belong to the same command. |

## 4. Authorization rules

| Capability | Admin | Operator |
|---|---:|---:|
| Users: read/create/update/disable | ✓ | — |
| Master data: read | ✓ | ✓ |
| Master data: create/update/deactivate | ✓ | — |
| Transaction draft: create/edit | ✓ | ✓ |
| Transaction: submit | ✓ | ✓ |
| Transaction: approve/reject | ✓ | — |
| Goods receipt / delivery: create | ✓ | ✓ |
| Inventory adjustment: create/submit | ✓ | ✓ |
| Invoice/payment: create/post/void | ✓ | — |
| Dashboard operational metrics | ✓ | permitted metrics only |
| Reports and audit trail | ✓ | — |

Maker-checker is mandatory for the MVP. An Admin who created a Purchase Request, Sales Order, or Inventory Adjustment cannot approve or reject that same document. There is no role-based or organization-setting override.

## 5. Authentication rules

| ID | Rule |
|---|---|
| BR-AUTH-001 | Only Admin can create an invitation or password-reset request for another user. |
| BR-AUTH-002 | Invitation and password-reset tokens use at least 256 bits of cryptographically secure randomness, are stored only as hashes, are single-use, and expire after 24 hours. The raw link is returned once for manual secure sharing and is never persisted in the database, application log, access log, analytics event, or audit payload. |
| BR-AUTH-003 | External email, WhatsApp, or other notification delivery is out of scope; Admin explicitly marks a link as manually shared. |
| BR-AUTH-004 | Activating an invitation requires a valid unused token and a password that meets policy; success changes `INVITED → ACTIVE` and consumes the token atomically. |
| BR-AUTH-005 | Password reset requires a valid unused token for an ACTIVE or LOCKED user; success consumes the token, replaces the password hash, revokes all access/refresh sessions, and returns `LOCKED → ACTIVE` when applicable. |
| BR-AUTH-006 | Public token endpoints return a generic invalid-or-expired response and never reveal whether an account exists. |
| BR-AUTH-007 | Creating, sharing, consuming, expiring, or revoking an invitation/reset token produces an audit entry with its stable action/BPMN ID without storing the raw token. |
| BR-AUTH-008 | The first Admin is created only by a local bootstrap CLI with direct database access. It takes the password through an interactive prompt or stdin, runs under a database advisory lock, and refuses to run when any user already exists. There is no bootstrap HTTP endpoint or default credential. |
| BR-AUTH-009 | Passwords are hashed with Argon2id using a unique salt. Initial parameters are 64 MiB memory, 3 iterations, parallelism 1, 32-byte output, and must be benchmarked on the deployment target without going below the documented OWASP minimum. Password length is 12–128 characters and all characters/spaces are allowed. |
| BR-AUTH-010 | Browser authentication uses opaque access and refresh tokens only in `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`, no-`Domain`, `__Host-` cookies. Access expires after 15 minutes; refresh has 7-day idle and 30-day absolute expiry. Refresh rotates both tokens, and reuse of a rotated refresh token revokes its entire token family. |
| BR-AUTH-011 | Every unsafe cookie-authenticated request requires a per-session synchronizer token in `X-CSRF-Token`, exact trusted-origin validation, and JSON content type. CSRF tokens are returned in a response body, never in a URL or audit/log payload. |
| BR-AUTH-012 | Login, refresh, invitation activation, and password-reset endpoints are rate-limited and use generic failures. Logout revokes the current session; password reset, user deactivation, detected refresh-token reuse, or an Admin revoke-all action revokes the affected sessions immediately. |

## 6. Procure-to-Pay rules

| ID | Rule |
|---|---|
| BR-P2P-001 | Purchase request requires at least one active product, positive quantity, purpose, and requester. |
| BR-P2P-002 | Only `DRAFT` request can be edited or submitted. |
| BR-P2P-003 | Approval/rejection requires `SUBMITTED`; rejection reason is mandatory. |
| BR-P2P-004 | Only `APPROVED` request can be converted to one purchase order; conversion preserves line-level reference. |
| BR-P2P-005 | Purchase order supplier must be active; order lines cannot exceed approved request quantities without re-approval. |
| BR-P2P-006 | Goods receipt quantity per PO line cannot exceed remaining ordered quantity. Partial receipts are allowed. |
| BR-P2P-007 | Posting goods receipt atomically writes receipt status, positive stock movement, stock projection, linked PO progress, and audit trail. |
| BR-P2P-008 | One non-VOID consolidated supplier invoice is allowed per PO after full receipt. It is postable only when supplier, PO, received quantity, price, and total pass three-way match with zero quantity and price tolerance; a replacement may be created only after the prior invoice is VOID and must reference it. |
| BR-P2P-009 | Invoice mismatch requires Admin resolution; no payment can be posted while invoice is unposted. |
| BR-P2P-010 | Payment amount is positive and cannot exceed invoice outstanding amount. Partial payments are allowed. |
| BR-P2P-011 | PO closes only when all quantities are received and all linked posted supplier invoices are fully paid. PR becomes `COMPLETED` when its PO closes. |
| BR-P2P-012 | A rejected Purchase Request is terminal. Copy creates a new DRAFT with a new ID/number and `copied_from_id`; it never reopens or mutates the rejected source. |

## 7. Order-to-Cash rules

| ID | Rule |
|---|---|
| BR-O2C-001 | Sales order requires active customer, at least one active product, positive quantity, and valid price. |
| BR-O2C-002 | Only `DRAFT` order can be edited or submitted. |
| BR-O2C-003 | Submission checks on-hand stock. MVP does not reserve stock; approval must recheck stock to prevent overselling. |
| BR-O2C-004 | Approval/rejection requires `SUBMITTED`; rejection reason is mandatory. |
| BR-O2C-005 | Delivery quantity cannot exceed approved undelivered quantity or current available stock. Partial delivery is allowed. |
| BR-O2C-006 | Posting delivery revalidates current stock, then atomically writes delivery status, negative stock movement, stock projection, order progress, and audit trail. Negative on-hand stock is prohibited. |
| BR-O2C-007 | One non-VOID consolidated sales invoice is generated after all approved quantities are delivered and bills delivered quantity only; a replacement may be generated only after the prior invoice is VOID and must reference it. |
| BR-O2C-008 | Customer payment cannot exceed invoice outstanding amount. Partial payments are allowed. |
| BR-O2C-009 | Sales order completes when all approved quantities are delivered and all related invoices are fully paid. |
| BR-O2C-010 | A rejected Sales Order is terminal. Copy creates a new DRAFT with a new ID/number and `copied_from_id`; it never reopens or mutates the rejected source. |

## 8. Inventory rules

| ID | Rule |
|---|---|
| BR-INV-001 | Every quantity change is represented by an immutable movement line with product, warehouse, signed quantity, reason, source type, and source ID. |
| BR-INV-002 | Allowed movement types are `GOODS_RECEIPT`, `SALES_DELIVERY`, `STOCK_ADJUSTMENT`, and paired `REVERSAL`. |
| BR-INV-003 | Adjustment requires counted quantity, system quantity snapshot, difference, reason, and warehouse. |
| BR-INV-004 | Adjustment is submitted and approved against the captured snapshot; if balance/version changed, approval fails and recount is required. |
| BR-INV-005 | Zero-difference count completes without movement but retains audit evidence. |
| BR-INV-006 | Negative adjustment cannot make on-hand quantity negative. |
| BR-INV-007 | `stock_balances` is a rebuildable projection, never the only stock record. |
| BR-INV-008 | Stock-document reversal requires Admin and a reason, creates an immutable paired inverse movement, revalidates that the result cannot make stock negative, and commits source state/balance/audit atomically. Receipt/delivery reversal is blocked while a downstream non-VOID invoice exists. |

## 9. Finance rules

| ID | Rule |
|---|---|
| BR-FIN-001 | Finance MVP tracks invoice, payment allocation, outstanding amount, due date, and payment status; it is not double-entry general accounting. |
| BR-FIN-002 | Posted invoice/payment is immutable. Admin may void it without a second approver only by supplying a mandatory reason. A payment void reverses its allocations, restores invoice outstanding/status, and reopens automatically closed source documents atomically when they are no longer fully paid. An invoice with any active allocation cannot be voided until those payments are voided; invoice void then records a reversal reference and audit trail. |
| BR-FIN-003 | Payment date cannot precede invoice date; currency must match invoice currency. |
| BR-FIN-004 | Payment allocation sum must equal payment amount. |
| BR-FIN-005 | Dashboard overdue values use posted invoice due date and outstanding amount. |

## 10. Explicit MVP exclusions

Purchase returns, sales returns/refunds, warehouse transfers, tax engine, stock reservation, administrative short-close, advanced cancellation, general ledger/journal accounting, and external notification integrations are out of scope. Rejected Purchase Requests and Sales Orders are terminal; users may copy them into a new `DRAFT` document.
