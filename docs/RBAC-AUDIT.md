# RBAC audit (2026-10-20)

## Implemented role matrix

| Capability | Staff | Supervisor | Super Admin |
| --- | --- | --- | --- |
| Read operational documents, dashboard, master data | Yes | Yes | Yes |
| Create and process PR, PO, GR, SO, delivery, stock adjustment | Yes | No | Yes |
| Approve or reject PR, SO, stock adjustment | No | Yes | Yes |
| Reverse GR, delivery, stock adjustment | No | Yes | Yes |
| Create AP/AR invoice and payment drafts | Yes | No | Yes |
| Match/post/void AP and AR transactions | No | Yes | Yes |
| Maintain master data and detailed reports | No | Yes | Yes |
| Invite/manage users; organization settings | No | No | Yes |

Supervisor approval and rejection require a different user from the document creator. An active Super Admin may self-approve as an explicit bypass; status, version, and audit checks still apply. UI visibility is only a convenience; HTTP authorization is decisive.

Migration 000010 maps `OPERATOR` to `STAFF`, promotes the oldest active `ADMIN` to `SUPER_ADMIN` to retain an account owner, and maps other `ADMIN` users to `SUPERVISOR`. If there was no active administrator, it selects the oldest administrator. Fresh installations bootstrap a Super Admin. Verify the selected owner before applying this migration to production data.

## Remaining audit findings

1. **P1 — AP/O2C maker-checker implemented, three-account browser flow pending.** Staff or Super Admin creates supplier/customer invoice and payment drafts. Supervisor matches/posts, and the database rejects self-review except for an active Super Admin. Migration `000012`, isolated database integration tests, and live read-only Finance browser smoke passed. A live Supervisor acceptance flow still needs an account.
2. **P1 — Role assignment cannot be changed in the UI.** Super Admin can invite users into one of the three roles, but there is no role-change endpoint. Existing Supervisors must be replaced with a new invitation if their role should change. A future endpoint should revoke sessions and audit each role change.
3. **P2 — Operational ownership is uneven.** PR/SO/adjustment approval checks creator identity in the database. PO issue, GR post, and AP match/post have maker-checker guards; submit and some cancel flows do not consistently restrict the document owner. Decide whether Staff may process another Staff member's draft.
4. **P2 — Three-account acceptance pending.** The old database was backed up and migrated from 9 to 12; role counts and three existing PR statuses were preserved. Updated Docker images pass health and read-only browser smoke for Super Admin and Staff. There is no Supervisor account yet in the main database, so the normal three-account approval flow still needs an invitation and acceptance test.

Before calling the three-role workflow accepted: invite and activate a Supervisor, then exercise one PR, SO, adjustment, AP, and AR with three accounts. Verify Staff gets HTTP 403 for approval/post actions, Supervisor gets HTTP 403 for creation and user management, self-approval gets 409, a different reviewer can approve, and an active Super Admin can self-approve with an audit entry. The migration and Docker rebuild are already complete.
