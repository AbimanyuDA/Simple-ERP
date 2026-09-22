# BPMN-to-Software Traceability Matrix

This document is Level 3 traceability. It is deliberately not a BPMN diagram.

Phase 2 concrete implementation paths and verification evidence are recorded per gate in `docs/checkpoints/`. Checkpoints 1 and 2 are documented in `CHECKPOINT-1.md` and `CHECKPOINT-2.md`; stable IDs remain unchanged and appear in application metadata, audit writes, automated-test names, and OpenAPI operation IDs.

## Stable ID convention

Format for executable business tasks:

```text
<DOMAIN>-<DOCUMENT>-<SEQUENCE>
```

- Domain: `PUR`, `SAL`, `INV`, `FIN`, `AUTH`, `MDM`, `RPT`, `AUD`.
- Document: `PR`, `PO`, `GR`, `SO`, `DLV`, `INV`, `PAY`, `ADJ`, or a clear three-letter abbreviation.
- Sequence: three digits, allocated monotonically within domain/document.
- IDs never change after release, even when the label changes. Removed IDs are retired and never reused.
- Gateways use `GW-<PROCESS>-<MEANING>`; flows use `FLOW-<PROCESS>-<SEQUENCE>`; message flows use `MSG-<PROCESS>-<SEQUENCE>`.
- UI/API/use case/test metadata must carry the BPMN task ID. Audit entries persist it as `trace_id` with `trace_kind=BPMN_TASK`.
- Supporting lifecycle/recovery commands that are sourced from a Level 0 capability use `ACT-<DOMAIN>-<RESOURCE>-<ACTION>`. They are not BPMN task IDs and must not be inserted into Level 2 merely to document software recovery behavior. Use-case metadata, audit `trace_id`, API contract, and tests carry the action ID.

Requirement IDs use `BR-*` from `BUSINESS-RULES.md`. Automated tests use `T-<TRACE-ID>-<NN>`, where the trace ID is a stable BPMN task ID or supporting `ACT-*` ID.

## User Invitation and Password Reset

| BPMN ID | Requirement | Actor | UI | API | Application use case | Database operation | From → To | Automated test |
|---|---|---|---|---|---|---|---|---|
| AUTH-USR-001 | BR-AUTH-001/002 | Admin | `/users/invite` | POST `/user-invitations` | `CreateUserInvitation` | insert INVITED user, invitation request, audit | `User [*] → INVITED`; `Invitation [*] → CREATED` | T-AUTH-USR-001-01 |
| AUTH-USR-002 | BR-AUTH-002/007 | System | same invitation submission | internal step of POST `/user-invitations` | `GenerateInvitationToken` | generate raw token; persist hash/expiry only; audit | `CREATED → CREATED` | T-AUTH-USR-002-01 |
| AUTH-USR-003 | BR-AUTH-003/007 | Admin | invitation detail, Copy and Mark Shared | POST `/user-invitations/{id}/mark-shared` | `MarkInvitationShared` | set `shared_at`, status, audit | `CREATED → SHARED` | T-AUTH-USR-003-01 |
| AUTH-USR-004 | BR-AUTH-004/006 | User | `/activate-account` | POST `/auth/activate-invitation` | `SubmitInvitationActivation` | validate token/password input; no partial write | `SHARED → SHARED` | T-AUTH-USR-004-01 |
| AUTH-USR-005 | BR-AUTH-004/007 | System | activation success | internal step of POST `/auth/activate-invitation` | `ActivateInvitedUser` | consume token; write password hash; activate user; audit | `Invitation SHARED → CONSUMED`; `User INVITED → ACTIVE` | T-AUTH-USR-005-01/02 |
| AUTH-PWD-001 | BR-AUTH-001/002 | Admin | user detail, Reset Password | POST `/users/{id}/password-resets` | `CreatePasswordReset` | insert reset request and audit | `Reset [*] → CREATED` | T-AUTH-PWD-001-01 |
| AUTH-PWD-002 | BR-AUTH-002/007 | System | same reset submission | internal step of POST `/users/{id}/password-resets` | `GeneratePasswordResetToken` | generate raw token; persist hash/expiry only; audit | `CREATED → CREATED` | T-AUTH-PWD-002-01 |
| AUTH-PWD-003 | BR-AUTH-003/007 | Admin | reset detail, Copy and Mark Shared | POST `/password-resets/{id}/mark-shared` | `MarkPasswordResetShared` | set `shared_at`, status, audit | `CREATED → SHARED` | T-AUTH-PWD-003-01 |
| AUTH-PWD-004 | BR-AUTH-005/006 | User | `/reset-password` | POST `/auth/reset-password` | `SubmitNewPassword` | validate token/password input; no partial write | `SHARED → SHARED` | T-AUTH-PWD-004-01 |
| AUTH-PWD-005 | BR-AUTH-005/007 | System | reset success | internal step of POST `/auth/reset-password` | `ConsumePasswordReset` | consume token; update password hash; revoke all access/refresh sessions; unlock user; audit | `Reset SHARED → CONSUMED`; `LOCKED → ACTIVE` if applicable | T-AUTH-PWD-005-01/02 |

