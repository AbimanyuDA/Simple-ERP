# Katalog BPMN Simple ERP

Buka [README](README.md) dahulu. Struktur khusus P2P ada di [P2P-REVIEW](P2P-REVIEW.md). SVG adalah preview; `.bpmn` adalah model BPMN 2.0 dengan DI.

## Level 0

| Diagram | BPMN | Preview |
| --- | --- | --- |
| Simple ERP - landscape keseluruhan | [l0-simple-erp](level-0/l0-simple-erp.bpmn) | [SVG](preview/l0-simple-erp.svg) |

## Level 1

| Diagram | BPMN | Preview |
| --- | --- | --- |
| Akses dan pengguna | [l1-access](level-1/l1-access.bpmn) | [SVG](preview/l1-access.svg) |
| Procure to Pay - permintaan sampai pelunasan | [l1-p2p](level-1/l1-p2p.bpmn) | [SVG](preview/l1-p2p.svg) |
| Order to Cash - pesanan sampai pelunasan | [l1-o2c](level-1/l1-o2c.bpmn) | [SVG](preview/l1-o2c.svg) |
| Pengendalian inventory | [l1-inventory](level-1/l1-inventory.bpmn) | [SVG](preview/l1-inventory.svg) |
| Finance AP dan AR | [l1-finance](level-1/l1-finance.bpmn) | [SVG](preview/l1-finance.svg) |
| Master data dan konfigurasi | [l1-administration](level-1/l1-administration.bpmn) | [SVG](preview/l1-administration.svg) |
| Monitoring, laporan dan audit | [l1-reporting](level-1/l1-reporting.bpmn) | [SVG](preview/l1-reporting.svg) |
| Perspektif role Staff | [role-staff](level-1/role-staff.bpmn) | [SVG](preview/role-staff.svg) |
| Perspektif role Supervisor | [role-supervisor](level-1/role-supervisor.bpmn) | [SVG](preview/role-supervisor.svg) |
| Perspektif role Super Admin | [role-super-admin](level-1/role-super-admin.bpmn) | [SVG](preview/role-super-admin.svg) |

## Level 2

| Diagram | BPMN | Preview |
| --- | --- | --- |
| Undangan dan pengelolaan pengguna | [l2-access](level-2/l2-access.bpmn) | [SVG](preview/l2-access.svg) |
| Login, sesi dan pemulihan password | [l2-session](level-2/l2-session.bpmn) | [SVG](preview/l2-session.svg) |
| Pemeliharaan master dan konfigurasi | [l2-master-settings](level-2/l2-master-settings.bpmn) | [SVG](preview/l2-master-settings.svg) |
| Purchase Request dari draft sampai review | [l2-pr](level-2-p2p/l2-pr.bpmn) | [SVG](preview/l2-pr.svg) |
| Purchase Order dan penerimaan barang | [l2-po-gr](level-2-p2p/l2-po-gr.bpmn) | [SVG](preview/l2-po-gr.svg) |
| Sales Order dari draft sampai konfirmasi | [l2-so](level-2/l2-so.bpmn) | [SVG](preview/l2-so.svg) |
| Pengiriman barang dan pencatatan stok keluar | [l2-delivery](level-2/l2-delivery.bpmn) | [SVG](preview/l2-delivery.svg) |
| Stock adjustment dengan review terpisah | [l2-adjustment](level-2/l2-adjustment.bpmn) | [SVG](preview/l2-adjustment.svg) |
| Invoice supplier sampai pelunasan | [l2-ap](level-2-p2p/l2-ap.bpmn) | [SVG](preview/l2-ap.svg) |
| Invoice penjualan sampai penerimaan pembayaran | [l2-ar](level-2/l2-ar.bpmn) | [SVG](preview/l2-ar.svg) |
| Pembatalan dan koreksi Purchasing | [l2-purchase-correction](level-2-p2p/l2-purchase-correction.bpmn) | [SVG](preview/l2-purchase-correction.svg) |
| Revisi, pembatalan dan koreksi Sales | [l2-sales-correction](level-2/l2-sales-correction.bpmn) | [SVG](preview/l2-sales-correction.svg) |
| Koreksi adjustment dan pembacaan stok | [l2-inventory-correction](level-2/l2-inventory-correction.bpmn) | [SVG](preview/l2-inventory-correction.svg) |
| Koreksi invoice dan payment | [l2-finance-correction](level-2/l2-finance-correction.bpmn) | [SVG](preview/l2-finance-correction.svg) |
| Koreksi khusus Procure to Pay | [l2-p2p-correction](level-2-p2p/l2-p2p-correction.bpmn) | [SVG](preview/l2-p2p-correction.svg) |
| Dashboard, daftar, laporan dan audit | [l2-reporting](level-2/l2-reporting.bpmn) | [SVG](preview/l2-reporting.svg) |
| Kontrol request dan health | [l2-platform](level-2/l2-platform.bpmn) | [SVG](preview/l2-platform.svg) |

