"""Reviewed as-is catalog. Labels describe current source, including known limitations."""


def populate(g):
    op, menu, life, Model = (g[k] for k in ("operation", "menu", "lifecycle", "Model"))
    M, R, A, U, S, D, P = (g[k] for k in ("MAKER", "REVIEWER", "OWNER", "USER", "SYSTEM", "DEPLOY", "PUBLIC"))
    auth = "backend/internal/auth/adapter/postgres/"
    pur = "backend/internal/purchasing/adapter/postgres/store.go"
    sal = "backend/internal/sales/adapter/postgres/store.go"
    inv = "backend/internal/inventory/adapter/postgres/store.go"
    fin = "backend/internal/finance/adapter/postgres/store.go"
    md = "backend/internal/masterdata/adapter/postgres/"
    rpt = "backend/internal/reporting/adapter/postgres/store.go"

    op("l3-bootstrap", "Bootstrap Super Admin pertama", D, "Jalankan bootstrap-admin melalui CLI",
       "Hash password; lock bootstrap; cek belum ada user",
       ["Simpan Super Admin ACTIVE", "Audit bootstrap; commit"],
       auth+"bootstrap.go", "CreateFirstAdmin", ["CLI bootstrap-admin --email --name"], "ACT-AUTH-BOOTSTRAP",
       "Hanya instalasi tanpa user. Ini pengelola deployment, bukan role Staff. Password dibaca dari stdin.")
    op("l3-login", "Login dan penerbitan sesi", U, "Masukkan email dan password",
       "Validasi JSON, limit percobaan, user ACTIVE dan password",
       ["Buat token acak; simpan hash sesi dan audit LOGIN", "Commit; kirim cookie akses, refresh, CSRF"],
       "backend/internal/auth/adapter/http/handler.go", "handleLogin", ["POST /auth/login"], "CAP-AUTH-001",
       "LoginService dan LoginStore memverifikasi password dan menyimpan sesi. Limit 5 kegagalan per email+IP per 15 menit dalam memori. Gagal: 400/401/415/429, bukan status user LOCKED.")
    op("l3-refresh", "Rotasi refresh token", U, "Kirim cookie refresh",
       "Cari hash; cek reuse, revoked dan expiry saat rotasi",
       ["Cabut sesi lama; buat sesi baru pada family sama", "Audit REFRESH_SESSION; commit; ganti cookie"],
       auth+"session.go", "ResolveRefresh / Rotate", ["POST /auth/refresh"], "CAP-AUTH-001",
       "Reuse yang terdeteksi oleh ResolveRefresh mencabut seluruh family dan menolak request. Rotate mewarisi deadline absolut family dan membatasi expiry akses/idle dengannya. Rotate tidak memeriksa status user secara langsung.",
       "401; cookie dibersihkan; reuse dapat mencabut family")
    op("l3-logout", "Logout sesi saat ini", U, "Pilih logout / panggil API",
       "RequireAuth + CSRF; cari sesi akses aktif",
       ["Cabut sesi yang memiliki access_hash", "Audit LOGOUT; commit; hapus cookie"],
       auth+"session.go", "Logout", ["POST /auth/logout"], "CAP-AUTH-001",
       "API dan tombol logout eksplisit tersedia di AppShell.")
    op("l3-invite", "Undang pengguna dan tetapkan role", A, "Isi nama, email, dan salah satu dari 3 role",
       "Role valid; email unik; hasil insert harus berhasil",
       ["Generate token 24 jam; simpan user INVITED dan hash token", "Audit INVITE_USER; commit; tampilkan raw token sekali"],
       auth+"lifecycle.go", "Invite", ["POST /user-invitations"], "AUTH-USR-001",
       "Role STAFF, SUPERVISOR, SUPER_ADMIN. Token digenerate sebelum store di LifecycleService. Tidak ada pengiriman email otomatis.")
    op("l3-token-share", "Tandai token telah dibagikan", A, "Bagikan token lewat kanal luar; tandai shared",
       "Token belum shared, consumed, revoked atau expired",
       ["Isi shared_at; audit MARK_TOKEN_SHARED", "Commit; respons 204"],
       auth+"lifecycle.go", "MarkTokenShared",
       ["POST /user-invitations/{id}/mark-shared", "POST /password-resets/{id}/mark-shared"],
       "AUTH-USR-003 / AUTH-PWD-003",
       "Mark-shared bukan syarat Consume. UI hanya menampilkan token undangan; endpoint mark-shared belum punya kontrol UI.")
    op("l3-token-revoke", "Cabut token undangan atau reset", A, "Pilih token untuk dicabut",
       "Token belum consumed atau revoked",
       ["Isi revoked_at; audit pencabutan token", "Commit; pemakaian token berikutnya ditolak"],
       auth+"lifecycle.go", "RevokeToken",
       ["POST /user-invitations/{id}/revoke", "POST /password-resets/{id}/revoke"],
       "ACT-AUTH-INVITE-REVOKE / ACT-AUTH-RESET-REVOKE", "Tidak ada job yang mengubah status EXPIRED; expiry diperiksa saat akses.")
    op("l3-activate-invitation", "Aktivasi pengguna dari undangan", P, "Buka /activate; kirim token dan password",
       "Password valid; lock token dan user; token valid; user INVITED",
       ["Consume token; hash password; ubah user ACTIVE", "Audit ACTIVATE_INVITATION; commit"],
       auth+"lifecycle.go", "Consume", ["POST /auth/activate-invitation"], "AUTH-USR-005",
       "Token valid tidak mensyaratkan shared_at. Hash password dibuat oleh LifecycleService sebelum transaksi Consume.",
       "400; token/password ditolak")
    op("l3-reset-request", "Terbitkan token reset password", A, "Pilih user yang membutuhkan reset",
       "Target user harus ACTIVE",
       ["Simpan hash reset token dengan expiry 24 jam", "Audit CREATE_PASSWORD_RESET; tampilkan token sekali"],
       auth+"lifecycle.go", "CreateReset", ["POST /users/{id}/password-resets"], "AUTH-PWD-001",
       "Permintaan reset dimulai Super Admin; tidak ada endpoint forgot-password publik.")
    op("l3-reset-consume", "Terapkan password baru", P, "Buka /reset-password; kirim token dan password",
       "Lock token dan user; token reset valid; user ACTIVE",
       ["Consume token; simpan hash password baru", "Cabut seluruh sesi user; audit RESET_PASSWORD; commit"],
       auth+"lifecycle.go", "Consume", ["POST /auth/reset-password"], "AUTH-PWD-005",
       "Tidak ada state user LOCKED atau langkah unlock di implementasi.",
       "400; token/password ditolak")
    op("l3-user-status", "Aktivasi administratif dan deaktivasi pengguna", A, "Pilih activate atau deactivate user",
       "Cek versi; cegah deaktivasi diri/owner terakhir",
       ["Ubah status dan versi; saat deaktivasi cabut sesi dan token", "Audit SET_USER_STATUS; commit"],
       auth+"users.go", "SetUserStatus",
       ["POST /users/{id}/activate", "POST /users/{id}/deactivate"], "ACT-AUTH-USER-STATUS",
       "API status menerima target ACTIVE/DEACTIVATED; tidak memvalidasi transisi status sebelumnya atau password_hash. Aktivasi administratif berbeda dari aktivasi undangan. Guard owner memakai advisory lock.")
    op("l3-session-revoke", "Cabut seluruh sesi pengguna", A, "Pilih user untuk revoke sessions",
       "Akses Super Admin; jalankan transaksi target user",
       ["Set revoked_at untuk semua sesi target yang masih aktif", "Audit REVOKE_SESSIONS; commit"],
       auth+"users.go", "RevokeSessions", ["POST /users/{id}/revoke-sessions"], "ACT-AUTH-SESSION-REVOKE",
       "Tidak mengubah role atau password; tidak mencabut action_tokens.")
    op("l3-master-create", "Buat master data", R, "Pilih jenis; isi kode dan nama",
       "Kind di whitelist; input valid; constraint database",
       ["Insert unit, product, supplier, customer atau warehouse", "Audit CREATE_MASTER_DATA; commit"],
       md+"store.go", "Create", ["POST /master-data/{kind}"], "CAP-MDM-001",
       "Untuk product, INSERT SELECT unit code='PCS'; RowsAffected belum diperiksa. Bila PCS tidak ada, respons sukses/audit dapat muncul tanpa product. Tidak ada endpoint edit nama atau hapus.")
    op("l3-master-status", "Aktif/nonaktifkan master data", R, "Pilih master dan status aktif",
       "Kind valid; id dan version cocok",
       ["Update active dan version", "Audit SET_MASTER_DATA_STATUS; commit"],
       md+"store.go", "SetActive", ["PATCH /master-data/{kind}/{id}/status"], "CAP-MDM-001")
    op("l3-settings", "Ubah pengaturan organisasi", A, "Isi organisasi, currency, timezone, warehouse",
       "Currency 3 karakter; timezone dan warehouse terisi",
       ["Update singleton organization_settings", "Audit UPDATE_SETTINGS; commit"],
       md+"settings.go", "Update", ["PATCH /settings/organization"], "CAP-MDM-001",
       "Warehouse mengikuti FK; validasi domain tidak memeriksa timezone IANA, kode ISO atau warehouse aktif. Baca melalui GET pada route yang sama.")
    op("l3-pr-create", "Buat Purchase Request", M, "Isi tujuan pembelian dan line produk",
       "Tujuan terisi; line ada; qty positif; harga nonnegatif; produk aktif",
       ["Insert PR DRAFT, creator dan lines", "Audit PUR-PR-001; commit"],
       pur, "CreateRequest", ["POST /purchase-requests"], "PUR-PR-001")
    op("l3-pr-submit", "Submit Purchase Request", M, "Kirim PR untuk review",
       "Lock PR; status DRAFT dan version cocok",
       ["Ubah PR menjadi SUBMITTED; naikkan versi", "Audit PUR-PR-002; commit"],
       pur, "TransitionRequest", ["POST /purchase-requests/{id}/submit"], "PUR-PR-002",
       "Tidak ada pemeriksaan actor==creator untuk submit.")
    op("l3-pr-review", "Review Purchase Request", R, "Pilih approve atau reject; isi alasan reject",
       "Lock PR; SUBMITTED; versi cocok; Supervisor bukan creator; Super Admin boleh self-review",
       ["Approve: APPROVED; reject beralasan: REJECTED", "Simpan reviewer; audit PUR-PR-003; commit"],
       pur, "TransitionRequest", ["POST /purchase-requests/{id}/approve", "POST /purchase-requests/{id}/reject"],
       "PUR-PR-003", "Bypass self-review Super Admin memeriksa role aktif di database; guard status/versi tetap berlaku.")
    op("l3-po-convert", "Konversi PR menjadi Purchase Order", M, "Pilih PR, supplier dan warehouse",
       "Lock PR APPROVED; supplier dan warehouse aktif",
       ["Salin PR lines ke PO DRAFT baru", "PR menjadi CONVERTED_TO_PO; audit; commit"],
       pur, "ConvertToOrder", ["POST /purchase-requests/{id}/purchase-order"], "PUR-PO-001",
       "Tidak ada endpoint edit PO. Konversi tidak memeriksa creator PR sama dengan actor.")
    op("l3-po-issue", "Terbitkan Purchase Order", R, "Review dan issue PO draft",
       "PO DRAFT; version cocok; Supervisor bukan creator; Super Admin boleh self-issue",
       ["Update PO menjadi ISSUED", "Audit PUR-PO-002; commit"],
       pur, "IssueOrder", ["POST /purchase-orders/{id}/issue"], "PUR-PO-002",
       "Tidak ada email atau pengiriman otomatis PO ke supplier.")
    op("l3-gr-create", "Catat penerimaan barang", M, "Pilih PO line dan jumlah diterima",
       "PO ISSUED/PARTIALLY_RECEIVED; line milik PO; qty positif",
       ["Insert Goods Receipt DRAFT dan lines", "Audit PUR-GR-001; commit"],
       pur, "CreateReceipt", ["POST /purchase-orders/{id}/goods-receipts"], "PUR-GR-001",
       "Batas sisa kuantitas diperiksa lagi saat post.")
    op("l3-gr-post", "Posting penerimaan dan stok masuk", R, "Review dan post Goods Receipt",
       "Lock GR/PO/lines; draft dan versi cocok; Supervisor bukan creator; Super Admin boleh self-post; qty tidak melebihi sisa",
       ["Tambahkan saldo dan immutable stock movements", "Update GR POSTED; PO PARTIALLY_RECEIVED atau RECEIVED", "Audit PUR-GR-002; commit atomik"],
       pur, "PostReceipt", ["POST /goods-receipts/{id}/post"], "PUR-GR-002")
    op("l3-pr-cancel", "Batalkan PR draft", M, "Isi alasan pembatalan PR",
       "PR DRAFT; versi cocok; actor creator atau Super Admin; alasan ada",
       ["Ubah PR CANCELLED; naikkan versi", "Audit ACT-PUR-PR-CANCEL; commit"],
       pur, "TransitionRequest", ["POST /purchase-requests/{id}/cancel"], "ACT-PUR-PR-CANCEL")
    op("l3-pr-copy", "Salin PR ditolak menjadi draft baru", M, "Pilih Copy to Draft",
       "PR sumber REJECTED; semua produk masih aktif",
       ["Salin purpose/lines ke PR baru; isi copied_from_id", "PR sumber tetap REJECTED; audit; commit"],
       pur, "CopyRequest", ["POST /purchase-requests/{id}/copy"], "ACT-PUR-PR-COPY")
    op("l3-po-cancel", "Batalkan Purchase Order", R, "Isi alasan pembatalan PO",
       "PO DRAFT/ISSUED; tidak ada GR POSTED; alasan ada",
       ["Ubah PO CANCELLED; PR asal kembali APPROVED", "Audit PO dan PR; commit atomik"],
       pur, "CancelOrder", ["POST /purchase-orders/{id}/cancel"], "ACT-PUR-PO-CANCEL",
       "PO batal tetap tercatat. PR dapat dikonversi ke satu PO pengganti aktif; tidak ada owner check atau version input.")
    op("l3-gr-cancel", "Batalkan Goods Receipt draft", M, "Isi alasan pembatalan GR",
       "GR DRAFT; alasan tidak kosong",
       ["Ubah GR CANCELLED; naikkan versi", "Audit ACT-PUR-GR-CANCEL; commit"],
       pur, "CancelReceipt", ["POST /goods-receipts/{id}/cancel"], "ACT-PUR-GR-CANCEL")
    op("l3-gr-reverse", "Reverse Goods Receipt yang sudah posted", R, "Isi alasan reversal",
       "Lock GR POSTED; tidak ada invoice non-VOID; stok akhir nonnegatif",
       ["Kurangi saldo/received qty; append movement inverse", "GR REVERSED; PO ISSUED atau PARTIALLY_RECEIVED", "Audit reversal; commit atomik"],
       pur, "ReverseReceipt", ["POST /goods-receipts/{id}/reverse"], "ACT-PUR-GR-REVERSE",
       "Movement lama tidak diedit/dihapus. Tidak ada syarat reviewer berbeda dari creator untuk reversal.")
    op("l3-so-create", "Buat Sales Order", M, "Pilih customer, warehouse dan line penjualan",
       "Master aktif; line ada; qty positif; harga nonnegatif",
       ["Insert SO DRAFT, creator dan lines", "Audit SAL-SO-001; commit"],
       sal, "CreateOrder", ["POST /sales-orders"], "SAL-SO-001")
    op("l3-so-edit", "Revisi line Sales Order draft", M, "Edit line SO",
       "Lock SO DRAFT; versi cocok; validasi produk dan nilai line",
       ["Ganti lines; naikkan versi SO", "Audit UPDATE_SALES_ORDER; commit"],
       sal, "UpdateOrder", ["PATCH /sales-orders/{id}"], "SAL-SO-001",
       "Tidak ada pemeriksaan editor==creator; maker-checker selanjutnya hanya membandingkan creator_id.")
    op("l3-so-stock", "Periksa ketersediaan stok SO", M, "Pilih Validate Stock",
       "Lock SO DRAFT; versi cocok; saldo cukup untuk tiap line",
       ["Audit VALIDATE_SALES_ORDER_STOCK", "Commit; SO tetap DRAFT; tanpa reservasi stok"],
       sal, "ValidateStock", ["POST /sales-orders/{id}/validate-stock"], "SAL-SO-003")
    op("l3-so-submit", "Submit Sales Order", M, "Kirim SO untuk review",
       "Lock SO DRAFT; versi cocok; cek stok ulang",
       ["Audit validasi stok; SO SUBMITTED", "Audit SAL-SO-002; commit"],
       sal, "TransitionOrder", ["POST /sales-orders/{id}/submit"], "SAL-SO-002 / SAL-SO-003")
    op("l3-so-review", "Review Sales Order", R, "Approve atau reject SO",
       "SUBMITTED; versi cocok; Supervisor bukan creator; Super Admin boleh self-review; approve perlu stok cukup",
       ["Approve: APPROVED; reject beralasan: REJECTED", "Simpan reviewer; audit SAL-SO-004; commit"],
       sal, "TransitionOrder", ["POST /sales-orders/{id}/approve", "POST /sales-orders/{id}/reject"],
       "SAL-SO-004", "Cek stok tidak mereservasi stok. Reject wajib alasan.")
    op("l3-so-confirm", "Konfirmasi Sales Order approved", M, "Pilih Confirm",
       "SO APPROVED; versi cocok; belum confirmed",
       ["Isi confirmed_at; status tetap APPROVED", "Naikkan versi; audit SAL-SO-005; commit"],
       sal, "TransitionOrder", ["POST /sales-orders/{id}/confirm"], "SAL-SO-005",
       "Confirm dipicu Staff/Super Admin melalui API; bukan aksi background otomatis.")
    op("l3-delivery-create", "Buat Delivery", M, "Pilih SO line dan jumlah dikirim",
       "SO APPROVED/PARTIALLY_DELIVERED dan confirmed; line valid; qty positif",
       ["Insert delivery DRAFT dan lines", "Audit SAL-DLV-001; commit"],
       sal, "CreateDelivery", ["POST /sales-orders/{id}/deliveries"], "SAL-DLV-001")
    op("l3-delivery-post", "Posting delivery dan stok keluar", M, "Post Delivery",
       "Lock delivery/SO/lines/saldo; versi cocok; sisa dan stok cukup",
       ["Kurangi stok; append stock movement keluar", "Delivery POSTED; SO PARTIALLY_DELIVERED atau DELIVERED", "Audit SAL-DLV-002; commit atomik"],
       sal, "PostDelivery", ["POST /deliveries/{id}/post"], "SAL-DLV-002")
    op("l3-so-cancel", "Batalkan Sales Order", M, "Isi alasan pembatalan SO",
       "SO DRAFT/APPROVED; versi cocok; actor creator atau Super Admin",
       ["Ubah SO CANCELLED", "Audit ACT-SAL-SO-CANCEL; commit"],
       sal, "TransitionOrder", ["POST /sales-orders/{id}/cancel"], "ACT-SAL-SO-CANCEL",
       "Alasan wajib. Tidak ada cancel untuk SO PARTIALLY_DELIVERED.")
    op("l3-so-copy", "Salin SO ditolak menjadi draft baru", M, "Pilih Copy to Draft",
       "SO sumber REJECTED; master sumber masih aktif",
       ["Salin customer, warehouse dan lines ke SO DRAFT baru", "Isi copied_from_id; audit; commit"],
       sal, "CopyOrder", ["POST /sales-orders/{id}/copy"], "ACT-SAL-SO-COPY")
    op("l3-delivery-cancel", "Batalkan delivery draft", M, "Isi alasan pembatalan delivery",
       "Delivery DRAFT; alasan tidak kosong",
       ["Ubah delivery CANCELLED", "Audit ACT-SAL-DLV-CANCEL; commit"],
       sal, "CancelDelivery", ["POST /deliveries/{id}/cancel"], "ACT-SAL-DLV-CANCEL")
    op("l3-delivery-reverse", "Reverse delivery posted", R, "Isi alasan reversal delivery",
       "Lock delivery POSTED; tidak ada invoice penjualan non-VOID",
       ["Tambah kembali stok; append movement inverse; kurangi delivered qty", "Delivery REVERSED; SO APPROVED atau PARTIALLY_DELIVERED", "Audit reversal; commit"],
       sal, "ReverseDelivery", ["POST /deliveries/{id}/reverse"], "ACT-SAL-DLV-REVERSE")
    op("l3-adj-create", "Catat stock opname / adjustment", M, "Pilih warehouse, produk dan counted quantity",
       "Warehouse/produk aktif; line ada; counted quantity nonnegatif",
       ["Snapshot saldo; simpan counted quantity pada adjustment DRAFT", "Audit INV-ADJ-001; commit"],
       inv, "CreateAdjustment", ["POST /inventory-adjustments"], "INV-ADJ-001",
       "Tidak ada endpoint edit atau copy adjustment saat ini.")
    op("l3-adj-submit", "Submit stock adjustment", M, "Kirim adjustment untuk review",
       "Lock adjustment DRAFT; versi cocok",
       ["Ubah adjustment SUBMITTED", "Audit INV-ADJ-002; commit"],
       inv, "TransitionAdjustment", ["POST /inventory-adjustments/{id}/submit"], "INV-ADJ-002")
    op("l3-adj-review", "Review stock adjustment", R, "Approve atau reject adjustment",
       "SUBMITTED; versi cocok; Supervisor bukan creator; Super Admin boleh self-review; approve perlu snapshot masih sama",
       ["Approve: APPROVED; reject beralasan: REJECTED", "Simpan reviewer; audit INV-ADJ-003; commit"],
       inv, "TransitionAdjustment", ["POST /inventory-adjustments/{id}/approve", "POST /inventory-adjustments/{id}/reject"],
       "INV-ADJ-003", "Jika saldo berubah, approve ditolak; belum ada refresh snapshot atau revisi adjustment.")
    op("l3-adj-post", "Posting selisih stock adjustment", M, "Post adjustment approved",
       "Lock adjustment APPROVED dan saldo; versi dan snapshot cocok",
       ["Hitung counted minus snapshot; update saldo dan movement nonzero", "Ubah adjustment COMPLETED", "Audit INV-ADJ-004 dan INV-ADJ-005; commit atomik"],
       inv, "PostAdjustment", ["POST /inventory-adjustments/{id}/post"], "INV-ADJ-004 / INV-ADJ-005")
    op("l3-adj-cancel", "Batalkan adjustment draft atau approved", M, "Isi alasan pembatalan adjustment",
       "DRAFT atau APPROVED; versi cocok; creator atau Super Admin aktif; alasan ada",
       ["Ubah adjustment CANCELLED", "Audit ACT-INV-ADJ-CANCEL; commit"],
       inv, "TransitionAdjustment", ["POST /inventory-adjustments/{id}/cancel"], "ACT-INV-ADJ-CANCEL",
       "Jika stok berubah setelah APPROVED, Staff membatalkan dengan alasan, membuat draft baru dari snapshot terkini, lalu meminta approval ulang.")
    op("l3-adj-reverse", "Reverse adjustment selesai", R, "Isi alasan reversal adjustment",
       "Lock adjustment COMPLETED; saldo setelah inverse nonnegatif",
       ["Append movement inverse; ubah saldo", "Adjustment REVERSED; audit; commit"],
       inv, "ReverseAdjustment", ["POST /inventory-adjustments/{id}/reverse"], "ACT-INV-ADJ-REVERSE")
    op("l3-ap-invoice", "Catat invoice supplier", M, "Isi invoice supplier dan referensi PO",
       "PO RECEIVED; metadata/lines valid; aturan invoice pengganti terpenuhi",
       ["Simpan invoice MATCH_PENDING dan line invoice", "Audit PUR-INV-001; commit"],
       fin, "CreateSupplierInvoice", ["POST /supplier-invoices"], "PUR-INV-001",
       "Hanya PO diterima penuh. Supervisor melakukan match setelah Staff membuat invoice.")
    op("l3-ap-match", "Match invoice supplier", R, "Pilih Match",
       "Lock invoice MATCH_PENDING; Supervisor bukan creator; Super Admin boleh self-match; baca line PO dan invoice",
       ["ExactMatch: cocok menjadi POSTED; tidak cocok MATCH_EXCEPTION", "Simpan mismatch_reason dan versi; audit PUR-INV-002; commit"],
       fin, "MatchSupplierInvoice", ["POST /supplier-invoices/{id}/match"], "PUR-INV-002",
       "Kode membandingkan line PO dengan invoice. Bukan query three-way match GR terpisah pada saat match; PO RECEIVED sudah disyaratkan saat create.")
    op("l3-ap-resolve", "Perbaiki mismatch invoice supplier", M, "Revisi lines invoice exception",
       "Lock invoice MATCH_EXCEPTION; line qty/harga valid",
       ["Ganti lines dan total; outstanding direset ke total", "Status MATCH_PENDING; audit PUR-INV-003; commit"],
       fin, "ResolveSupplierInvoice", ["POST /supplier-invoices/{id}/resolve-match"], "PUR-INV-003")
    op("l3-ar-invoice", "Buat draft invoice penjualan", M, "Pilih SO; isi tanggal dan currency",
       "Lock SO DELIVERED; total positif; aturan replacement terpenuhi",
       ["Ambil delivered qty dan harga SO; simpan invoice DRAFT", "Audit SAL-INV-001; commit"],
       fin, "CreateSalesInvoice", ["POST /sales-orders/{id}/sales-invoice"], "SAL-INV-001",
       "Invoice konsolidasi setelah seluruh delivery. Supervisor harus post sebelum dapat dibayar.")
    op("l3-ar-invoice-post", "Post invoice penjualan", R, "Pilih Post Invoice",
       "Lock invoice DRAFT; versi cocok; SO masih DELIVERED; Supervisor bukan creator, Super Admin aktif boleh self-post",
       ["Ubah invoice menjadi POSTED dan naikkan versi", "Audit SAL-INV-001; commit"],
       fin, "PostSalesInvoice", ["POST /sales-invoices/{id}/post"], "SAL-INV-001")
    for side, party, create, post, prefix, route in [
        ("ap", "supplier", "CreateSupplierPayment", "PostSupplierPayment", "PUR", "supplier-payments"),
        ("ar", "customer", "CreateCustomerPayment", "PostCustomerPayment", "SAL", "customer-payments"),
    ]:
        op(f"l3-{side}-payment", f"Catat pembayaran {party}", M, f"Isi pembayaran {party} dan alokasi invoice",
           "Party aktif; alokasi unik sesuai amount; metadata terisi; constraint DB",
           ["Simpan payment DRAFT dan alokasi invoice", f"Audit {prefix}-PAY-001; commit"],
           fin, create, [f"POST /{route}"], f"{prefix}-PAY-001",
           "Pembayaran dicatat, tidak mengeksekusi transfer bank. Validasi invoice dan outstanding diperketat saat post.")
        close = "PO RECEIVED ke CLOSED; PR ke COMPLETED" if side == "ap" else "SO DELIVERED ke COMPLETED"
        op(f"l3-{side}-payment-post", f"Posting pembayaran {party}", R, "Pilih Post Payment",
           "Lock payment/invoice; DRAFT dan versi cocok; party/currency/tanggal/outstanding valid; Supervisor bukan creator, Super Admin aktif boleh self-post",
           ["Kurangi outstanding; invoice PARTIALLY_PAID atau PAID",
            f"Jika semua invoice non-VOID lunas: {close}",
            "Payment POSTED; audit alokasi/penutupan; commit atomik"],
           fin, post, [f"POST /{route}/{{id}}/post"], f"{prefix}-PAY-002",
           "Created_by dibandingkan dengan actor, kecuali Super Admin aktif sebagai bypass.")
        voidfn = "VoidSupplierInvoice" if side == "ap" else "VoidSalesInvoice"
        invroute = "supplier-invoices" if side == "ap" else "sales-invoices"
        op(f"l3-{side}-invoice-void", f"Void invoice {party}", R, "Isi alasan void invoice",
           "Invoice POSTED; untuk AR DRAFT juga boleh; tanpa alokasi dari payment POSTED; alasan ada",
           ["Tandai invoice VOID; simpan voided_by dan alasan", "Audit void; commit"],
           fin, voidfn, [f"POST /{invroute}/{{id}}/void"],
           "ACT-FIN-SINV-VOID" if side == "ap" else "ACT-FIN-CINV-VOID",
           "Tidak membuat jurnal akuntansi baru; aplikasi tidak memiliki general ledger.")
        voidpay = "VoidSupplierPayment" if side == "ap" else "VoidCustomerPayment"
        reopen = "PO CLOSED ke RECEIVED; PR COMPLETED ke CONVERTED_TO_PO" if side == "ap" else "SO COMPLETED ke DELIVERED"
        op(f"l3-{side}-payment-void", f"Void pembayaran {party}", R, "Isi alasan void payment",
           "Lock payment POSTED dan invoice terkait; alasan ada",
           ["Kembalikan outstanding; hitung ulang status invoice",
            f"Jika tidak lunas lagi: {reopen}",
            "Payment VOID; audit reversal alokasi; commit"],
           fin, voidpay, [f"POST /{route}/{{id}}/void"],
           "ACT-FIN-SPAY-VOID" if side == "ap" else "ACT-FIN-CPAY-VOID",
           "Alokasi historis dipertahankan; efek payment VOID dikecualikan dari hitungan pembayaran aktif.")
    op("l3-operational-read", "Baca dokumen dan saldo operasional", U, "Buka daftar atau detail operasional",
       "Sesi aktif; ID detail harus ditemukan",
       ["Query daftar/detail sesuai endpoint", "Kirim JSON; tidak mengubah dokumen"],
       pur, "ListRequests / ListOrders / GetOrder / ListReceipts",
       ["GET /purchase-requests", "GET /purchase-orders", "GET /purchase-orders/{id}", "GET /goods-receipts",
        "GET /sales-orders", "GET /sales-orders/{id}", "GET /deliveries", "GET /inventory/balances",
        "GET /inventory/movements", "GET /inventory-adjustments", "GET /master-data/{kind}",
        "GET /auth/session"], note="Sumber tambahan: sales Store.ListOrders/GetOrder/ListDeliveries; inventory Store.Balances/Movements/ListAdjustments; masterdata Store.List; auth/session handler inline main.go. Daftar lintas creator; movement dibatasi 500.")
    op("l3-admin-read", "Baca pengguna dan pengaturan", A, "Buka Users atau Settings",
       "Sesi aktif; role SUPER_ADMIN",
       ["Query users atau organization_settings", "Kirim JSON tanpa perubahan"],
       auth+"users.go", "ListUsers", ["GET /users", "GET /settings/organization"],
       note="Sumber settings: internal/masterdata/adapter/postgres/settings.go Get.")
    op("l3-finance-read", "Baca invoice dan payment", U, "Buka Finance",
       "Semua role aktif dapat membaca AP dan AR",
       ["Query daftar AP dan AR untuk semua role", "Tampilkan status, total dan outstanding"],
       fin, "ListSupplierInvoices / ListSupplierPayments / ListSalesInvoices / ListCustomerPayments",
       ["GET /supplier-invoices", "GET /supplier-payments", "GET /sales-invoices", "GET /customer-payments"])
    op("l3-dashboard", "Baca ringkasan Dashboard", U, "Buka Dashboard",
       "Sesi aktif; query agregat berhasil",
       ["Hitung approval, order aktif, zero stock, utang/piutang dan overdue", "Kirim ringkasan JSON"],
       rpt, "Summary", ["GET /dashboard/summary"],
       note="Endpoint sama untuk semua role, termasuk nilai keuangan. Zero stock memakai default warehouse organisasi.")
    op("l3-report", "Baca laporan dan audit trail", R, "Pilih laporan atau filter audit",
       "Role Supervisor/Super Admin; query berhasil",
       ["Query purchases, sales, inventory, AP/AR atau audit", "Kirim hasil; tidak memutasi transaksi"],
       rpt, "Purchases / Sales / Inventory / Payables / Receivables / Audit",
       ["GET /reports/purchases", "GET /reports/sales", "GET /reports/inventory",
        "GET /reports/payables", "GET /reports/receivables", "GET /audit-logs"],
       note="Laporan finansial limit 100; inventory 500; audit 200 dengan filter entity/actor/action. Tidak ada route export/PDF atau pagination.")

    # Shared HTTP behavior is drawn separately rather than implying every module
    # has identical middleware or that DB writes can partially commit.
    m = Model("l3-kontrol-request", 3, "Kontrol request, otorisasi dan idempotency",
              [("backend/cmd/api/main.go", "main"), ("backend/internal/platform/idempotency/middleware.go", "Middleware"),
               ("backend/internal/auth/adapter/http/middleware.go", "RequireAuth / RequireRole")],
              "Berlaku untuk route terproteksi. Idempotency hanya group purchasing, sales, inventory, finance, reporting. Auth/admin/master/settings di luar middleware idempotency. Role checks handler berada setelah middleware idempotency pada group operasi.")
    for key, label, kind, lane in [
        ("start", "Request HTTP", "startEvent", U),
        ("auth", "Cek cookie, user ACTIVE, rate limit, CSRF/origin, dan tipe route", "serviceTask", S),
        ("authok", "Hasil akses dan tipe route?", "exclusiveGateway", S),
        ("denied", "401 / 403 / 429", "endEvent", S),
        ("idem", "Validasi key; claim hash method/path/body", "serviceTask", S),
        ("replay", "Hasil klaim key?", "exclusiveGateway", S),
        ("old", "Replay respons tersimpan", "endEvent", S),
        ("conflict", "409: key berbeda atau masih diproses", "endEvent", S),
        ("keyerror", "400 / 500: key atau claim gagal", "endEvent", S),
        ("handler", "Handler cek role; validasi input; jalankan use case", "serviceTask", S),
        ("result", "Simpan respons <500; release key jika 5xx; kirim respons", "serviceTask", S),
        ("end", "Request selesai", "endEvent", S),
    ]:
        m.n(key, label, lane, kind)
    m.chain("start", "auth", "authok")
    m.f("authok", "denied", "Akses ditolak", True)
    m.f("authok", "idem", "Mutasi group operasi")
    m.f("authok", "handler", "Route lain")
    m.chain("idem", "replay")
    m.f("replay", "handler", "Key baru")
    m.f("replay", "old", "Hash sama; respons ada")
    m.f("replay", "conflict", "Konflik / in progress")
    m.f("replay", "keyerror", "Key / claim gagal", True)
    m.chain("handler", "result", "end")
    op("l3-health", "Pemeriksaan liveness aplikasi", "Pemantau deployment", "GET healthz",
       "HTTP server dapat merespons",
       ["Kembalikan status ok (200)"], "backend/internal/platform/httpserver/router.go", "NewRouter",
       ["GET /healthz"], note="Healthz tidak mengecek database secara live; ini liveness, bukan readiness.")

    # L2 decomposes business lifecycles, with explicit checks between separate requests.
    menu("l2-access", 2, "Undangan dan pengelolaan pengguna", [
        ("Bootstrap awal", "l3-bootstrap"), ("Undang pengguna", "l3-invite"),
        ("Tandai token dibagikan", "l3-token-share"), ("Aktivasi undangan", "l3-activate-invitation"),
        ("Status pengguna", "l3-user-status"), ("Cabut sesi", "l3-session-revoke"),
        ("Cabut token", "l3-token-revoke"), ("Baca pengguna", "l3-admin-read")],
        note="Masing-masing child memiliki lane role sendiri; pilihan pada level ini adalah katalog lifecycle, bukan izin semua role.")
    menu("l2-session", 2, "Login, sesi dan pemulihan password", [
        ("Login", "l3-login"), ("Refresh sesi", "l3-refresh"), ("Logout", "l3-logout"),
        ("Terbitkan reset oleh owner", "l3-reset-request"), ("Gunakan token reset", "l3-reset-consume")])
    menu("l2-master-settings", 2, "Pemeliharaan master dan konfigurasi", [
        ("Baca master", "l3-operational-read"), ("Buat master", "l3-master-create"),
        ("Ubah status master", "l3-master-status"), ("Baca settings", "l3-admin-read"),
        ("Ubah settings", "l3-settings")])
    life("l2-pr", "Purchase Request dari draft sampai review", [
        ("Buat PR", M, "l3-pr-create", "PR DRAFT berhasil?"),
        ("Submit PR", M, "l3-pr-submit", "PR SUBMITTED?"),
        ("Review PR", R, "l3-pr-review", "PR APPROVED?"),
    ], [(pur, "CreateRequest / TransitionRequest")], "REJECTED dapat disalin lewat l2-purchase-correction; bukan edit atau resubmit PR lama.")
    life("l2-po-gr", "Purchase Order dan penerimaan barang", [
        ("Konversi PR APPROVED", M, "l3-po-convert", "PO DRAFT berhasil?"),
        ("Supervisor issue PO", R, "l3-po-issue", "PO ISSUED?"),
        ("Catat penerimaan", M, "l3-gr-create", "GR DRAFT berhasil?"),
        ("Supervisor post penerimaan", R, "l3-gr-post", "GR POSTED?"),
    ], [(pur, "ConvertToOrder / IssueOrder / CreateReceipt / PostReceipt")],
       "Satu iterasi penerimaan. Jika PO PARTIALLY_RECEIVED, buat dan post GR berikutnya; hanya PO RECEIVED dapat dibuatkan invoice.")
    life("l2-so", "Sales Order dari draft sampai konfirmasi", [
        ("Buat SO", M, "l3-so-create", "SO DRAFT berhasil?"),
        ("Submit dan cek stok", M, "l3-so-submit", "SO SUBMITTED?"),
        ("Review SO", R, "l3-so-review", "SO APPROVED?"),
        ("Confirm SO", M, "l3-so-confirm", "confirmed_at terisi?"),
    ], [(sal, "CreateOrder / TransitionOrder")], "Edit dan validasi stok manual tersedia melalui l2-sales-correction. Submit/approve juga memeriksa stok.")
    life("l2-delivery", "Pengiriman barang dan pencatatan stok keluar", [
        ("Buat delivery untuk SO confirmed", M, "l3-delivery-create", "Delivery DRAFT?"),
        ("Post delivery", M, "l3-delivery-post", "Delivery POSTED?"),
    ], [(sal, "CreateDelivery / PostDelivery")],
       "Ulangi untuk sisa bila SO PARTIALLY_DELIVERED. Invoice hanya setelah SO DELIVERED.")
    life("l2-adjustment", "Stock adjustment dengan review terpisah", [
        ("Catat hasil hitung dan snapshot", M, "l3-adj-create", "Adjustment DRAFT?"),
        ("Submit adjustment", M, "l3-adj-submit", "Adjustment SUBMITTED?"),
        ("Review dan cek snapshot", R, "l3-adj-review", "Adjustment APPROVED?"),
        ("Post selisih", M, "l3-adj-post", "Adjustment COMPLETED?"),
    ], [(inv, "CreateAdjustment / TransitionAdjustment / PostAdjustment")],
       "Review dan posting gagal bila snapshot tidak sama. Bila sudah APPROVED, creator dapat cancel dengan alasan lalu membuat draft baru dan meminta approval ulang; tidak ada refresh snapshot otomatis.")
    life("l2-ap", "Invoice supplier sampai pelunasan", [
        ("Staff catat invoice setelah PO RECEIVED", M, "l3-ap-invoice", "Invoice MATCH_PENDING?"),
        ("Match invoice", R, "l3-ap-match", "Invoice POSTED?"),
        ("Staff catat pembayaran supplier", M, "l3-ap-payment", "Payment DRAFT?"),
        ("Post pembayaran", R, "l3-ap-payment-post", "Payment POSTED?"),
    ], [(fin, "CreateSupplierInvoice / MatchSupplierInvoice / PostSupplierPayment")],
       "MATCH_EXCEPTION: Staff resolve lalu Supervisor match ulang. Pembayaran parsial diulang; PO/PR ditutup hanya jika semua invoice non-VOID PAID. Supervisor tidak boleh match/post dokumen sendiri; Super Admin boleh bypass.")
    life("l2-ar", "Invoice penjualan sampai penerimaan pembayaran", [
        ("Staff buat invoice setelah SO DELIVERED", M, "l3-ar-invoice", "Invoice DRAFT?"),
        ("Supervisor post invoice", R, "l3-ar-invoice-post", "Invoice POSTED?"),
        ("Staff catat pembayaran customer", M, "l3-ar-payment", "Payment DRAFT?"),
        ("Post pembayaran", R, "l3-ar-payment-post", "Payment POSTED?"),
    ], [(fin, "CreateSalesInvoice / PostSalesInvoice / CreateCustomerPayment / PostCustomerPayment")],
       "Supervisor tidak boleh post dokumen buatannya sendiri; Super Admin boleh bypass. Pembayaran dapat parsial; SO COMPLETED hanya setelah semua invoice non-VOID PAID. Tidak ada transfer bank otomatis.")
    menu("l2-purchase-correction", 2, "Pembatalan dan koreksi Purchasing", [
        ("Batalkan PR", "l3-pr-cancel"), ("Copy PR ditolak", "l3-pr-copy"),
        ("Batalkan PO", "l3-po-cancel"), ("Batalkan GR draft", "l3-gr-cancel"), ("Reverse GR", "l3-gr-reverse")])
    menu("l2-sales-correction", 2, "Revisi, pembatalan dan koreksi Sales", [
        ("Edit SO draft", "l3-so-edit"), ("Cek stok manual", "l3-so-stock"),
        ("Cancel SO", "l3-so-cancel"), ("Copy SO ditolak", "l3-so-copy"),
        ("Cancel delivery draft", "l3-delivery-cancel"), ("Reverse delivery", "l3-delivery-reverse")])
    menu("l2-inventory-correction", 2, "Koreksi adjustment dan pembacaan stok", [
        ("Baca saldo/movement", "l3-operational-read"), ("Cancel draft/approved", "l3-adj-cancel"), ("Reverse completed", "l3-adj-reverse")])
    menu("l2-finance-correction", 2, "Koreksi invoice dan payment", [
        ("Resolve mismatch supplier", "l3-ap-resolve"), ("Match ulang", "l3-ap-match"),
        ("Void payment supplier", "l3-ap-payment-void"), ("Void invoice supplier", "l3-ap-invoice-void"),
        ("Void payment customer", "l3-ar-payment-void"), ("Void invoice sales", "l3-ar-invoice-void")], U,
        "Bila invoice sudah dibayar, void payment dulu; invoice void hanya bila tanpa alokasi payment POSTED. Invoice pengganti memakai create dengan replacesInvoiceId.")
    menu("l2-p2p-correction", 2, "Koreksi khusus Procure to Pay", [
        ("Koreksi PR, PO, atau GR", "l2-purchase-correction"),
        ("Resolve mismatch invoice", "l3-ap-resolve"),
        ("Match ulang invoice", "l3-ap-match"), ("Void payment supplier", "l3-ap-payment-void"),
        ("Void invoice supplier", "l3-ap-invoice-void")],
        actor="Staff atau Supervisor sesuai aksi",
        note="Katalog alternatif, bukan urutan wajib. Reversal GR perlu invoice non-VOID sudah tidak ada; invoice beralokasi dari payment POSTED memerlukan void payment lebih dahulu. Role final tetap mengikuti guard L3/API.")
    menu("l2-reporting", 2, "Dashboard, daftar, laporan dan audit", [
        ("Dashboard semua role", "l3-dashboard"), ("Daftar/detail operasional", "l3-operational-read"),
        ("Baca finance", "l3-finance-read"), ("Laporan/audit", "l3-report")])
    menu("l2-platform", 2, "Kontrol request dan health", [
        ("Kontrol route terproteksi", "l3-kontrol-request"), ("Health endpoint publik", "l3-health")])

    domains = [
        ("access", "Akses dan pengguna", [("Undangan dan user admin", "l2-access"), ("Sesi dan password", "l2-session")]),
        ("p2p", "Procure to Pay", [("Permintaan dan review", "l2-pr"), ("Order dan penerimaan", "l2-po-gr"), ("Invoice dan pembayaran", "l2-ap"), ("Koreksi Purchasing", "l2-purchase-correction")]),
        ("o2c", "Order to Cash", [("Order dan review", "l2-so"), ("Delivery", "l2-delivery"), ("Invoice dan penerimaan pembayaran", "l2-ar"), ("Koreksi Sales", "l2-sales-correction")]),
        ("inventory", "Pengendalian inventory", [("Stock adjustment", "l2-adjustment"), ("Koreksi dan baca stok", "l2-inventory-correction")]),
        ("finance", "Finance AP dan AR", [("Accounts Payable", "l2-ap"), ("Accounts Receivable", "l2-ar"), ("Koreksi keuangan", "l2-finance-correction")]),
        ("administration", "Master data dan konfigurasi", [("Master dan organisasi", "l2-master-settings"), ("Kontrol platform", "l2-platform")]),
        ("reporting", "Monitoring, laporan dan audit", [("Baca dan telusuri transaksi", "l2-reporting")]),
    ]
    for slug, title, choices in domains:
        menu(f"l1-{slug}", 1, title, choices,
             note="Dekomposisi kapabilitas. Pilihan tahap tidak melewati precondition status di L2/L3; bukan semua tahap dijalankan berurutan oleh gateway.")
    # The two core value chains also show their actual end-to-end sequence.
    for slug, title, stages, correction, sources in [
        ("p2p", "Procure to Pay - permintaan sampai pelunasan", [
            ("Permintaan dan review PR", M, "l2-pr", "PR APPROVED?"),
            ("PO dan penerimaan penuh", M, "l2-po-gr", "PO RECEIVED?"),
            ("Invoice dan pembayaran supplier", "Staff dan Supervisor", "l2-ap", "PO CLOSED dan PR COMPLETED?"),
        ], "l2-p2p-correction", [(pur, "CreateRequest / ConvertToOrder / PostReceipt"), (fin, "PostSupplierPayment")]),
        ("o2c", "Order to Cash - pesanan sampai pelunasan", [
            ("Order, review dan konfirmasi", M, "l2-so", "SO APPROVED dan confirmed?"),
            ("Delivery penuh", M, "l2-delivery", "SO DELIVERED?"),
            ("Invoice dan pembayaran customer", "Staff dan Supervisor", "l2-ar", "SO COMPLETED?"),
        ], "l2-sales-correction", [(sal, "CreateOrder / TransitionOrder / PostDelivery"), (fin, "PostCustomerPayment")]),
    ]:
        model = life(f"l1-{slug}", title, stages, sources,
                     "Alur bisnis lintas request dan role; bukan satu transaksi DB. Koreksi dokumen yang sudah ada adalah jalur alternatif. Detail role berada pada child L2/L3; koreksi Finance melalui l2-finance-correction.")
        model.level = 1
        if slug == "p2p":
            model.note = ("Perspektif bisnis Staff dan Supervisor. Staff menyiapkan PR, PO, GR, invoice dan payment supplier; Supervisor mereview/menerbitkan/mem-post/match. "
                          "Super Admin tidak ditampilkan sebagai pelaku proses, tetapi kode saat ini masih mengizinkan akses operasionalnya. "
                          "Supervisor tidak boleh mengesahkan dokumennya sendiri; Super Admin boleh bypass self-approval. "
                          "Koreksi PR/PO/GR/invoice/payment tersedia melalui l2-p2p-correction.")
            for node in model.nodes:
                if node["lane"] == M:
                    node["lane"] = "Staff dan Supervisor" if node["key"].startswith("stage") else "Staff"
                elif node["lane"] == R:
                    node["lane"] = "Supervisor"
        model.edges = [e for e in model.edges if e["source"] != "start"]
        model.n("mode", "Proses baru atau koreksi?", "Staff" if slug == "p2p" else M, "exclusiveGateway")
        correction_actor = "Staff atau Supervisor sesuai aksi" if slug == "p2p" else U
        model.n("correction", "Koreksi dokumen yang ada", correction_actor, "callActivity", child=correction)
        model.n("corrected", "Upaya koreksi selesai; cek hasil" if slug == "p2p" else "Hasil koreksi diterima", correction_actor, "endEvent")
        model.nodes.insert(1, model.nodes.pop(next(i for i,n in enumerate(model.nodes) if n["key"] == "mode")))
        model.f("start", "mode")
        model.f("mode", "stage0", "Proses baru")
        model.f("mode", "correction", "Koreksi", slug != "p2p")
        model.f("correction", "corrected")
    menu("l0-simple-erp", 0, "Simple ERP - landscape keseluruhan", [
        (title, f"l1-{slug}") for slug, title, _ in domains],
        note="Landscape berformat BPMN: satu kebutuhan dipilih per instansi. Bukan workflow monolitik yang mengharuskan semua modul berjalan. Level 0-3 adalah konvensi dekomposisi paket ini, bukan level wajib standar BPMN.")
    menu("role-staff", 1, "Perspektif role Staff", [
        ("Login", "l3-login"), ("Refresh", "l3-refresh"), ("Logout", "l3-logout"), ("Buat PR", "l3-pr-create"),
        ("Submit PR", "l3-pr-submit"), ("Konversi PO", "l3-po-convert"),
        ("Buat GR", "l3-gr-create"), ("Catat invoice supplier", "l3-ap-invoice"),
        ("Resolve invoice supplier", "l3-ap-resolve"), ("Buat payment supplier", "l3-ap-payment"),
        ("Buat SO", "l3-so-create"), ("Submit SO", "l3-so-submit"), ("Confirm SO", "l3-so-confirm"),
        ("Delivery", "l2-delivery"), ("Buat adjustment", "l3-adj-create"),
        ("Buat invoice sales", "l3-ar-invoice"), ("Buat payment customer", "l3-ar-payment"),
        ("Submit adjustment", "l3-adj-submit"), ("Post adjustment approved", "l3-adj-post"),
        ("Daftar/detail", "l3-operational-read"), ("Dashboard", "l3-dashboard")], M,
        "Fokus aksi operasional utama Staff. Cancel/copy dan edit sesuai matriks ROLES.md. Staff menyiapkan AP dan AR, tetapi tidak match/post/void; tidak memiliki approval, reversal, user admin, atau settings.")
    menu("role-supervisor", 1, "Perspektif role Supervisor", [
        ("Review PR", "l3-pr-review"), ("Review SO", "l3-so-review"), ("Review adjustment", "l3-adj-review"),
        ("Issue PO", "l3-po-issue"), ("Post GR", "l3-gr-post"), ("Cancel PO", "l3-po-cancel"),
        ("Match invoice supplier", "l3-ap-match"), ("Post payment supplier", "l3-ap-payment-post"),
        ("Void invoice supplier", "l3-ap-invoice-void"), ("Void payment supplier", "l3-ap-payment-void"),
        ("Post invoice sales", "l3-ar-invoice-post"), ("Post payment customer", "l3-ar-payment-post"),
        ("Finance AR", "l2-ar"), ("Koreksi Finance", "l2-finance-correction"),
        ("Reverse GR", "l3-gr-reverse"), ("Reverse delivery", "l3-delivery-reverse"),
        ("Reverse adjustment", "l3-adj-reverse"), ("Buat master", "l3-master-create"),
        ("Status master", "l3-master-status"), ("Laporan/audit", "l3-report")], R,
        "Supervisor tidak membuat dokumen operasional maupun draft AP/AR, tetapi mereview atau mem-post dokumen Staff. Master data tetap dapat dipelihara Supervisor.")
    menu("role-super-admin", 1, "Perspektif role Super Admin", [
        ("Akses operasional", "l1-p2p"), ("Akses penjualan", "l1-o2c"), ("Akses inventory", "l1-inventory"),
        ("Finance", "l1-finance"), ("Kelola pengguna", "l2-access"),
        ("Master/settings", "l2-master-settings"), ("Monitoring", "l2-reporting")], A,
        "Super Admin dapat menjalankan operasi dan review, termasuk self-approval sebagai bypass; status, versi, stok dan audit tetap berlaku.")

    # Posting can fail, finish the parent, or leave a partial balance.
    # All three outcomes come from the posting request; no extra calculation
    # task or second gateway is performed by the application.
    for slug, gate, next_stage, question, full, partial, failed in [
        ("l2-po-gr", "gate3", "stage2", "Hasil posting GR dan PO?", "PO RECEIVED", "PO PARTIALLY_RECEIVED", "GR tidak POSTED"),
        ("l2-delivery", "gate1", "stage0", "Hasil posting delivery dan SO?", "SO DELIVERED", "SO PARTIALLY_DELIVERED", "Delivery tidak POSTED"),
        ("l2-ap", "gate3", "stage2", "Hasil posting payment supplier?", "PO CLOSED", "PO RECEIVED", "Payment gagal"),
        ("l2-ar", "gate3", "stage2", "Hasil posting payment customer?", "SO COMPLETED", "SO DELIVERED", "Payment gagal"),
    ]:
        model = g["MODELS"][slug]
        node = next(n for n in model.nodes if n["key"] == gate)
        node["label"] = question
        obsolete = {"complete", f"check_{gate}_complete"}
        model.nodes = [n for n in model.nodes if n["key"] not in obsolete]
        model.edges = [e for e in model.edges if e["source"] not in obsolete | {gate}
                       and e["target"] not in obsolete]
        model.f(gate, "end", full)
        model.f(gate, next_stage, partial)
        model.f(gate, f"exit{gate[4:]}", failed, True)

    # MatchSupplierInvoice commits either POSTED or MATCH_EXCEPTION. Resolve
    # changes the exception back to MATCH_PENDING before the next match call.
    ap = g["MODELS"]["l2-ap"]
    ap_gate = next(n for n in ap.nodes if n["key"] == "gate1")
    ap_gate["label"] = "Hasil match invoice?"
    ap.nodes = [n for n in ap.nodes if n["key"] not in {"exception", "match_eval", "check_gate1_exception"}]
    ap.edges = [e for e in ap.edges if e["source"] not in {"gate1", "exception", "match_eval", "check_gate1_exception"}
                and e["target"] not in {"exception", "match_eval", "check_gate1_exception"}]
    ap.n("resolve", "Staff perbaiki MATCH_EXCEPTION", M, "callActivity", child="l3-ap-resolve")
    ap.n("resolved", "Hasil perbaikan invoice?", kind="exclusiveGateway")
    ap.f("gate1", "stage2", "POSTED")
    ap.f("gate1", "resolve", "Mismatch")
    ap.f("gate1", "exit1", "Request match gagal", True)
    ap.f("resolve", "resolved")
    ap.f("resolved", "stage1", "MATCH_PENDING")
    ap.f("resolved", "exit1", "Perbaikan gagal", True)

    # A rejected review is a committed business outcome, not a failed request.
    for slug, gate, status in [
        ("l2-pr", "gate2", "PR REJECTED"),
        ("l2-so", "gate2", "SO REJECTED"),
        ("l2-adjustment", "gate2", "Adjustment REJECTED"),
    ]:
        model = g["MODELS"][slug]
        node = next(n for n in model.nodes if n["key"] == gate)
        node["label"] = "Hasil review?"
        model.edges = [e for e in model.edges if e["source"] != gate]
        model.n("rejected", status, kind="endEvent")
        model.f(gate, "rejected", status)
        next_stage = "end" if slug == "l2-pr" else f"stage{int(gate[4:])+1}"
        model.f(gate, next_stage, "APPROVED")
        model.f(gate, f"exit{gate[4:]}", "Review gagal", True)

    # Each detailed review/match has one outcome gateway: accepted, rejected
    # (or mismatch), or invalid request. No invented evaluation request.
    for slug, question, yes, no, yes_label, no_label in [
        ("l3-pr-review", "Hasil review PR?", "Set PR APPROVED", "Set PR REJECTED dan alasan", "Approve", "Reject"),
        ("l3-so-review", "Hasil review SO?", "Set SO APPROVED", "Set SO REJECTED dan alasan", "Approve", "Reject"),
        ("l3-adj-review", "Hasil review adjustment?", "Set adjustment APPROVED", "Set adjustment REJECTED dan alasan", "Approve", "Reject"),
        ("l3-ap-match", "Hasil ExactMatch?", "Set invoice POSTED", "Set invoice MATCH_EXCEPTION", "Cocok", "Mismatch"),
    ]:
        model = g["MODELS"][slug]
        at = next(i for i,n in enumerate(model.nodes) if n["key"] == "effect0")
        model.nodes.pop(at)
        model.edges = [e for e in model.edges if e["source"] != "effect0" and e["target"] != "effect0"
                       and e["source"] not in {"decision", "evaluate"} and e["target"] not in {"decision", "evaluate"}]
        node = next(n for n in model.nodes if n["key"] == "valid")
        node["label"] = question
        model.n("positive", yes)
        model.n("negative", no)
        added = model.nodes[-2:]
        model.nodes = model.nodes[:-2]
        model.nodes[at:at] = added
        model.f("valid", "positive", yes_label)
        model.f("valid", "negative", no_label)
        model.f("positive", "effect1")
        model.f("negative", "effect1")