## Procure-to-Pay

| BPMN ID | Requirement | Actor | UI | API | Application use case | Database operation | From → To | Automated test |
|---|---|---|---|---|---|---|---|---|
| PUR-PR-001 | BR-P2P-001/002 | Operator | `/purchasing/requests/new`, edit page | POST/PATCH `/purchase-requests` | `CreatePurchaseRequest`, `UpdatePurchaseRequest` | insert/update PR + lines + audit | `[*] → DRAFT` / `DRAFT → DRAFT` | T-PUR-PR-001-01 |
| PUR-PR-002 | BR-P2P-002 | Operator | request detail, Submit | POST `/{id}/submit` | `SubmitPurchaseRequest` | guarded PR update + audit | `DRAFT → SUBMITTED` | T-PUR-PR-002-01 |
| PUR-PR-003 | BR-P2P-003, BR-GEN-011 | Admin | approval inbox/detail | POST `/{id}/approve` or `/reject` | `ReviewPurchaseRequest` | guarded PR update, maker-checker, reviewer/reason + audit | `SUBMITTED → APPROVED|REJECTED` | T-PUR-PR-003-01/02/03 |
| PUR-PO-001 | BR-P2P-004/005 | Operator | approved request, Convert | POST `/{id}/purchase-order`; PATCH PO | `ConvertPurchaseRequestToOrder` | insert PO + lines; update PR; audit | `PR APPROVED → CONVERTED_TO_PO`, `PO [*] → DRAFT` | T-PUR-PO-001-01 |
| PUR-PO-002 | BR-P2P-005 | System | PO detail, Issue | POST `/purchase-orders/{id}/issue` | `IssuePurchaseOrder` | guarded PO update + audit | `DRAFT → ISSUED` | T-PUR-PO-002-01 |
| PUR-GR-001 | BR-P2P-006 | Operator | PO Receive Goods form | POST `/purchase-orders/{id}/goods-receipts` | `CreateGoodsReceipt` | insert GR + lines + audit | `GR [*] → DRAFT` | T-PUR-GR-001-01 |
| PUR-GR-002 | BR-P2P-006/007, BR-GEN-012 | System | receipt confirmation | POST `/goods-receipts/{id}/post` | `PostGoodsReceipt` | revalidate source/version; guarded GR/PO update; positive movement/balance; audit | `GR DRAFT → POSTED`; `PO ISSUED|PARTIALLY_RECEIVED → PARTIALLY_RECEIVED|RECEIVED` | T-PUR-GR-002-01/02/03 |
| PUR-INV-001 | BR-P2P-008 | Admin | Finance supplier invoice form | POST `/supplier-invoices` | `RecordSupplierInvoice` | insert invoice + lines + audit | `[*] → MATCH_PENDING` | T-PUR-INV-001-01 |
| PUR-INV-002 | BR-P2P-008/009 | System | invoice match result | POST `/{id}/match`, internal post | `MatchSupplierInvoice` | read PO/GR; update match result/status; audit | `MATCH_PENDING → POSTED|MATCH_EXCEPTION` | T-PUR-INV-002-01/02 |
| PUR-INV-003 | BR-P2P-009 | Admin | match exception resolution | POST `/{id}/resolve-match` | `ResolveSupplierInvoiceMatch` | update allowed invoice fields/resolution + audit | `MATCH_EXCEPTION → MATCH_PENDING` | T-PUR-INV-003-01 |
| PUR-PAY-001 | BR-P2P-010, BR-FIN-003/004 | Admin | supplier payment form | POST `/supplier-payments` | `CreateSupplierPayment` | insert payment draft + allocations + audit | `[*] → DRAFT` | T-PUR-PAY-001-01 |
| PUR-PAY-002 | BR-P2P-010 | System | payment confirmation | POST `/supplier-payments/{id}/post` | `PostSupplierPayment` | post payment; apply allocations/outstanding; update invoice; audit | `Payment DRAFT → POSTED`; `Invoice POSTED|PARTIALLY_PAID → PARTIALLY_PAID|PAID` | T-PUR-PAY-002-01/02 |
| PUR-CLOSE-001 | BR-P2P-011 | System | PO completion status | internal follow-on of final supplier-payment posting | `ClosePurchaseDocuments` | guarded PO and PR updates + audit in payment transaction | `PO RECEIVED → CLOSED`; `PR CONVERTED_TO_PO → COMPLETED` | T-PUR-CLOSE-001-01 |