## Level 3

| Diagram | BPMN | Preview |
| --- | --- | --- |
| Bootstrap Super Admin pertama | [l3-bootstrap](level-3/l3-bootstrap.bpmn) | [SVG](preview/l3-bootstrap.svg) |
| Login dan penerbitan sesi | [l3-login](level-3/l3-login.bpmn) | [SVG](preview/l3-login.svg) |
| Rotasi refresh token | [l3-refresh](level-3/l3-refresh.bpmn) | [SVG](preview/l3-refresh.svg) |
| Logout sesi saat ini | [l3-logout](level-3/l3-logout.bpmn) | [SVG](preview/l3-logout.svg) |
| Undang pengguna dan tetapkan role | [l3-invite](level-3/l3-invite.bpmn) | [SVG](preview/l3-invite.svg) |
| Tandai token telah dibagikan | [l3-token-share](level-3/l3-token-share.bpmn) | [SVG](preview/l3-token-share.svg) |
| Cabut token undangan atau reset | [l3-token-revoke](level-3/l3-token-revoke.bpmn) | [SVG](preview/l3-token-revoke.svg) |
| Aktivasi pengguna dari undangan | [l3-activate-invitation](level-3/l3-activate-invitation.bpmn) | [SVG](preview/l3-activate-invitation.svg) |
| Terbitkan token reset password | [l3-reset-request](level-3/l3-reset-request.bpmn) | [SVG](preview/l3-reset-request.svg) |
| Terapkan password baru | [l3-reset-consume](level-3/l3-reset-consume.bpmn) | [SVG](preview/l3-reset-consume.svg) |
| Aktivasi administratif dan deaktivasi pengguna | [l3-user-status](level-3/l3-user-status.bpmn) | [SVG](preview/l3-user-status.svg) |
| Cabut seluruh sesi pengguna | [l3-session-revoke](level-3/l3-session-revoke.bpmn) | [SVG](preview/l3-session-revoke.svg) |
| Buat master data | [l3-master-create](level-3/l3-master-create.bpmn) | [SVG](preview/l3-master-create.svg) |
| Aktif/nonaktifkan master data | [l3-master-status](level-3/l3-master-status.bpmn) | [SVG](preview/l3-master-status.svg) |
| Ubah pengaturan organisasi | [l3-settings](level-3/l3-settings.bpmn) | [SVG](preview/l3-settings.svg) |
| Buat Purchase Request | [l3-pr-create](level-3-p2p/l3-pr-create.bpmn) | [SVG](preview/l3-pr-create.svg) |
| Submit Purchase Request | [l3-pr-submit](level-3-p2p/l3-pr-submit.bpmn) | [SVG](preview/l3-pr-submit.svg) |
| Review Purchase Request | [l3-pr-review](level-3-p2p/l3-pr-review.bpmn) | [SVG](preview/l3-pr-review.svg) |
| Konversi PR menjadi Purchase Order | [l3-po-convert](level-3-p2p/l3-po-convert.bpmn) | [SVG](preview/l3-po-convert.svg) |
| Terbitkan Purchase Order | [l3-po-issue](level-3-p2p/l3-po-issue.bpmn) | [SVG](preview/l3-po-issue.svg) |
| Catat penerimaan barang | [l3-gr-create](level-3-p2p/l3-gr-create.bpmn) | [SVG](preview/l3-gr-create.svg) |
| Posting penerimaan dan stok masuk | [l3-gr-post](level-3-p2p/l3-gr-post.bpmn) | [SVG](preview/l3-gr-post.svg) |
| Batalkan PR draft | [l3-pr-cancel](level-3-p2p/l3-pr-cancel.bpmn) | [SVG](preview/l3-pr-cancel.svg) |
| Salin PR ditolak menjadi draft baru | [l3-pr-copy](level-3-p2p/l3-pr-copy.bpmn) | [SVG](preview/l3-pr-copy.svg) |
| Batalkan Purchase Order | [l3-po-cancel](level-3-p2p/l3-po-cancel.bpmn) | [SVG](preview/l3-po-cancel.svg) |
| Batalkan Goods Receipt draft | [l3-gr-cancel](level-3-p2p/l3-gr-cancel.bpmn) | [SVG](preview/l3-gr-cancel.svg) |
| Reverse Goods Receipt yang sudah posted | [l3-gr-reverse](level-3-p2p/l3-gr-reverse.bpmn) | [SVG](preview/l3-gr-reverse.svg) |
| Buat Sales Order | [l3-so-create](level-3/l3-so-create.bpmn) | [SVG](preview/l3-so-create.svg) |
| Revisi line Sales Order draft | [l3-so-edit](level-3/l3-so-edit.bpmn) | [SVG](preview/l3-so-edit.svg) |
| Periksa ketersediaan stok SO | [l3-so-stock](level-3/l3-so-stock.bpmn) | [SVG](preview/l3-so-stock.svg) |
| Submit Sales Order | [l3-so-submit](level-3/l3-so-submit.bpmn) | [SVG](preview/l3-so-submit.svg) |
| Review Sales Order | [l3-so-review](level-3/l3-so-review.bpmn) | [SVG](preview/l3-so-review.svg) |
| Konfirmasi Sales Order approved | [l3-so-confirm](level-3/l3-so-confirm.bpmn) | [SVG](preview/l3-so-confirm.svg) |
| Buat Delivery | [l3-delivery-create](level-3/l3-delivery-create.bpmn) | [SVG](preview/l3-delivery-create.svg) |
| Posting delivery dan stok keluar | [l3-delivery-post](level-3/l3-delivery-post.bpmn) | [SVG](preview/l3-delivery-post.svg) |
| Batalkan Sales Order | [l3-so-cancel](level-3/l3-so-cancel.bpmn) | [SVG](preview/l3-so-cancel.svg) |
| Salin SO ditolak menjadi draft baru | [l3-so-copy](level-3/l3-so-copy.bpmn) | [SVG](preview/l3-so-copy.svg) |
| Batalkan delivery draft | [l3-delivery-cancel](level-3/l3-delivery-cancel.bpmn) | [SVG](preview/l3-delivery-cancel.svg) |
| Reverse delivery posted | [l3-delivery-reverse](level-3/l3-delivery-reverse.bpmn) | [SVG](preview/l3-delivery-reverse.svg) |
| Catat stock opname / adjustment | [l3-adj-create](level-3/l3-adj-create.bpmn) | [SVG](preview/l3-adj-create.svg) |
| Submit stock adjustment | [l3-adj-submit](level-3/l3-adj-submit.bpmn) | [SVG](preview/l3-adj-submit.svg) |
| Review stock adjustment | [l3-adj-review](level-3/l3-adj-review.bpmn) | [SVG](preview/l3-adj-review.svg) |
| Posting selisih stock adjustment | [l3-adj-post](level-3/l3-adj-post.bpmn) | [SVG](preview/l3-adj-post.svg) |
| Batalkan adjustment draft atau approved | [l3-adj-cancel](level-3/l3-adj-cancel.bpmn) | [SVG](preview/l3-adj-cancel.svg) |
| Reverse adjustment selesai | [l3-adj-reverse](level-3/l3-adj-reverse.bpmn) | [SVG](preview/l3-adj-reverse.svg) |
| Catat invoice supplier | [l3-ap-invoice](level-3-p2p/l3-ap-invoice.bpmn) | [SVG](preview/l3-ap-invoice.svg) |
| Match invoice supplier | [l3-ap-match](level-3-p2p/l3-ap-match.bpmn) | [SVG](preview/l3-ap-match.svg) |
| Perbaiki mismatch invoice supplier | [l3-ap-resolve](level-3-p2p/l3-ap-resolve.bpmn) | [SVG](preview/l3-ap-resolve.svg) |
| Buat draft invoice penjualan | [l3-ar-invoice](level-3/l3-ar-invoice.bpmn) | [SVG](preview/l3-ar-invoice.svg) |
| Post invoice penjualan | [l3-ar-invoice-post](level-3/l3-ar-invoice-post.bpmn) | [SVG](preview/l3-ar-invoice-post.svg) |
| Catat pembayaran supplier | [l3-ap-payment](level-3-p2p/l3-ap-payment.bpmn) | [SVG](preview/l3-ap-payment.svg) |
| Posting pembayaran supplier | [l3-ap-payment-post](level-3-p2p/l3-ap-payment-post.bpmn) | [SVG](preview/l3-ap-payment-post.svg) |
| Void invoice supplier | [l3-ap-invoice-void](level-3-p2p/l3-ap-invoice-void.bpmn) | [SVG](preview/l3-ap-invoice-void.svg) |
| Void pembayaran supplier | [l3-ap-payment-void](level-3-p2p/l3-ap-payment-void.bpmn) | [SVG](preview/l3-ap-payment-void.svg) |
| Catat pembayaran customer | [l3-ar-payment](level-3/l3-ar-payment.bpmn) | [SVG](preview/l3-ar-payment.svg) |
| Posting pembayaran customer | [l3-ar-payment-post](level-3/l3-ar-payment-post.bpmn) | [SVG](preview/l3-ar-payment-post.svg) |
| Void invoice customer | [l3-ar-invoice-void](level-3/l3-ar-invoice-void.bpmn) | [SVG](preview/l3-ar-invoice-void.svg) |
| Void pembayaran customer | [l3-ar-payment-void](level-3/l3-ar-payment-void.bpmn) | [SVG](preview/l3-ar-payment-void.svg) |
| Baca dokumen dan saldo operasional | [l3-operational-read](level-3/l3-operational-read.bpmn) | [SVG](preview/l3-operational-read.svg) |
| Baca pengguna dan pengaturan | [l3-admin-read](level-3/l3-admin-read.bpmn) | [SVG](preview/l3-admin-read.svg) |
| Baca invoice dan payment | [l3-finance-read](level-3/l3-finance-read.bpmn) | [SVG](preview/l3-finance-read.svg) |
| Baca ringkasan Dashboard | [l3-dashboard](level-3/l3-dashboard.bpmn) | [SVG](preview/l3-dashboard.svg) |
| Baca laporan dan audit trail | [l3-report](level-3/l3-report.bpmn) | [SVG](preview/l3-report.svg) |
| Kontrol request, otorisasi dan idempotency | [l3-kontrol-request](level-3/l3-kontrol-request.bpmn) | [SVG](preview/l3-kontrol-request.svg) |
| Pemeriksaan liveness aplikasi | [l3-health](level-3/l3-health.bpmn) | [SVG](preview/l3-health.svg) |