## Order-to-Cash

| BPMN ID | Requirement | Actor | UI | API | Application use case | Database operation | From → To | Automated test |
|---|---|---|---|---|---|---|---|---|
| SAL-SO-001 | BR-O2C-001/002 | Operator | `/sales/orders/new`, edit page | POST/PATCH `/sales-orders` | `CreateSalesOrder`, `UpdateSalesOrder` | insert/update SO + lines + audit | `[*] → DRAFT` / `DRAFT → DRAFT` | T-SAL-SO-001-01 |
| SAL-SO-003 | BR-O2C-003 | System | stock availability feedback | POST `/{id}/validate-stock` | `ValidateSalesOrderStock` | read balance/version; no stock mutation | `DRAFT → DRAFT` | T-SAL-SO-003-01/02 |
| SAL-SO-002 | BR-O2C-002/003 | Operator | order detail, Submit | POST `/{id}/submit` | `SubmitSalesOrder` | guarded SO update + audit | `DRAFT → SUBMITTED` | T-SAL-SO-002-01 |
| SAL-SO-004 | BR-O2C-003/004, BR-GEN-011 | Admin | approval inbox/detail | POST `/{id}/approve` or `/reject` | `ReviewSalesOrder` | stock recheck; maker-checker; guarded SO update + audit | `SUBMITTED → APPROVED|REJECTED` | T-SAL-SO-004-01/02/03 |
| SAL-SO-005 | BR-O2C-004 | System | order confirmation indicator | POST `/{id}/confirm` | `ConfirmSalesOrder` | write confirmation timestamp/audit | `APPROVED → APPROVED` | T-SAL-SO-005-01 |
| SAL-DLV-001 | BR-O2C-005 | Operator | SO Create Delivery form | POST `/sales-orders/{id}/deliveries` | `CreateDelivery` | insert delivery + lines + audit | `Delivery [*] → DRAFT` | T-SAL-DLV-001-01 |
| SAL-DLV-002 | BR-O2C-005/006, BR-GEN-012 | System | delivery confirmation | POST `/deliveries/{id}/post` | `PostDelivery` | revalidate stock/source/version; negative movement/balance; guarded delivery/SO update; audit | `Delivery DRAFT → POSTED`; `SO APPROVED|PARTIALLY_DELIVERED → PARTIALLY_DELIVERED|DELIVERED` | T-SAL-DLV-002-01/02/03 |
| SAL-INV-001 | BR-O2C-007 | System | sales order invoice detail | POST `/sales-orders/{id}/sales-invoice` | `GenerateSalesInvoice` | insert consolidated posted invoice + lines for delivered quantities; audit | `Invoice [*] → POSTED` | T-SAL-INV-001-01 |
| SAL-PAY-001 | BR-O2C-008, BR-FIN-003/004 | Admin | customer payment form | POST `/customer-payments` | `CreateCustomerPayment` | insert payment draft + allocations + audit | `[*] → DRAFT` | T-SAL-PAY-001-01 |
| SAL-PAY-002 | BR-O2C-008 | System | payment confirmation | POST `/customer-payments/{id}/post` | `PostCustomerPayment` | post payment; apply allocation/outstanding; update invoice; audit | `Payment DRAFT → POSTED`; `Invoice POSTED|PARTIALLY_PAID → PARTIALLY_PAID|PAID` | T-SAL-PAY-002-01/02 |
| SAL-SO-006 | BR-O2C-009 | System | SO completion status | internal follow-on of final customer-payment posting | `CloseSalesOrder` | guarded SO update + audit in payment transaction | `DELIVERED → COMPLETED` | T-SAL-SO-006-01 |

## Inventory Adjustment

| BPMN ID | Requirement | Actor | UI | API | Application use case | Database operation | From → To | Automated test |
|---|---|---|---|---|---|---|---|---|
| INV-ADJ-001 | BR-INV-003/004 | Operator | `/inventory/adjustments/new`, edit page | POST/PATCH `/inventory-adjustments` | `CreateInventoryAdjustment`, `UpdateInventoryAdjustment` | read balance snapshot; insert/update adjustment + lines + audit | `[*] → DRAFT` | T-INV-ADJ-001-01 |
| INV-ADJ-002 | BR-INV-003 | Operator | adjustment detail, Submit | POST `/{id}/submit` | `SubmitInventoryAdjustment` | guarded update + audit | `DRAFT → SUBMITTED` | T-INV-ADJ-002-01 |
| INV-ADJ-003 | BR-INV-004/006, BR-GEN-011 | Admin | adjustment approval | POST `/{id}/approve` or `/reject` | `ReviewInventoryAdjustment` | verify stock version and maker-checker; guarded update + audit | `SUBMITTED → APPROVED|REJECTED` | T-INV-ADJ-003-01/02/03 |
| INV-ADJ-004 | BR-INV-001/002/006/007, BR-GEN-012 | System | posting result | POST `/{id}/post` | `PostInventoryAdjustment` | revalidate stock/source/version; movement/balance; update adjustment; audit | `APPROVED → POSTED` | T-INV-ADJ-004-01/02/03 |
| INV-ADJ-005 | BR-INV-005 | System | completion status | same transaction as post/approve | `CompleteInventoryAdjustment` | guarded adjustment update + audit | `APPROVED|POSTED → COMPLETED` | T-INV-ADJ-005-01 |

## Supporting software capabilities and BPMN source

These are platform/supporting capabilities represented at Level 0 rather than pretending they are steps inside each operational workflow.

| BPMN source | Requirement / capability | UI | API area | Backend module | Test family |
|---|---|---|---|---|---|
| CAP-AUTH-001 | authentication, authorization, user lifecycle | Login, Users, Settings | `/auth/*`, `/users/*`, `/me` | `internal/auth` | T-CAP-AUTH-* |
| CAP-MDM-001 | Products, Suppliers, Customers, Warehouses, Units | Master Data | master endpoints | `internal/masterdata` | T-CAP-MDM-* |
| CAP-P2P-001 | detailed P2P source | Purchasing | purchase/receipt endpoints | `internal/purchasing` | T-PUR-* |
| CAP-O2C-001 | detailed O2C source | Sales | order/delivery endpoints | `internal/sales` | T-SAL-* |
| CAP-INV-001 | ledger and adjustment | Inventory | `/inventory/*` | `internal/inventory` | T-INV-* |
| CAP-FIN-001 | invoices, allocations, payments | Finance | finance endpoints | `internal/finance` | T-FIN-* |
| CAP-RPT-001 | dashboard and reports | Dashboard, reports | `/dashboard/*`, `/reports/*` | `internal/reporting` | T-CAP-RPT-* |
| CAP-AUD-001 | immutable audit lookup | Audit view | `/audit-logs` | `internal/audit` | T-CAP-AUD-* |

## Stable supporting lifecycle/action mappings

These actions are sourced from Level 0 capabilities because they are initialization, secondary lifecycle, or correction operations rather than steps in the Level 2 happy paths. Each row is normative before implementation.

| Action ID | BPMN source | Requirement | Actor/trigger | UI / API | Use case | State/result | Audit expectation | Test family |
|---|---|---|---|---|---|---|---|---|
| ACT-AUTH-ADMIN-BOOTSTRAP | CAP-AUTH-001 | BR-AUTH-008 | Local operator/System | CLI only; no HTTP/UI | `BootstrapFirstAdmin` | no users → first ACTIVE Admin | system actor, success/refusal, no password | T-ACT-AUTH-ADMIN-BOOTSTRAP-* |
| ACT-AUTH-INVITE-REVOKE | CAP-AUTH-001 | BR-AUTH-002/007 | Admin | POST `/user-invitations/{id}/revoke` | `RevokeInvitation` | CREATED/SHARED → REVOKED | actor, token record ID, prior/new state; no raw token | T-ACT-AUTH-INVITE-REVOKE-* |
| ACT-AUTH-INVITE-EXPIRE | CAP-AUTH-001 | BR-AUTH-002/007 | System on access | internal | `ExpireInvitation` | CREATED/SHARED → EXPIRED | system actor, token record ID, expiry time; no raw token | T-ACT-AUTH-INVITE-EXPIRE-* |
| ACT-AUTH-RESET-REVOKE | CAP-AUTH-001 | BR-AUTH-002/007 | Admin | POST `/password-resets/{id}/revoke` | `RevokePasswordReset` | CREATED/SHARED → REVOKED | actor, token record ID, prior/new state; no raw token | T-ACT-AUTH-RESET-REVOKE-* |
| ACT-AUTH-RESET-EXPIRE | CAP-AUTH-001 | BR-AUTH-002/007 | System on access | internal | `ExpirePasswordReset` | CREATED/SHARED → EXPIRED | system actor, token record ID, expiry time; no raw token | T-ACT-AUTH-RESET-EXPIRE-* |
| ACT-AUTH-SESSION-REVOKE | CAP-AUTH-001 | BR-AUTH-010/012 | Admin/System | POST `/users/{id}/revoke-sessions`; internal reset/deactivate/reuse | `RevokeUserSessions` | active sessions → REVOKED | initiator/reason/count; no session token | T-ACT-AUTH-SESSION-REVOKE-* |
| ACT-PUR-PR-CANCEL | CAP-P2P-001 | BR-GEN-014 | Creator/Admin | POST `/purchase-requests/{id}/cancel` | `CancelPurchaseRequest` | DRAFT → CANCELLED | actor, reason, prior/new state | T-ACT-PUR-PR-CANCEL-* |
| ACT-PUR-PR-COPY | CAP-P2P-001 | BR-P2P-012 | Admin/Operator | POST `/purchase-requests/{id}/copy` | `CopyRejectedPurchaseRequest` | REJECTED source unchanged; new DRAFT with `copied_from_id` | actor, source/new IDs | T-ACT-PUR-PR-COPY-* |
| ACT-PUR-PO-CANCEL | CAP-P2P-001 | BR-GEN-014 | Admin/Operator | POST `/purchase-orders/{id}/cancel` | `CancelPurchaseOrder` | DRAFT/ISSUED with no receipt → CANCELLED | actor, reason, prior/new state | T-ACT-PUR-PO-CANCEL-* |
| ACT-PUR-GR-CANCEL | CAP-P2P-001 | BR-GEN-014 | Admin/Operator | POST `/goods-receipts/{id}/cancel` | `CancelGoodsReceipt` | DRAFT → CANCELLED | actor, reason, prior/new state | T-ACT-PUR-GR-CANCEL-* |
| ACT-PUR-GR-REVERSE | CAP-P2P-001 | BR-GEN-004/015, BR-INV-008 | Admin | POST `/goods-receipts/{id}/reverse` | `ReverseGoodsReceipt` | POSTED → REVERSED plus inverse movement; block active invoice/negative result | actor, reason, source/reversal movement IDs, states | T-ACT-PUR-GR-REVERSE-* |
| ACT-SAL-SO-CANCEL | CAP-O2C-001 | BR-GEN-014 | Creator/Admin | POST `/sales-orders/{id}/cancel` | `CancelSalesOrder` | DRAFT/APPROVED with no delivery → CANCELLED | actor, reason, prior/new state | T-ACT-SAL-SO-CANCEL-* |
| ACT-SAL-SO-COPY | CAP-O2C-001 | BR-O2C-010 | Admin/Operator | POST `/sales-orders/{id}/copy` | `CopyRejectedSalesOrder` | REJECTED source unchanged; new DRAFT with `copied_from_id` | actor, source/new IDs | T-ACT-SAL-SO-COPY-* |
| ACT-SAL-DLV-CANCEL | CAP-O2C-001 | BR-GEN-014 | Admin/Operator | POST `/deliveries/{id}/cancel` | `CancelDelivery` | DRAFT → CANCELLED | actor, reason, prior/new state | T-ACT-SAL-DLV-CANCEL-* |
| ACT-SAL-DLV-REVERSE | CAP-O2C-001 | BR-GEN-004/015, BR-INV-008 | Admin | POST `/deliveries/{id}/reverse` | `ReverseDelivery` | POSTED → REVERSED plus inverse movement; block active invoice | actor, reason, source/reversal movement IDs, states | T-ACT-SAL-DLV-REVERSE-* |
| ACT-INV-ADJ-CANCEL | CAP-INV-001 | BR-GEN-014 | Creator/Admin | POST `/inventory-adjustments/{id}/cancel` | `CancelInventoryAdjustment` | DRAFT → CANCELLED | actor, reason, prior/new state | T-ACT-INV-ADJ-CANCEL-* |
| ACT-INV-ADJ-REVERSE | CAP-INV-001 | BR-GEN-004/015, BR-INV-008 | Admin | POST `/inventory-adjustments/{id}/reverse` | `ReverseInventoryAdjustment` | COMPLETED → REVERSED plus inverse movement; block negative result | actor, reason, source/reversal movement IDs, states | T-ACT-INV-ADJ-REVERSE-* |
| ACT-FIN-SINV-VOID | CAP-FIN-001 | BR-FIN-002 | Admin | POST `/supplier-invoices/{id}/void` | `VoidSupplierInvoice` | unallocated POSTED → VOID plus reversal | actor, mandatory reason, reversal ID | T-ACT-FIN-SINV-VOID-* |
| ACT-FIN-SPAY-VOID | CAP-FIN-001 | BR-FIN-002 | Admin | POST `/supplier-payments/{id}/void` | `VoidSupplierPayment` | POSTED → VOID; reverse allocation and reopen PR/PO if no longer paid | actor, mandatory reason, allocation/invoice/source-state deltas | T-ACT-FIN-SPAY-VOID-* |
| ACT-FIN-CINV-VOID | CAP-FIN-001 | BR-FIN-002 | Admin | POST `/sales-invoices/{id}/void` | `VoidSalesInvoice` | unallocated POSTED → VOID plus reversal | actor, mandatory reason, reversal ID | T-ACT-FIN-CINV-VOID-* |
| ACT-FIN-CPAY-VOID | CAP-FIN-001 | BR-FIN-002 | Admin | POST `/customer-payments/{id}/void` | `VoidCustomerPayment` | POSTED → VOID; reverse allocation and reopen SO if no longer paid | actor, mandatory reason, allocation/invoice/source-state deltas | T-ACT-FIN-CPAY-VOID-* |

## Coverage rules for future CI

CI should fail when:

- a Level 2 user/service task has no row in this file;
- a mapped API route or use case does not contain its BPMN ID metadata;
- a supporting lifecycle action lacks its stable action ID in use-case metadata, audit assertion, API contract, or test name;
- a test ID referenced here does not exist after its feature is implemented;
- a state transition in code is absent from both BPMN and `STATE-MACHINES.md`;
- an endpoint mutates stock/finance without an audit expectation.
