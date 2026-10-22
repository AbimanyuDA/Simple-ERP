# Audit keseluruhan — temuan dan prioritas

Snapshot source dan deployment lokal **2026-10-21**. Audit ini membaca alur frontend/backend, otorisasi, lifecycle, constraint/migration, stok, finance, dan reporting. Migrasi serta tes integrasi lulus pada database uji. Backup database utama dibuat, migrasi data lama 9 → 12 berhasil, image Docker backend/frontend baru berjalan, dan smoke browser login/navigasi/Finance lulus. Belum ada pentest atau bukti seluruh alur bisnis pada database utama.

Pekerjaan lanjutan P2P mengubah backend, UI, tes, dan paket `docs/bpm-haqi`. Temuan lain di bawah belum otomatis diperbaiki; perubahan aplikasi lain yang sudah ada di working tree tetap dipertahankan.

## Ringkasan

Fondasi aplikasi sudah memiliki transaksi database, status guard, version checks pada banyak aksi, review PR/SO/adjustment yang terpisah dari creator, audit trail, immutable stock movement, dan jalur reversal/void. Jangan menambah role atau approval engine generik dulu.

Prioritas praktis: undang satu Supervisor untuk uji alur tiga akun pada aplikasi utama; database hasil migrasi hanya memiliki Super Admin dan Staff. Perbaikan sesi, create product, aktivasi user, recovery PO, recovery adjustment, serta maker-checker AP/AR lulus tes integrasi pada database uji. Browser suite 12/12 lulus pada stack uji terpisah; smoke read-only juga lulus pada Docker utama.

P1 = perlu dibereskan sebelum mengandalkan kontrol tersebut; P2 = inkonsistensi/recovery/usability penting; P3 = penguatan selanjutnya. Tidak ada klaim sistem bebas P0: audit statis bukan bukti keamanan menyeluruh.

## P1 — kontrol penting

### A01. Batas absolut refresh session — diperbaiki di source

- Bukti: [Rotate](../../backend/internal/auth/adapter/postgres/session.go) kini menyalin deadline absolut dari sesi lama dan membatasi access/idle expiry dengan deadline tersebut.
- Status: diperbaiki di source; tes regresi lulus pada database uji terisolasi.
- BPMN: `l3-refresh` perlu dibaca bersama guard deadline absolut yang sekarang ditegakkan; validasi diagram tidak menggantikan tes sesi.

### A02. Bypass Super Admin perlu pemantauan audit — kebijakan yang disengaja

- Bukti: [UpdateOrder dan TransitionOrder](../../backend/internal/sales/adapter/postgres/store.go). Supervisor tidak dapat mengedit draft melalui handler dan tidak boleh review SO buatannya sendiri. Super Admin boleh mengedit maupun self-approve sebagai bypass yang diminta.
- Dampak: bypass adalah kebijakan akses tinggi, bukan lagi pelanggaran maker-checker. Audit sudah mencatat actor dan transisi, tetapi belum ada penanda/alasan khusus bahwa tindakan itu menggunakan bypass.
- Penguatan sederhana: pantau event approval oleh Super Admin; bila perlu, tambah alasan bypass dan tampilkan jelas di audit. Jangan memblokir self-approval Super Admin tanpa mengubah kebijakan role.

### A03. Create product tanpa baris — diperbaiki di source

- Bukti: [master Store.Create](../../backend/internal/masterdata/adapter/postgres/store.go) kini mensyaratkan unit PCS aktif dan tepat satu baris produk terinsert sebelum audit/commit.
- Status: diperbaiki di source; tes tanpa PCS aktif lulus pada database uji terisolasi.

### A04. AR/O2C maker-checker — diperbaiki di source

- P2P/AP **sudah diubah di source**: Staff membuat invoice/payment supplier; Supervisor match/post; [route](../../backend/cmd/api/main.go) dan [finance Store](../../backend/internal/finance/adapter/postgres/store.go) menolak Supervisor memproses dokumennya sendiri, tetapi mengizinkan bypass Super Admin aktif.
- Sales invoice sekarang dibuat `DRAFT` oleh Staff lalu di-post oleh Supervisor; customer payment mengikuti pola yang sama. Post menolak actor pembuat, kecuali Super Admin aktif. Draft invoice tidak dapat digunakan untuk membuat payment.
- Migration `000012` dan tes integrasi O2C lulus pada database uji terisolasi. Migrasi data lama dan smoke browser utama lulus, tetapi transaksi O2C tiga akun belum diuji pada database utama karena akun Supervisor belum tersedia.

## P2 — konsistensi alur, UI, dan pemulihan

| ID | Temuan dan bukti | Dampak / tindakan yang disarankan |
| --- | --- | --- |
| A05 | [SetUserStatus](../../backend/internal/auth/adapter/postgres/users.go) kini hanya menerima reaktivasi dari DEACTIVATED yang memiliki password | Diperbaiki di source; tes undangan tanpa password lulus pada database uji terisolasi |
| A06 | [DashboardWorkspace](../../frontend/features/dashboard/dashboard-workspace.tsx) dahulu menampilkan semua tab reports/audit dan shortcut create | **Diperbaiki di source**: tab laporan/audit hanya reviewer; shortcut create hanya Staff/Super Admin. Browser utama sudah diuji untuk login/navigasi dua akun, belum Supervisor |
| A07 | [Summary backend](../../backend/internal/reporting/adapter/postgres/store.go) diakses semua role dan mengembalikan nilai AP/AR/overdue | Tentukan apakah Staff boleh melihat angka keuangan. Jika tidak, batasi **response backend**, bukan hanya menyembunyikan card UI |
| A08 | [AppShell](../../frontend/components/app-shell.tsx) kini menyediakan logout eksplisit; [Users UI](../../frontend/features/users/users-panel.tsx) masih terutama list/invite | Logout diperbaiki di source dan dites di unit UI; aksi reset/deactivate/revoke yang penting masih perlu UI tanpa membangun token-management rumit sekaligus |
| A09 | [CancelOrder](../../backend/internal/purchasing/adapter/postgres/store.go) dan migration `000011` kini membatalkan PO, memulihkan PR ke APPROVED, serta membatasi satu PO aktif per PR | Migrasi dan skenario PO pengganti lulus pada database uji terisolasi; migrasi database utama juga berhasil, tetapi transaksi pengganti tidak dijalankan pada data pengguna |
| A10 | [Adjustment Store](../../backend/internal/inventory/adapter/postgres/store.go) tetap menolak snapshot stok lama saat approve/post | APPROVED stale kini bisa dibatalkan creator dengan alasan, lalu dibuat ulang dan dimintakan approval lagi; tes integrasi recovery lulus pada database uji |
| A11 | Owner checks bervariasi pada [Purchasing](../../backend/internal/purchasing/adapter/postgres/store.go), [Sales](../../backend/internal/sales/adapter/postgres/store.go), [Inventory](../../backend/internal/inventory/adapter/postgres/store.go) | Cancel PR/SO/adjustment dibatasi creator/owner, tetapi beberapa submit/edit/cancel turunan tidak. Dokumentasikan apakah Staff bekerja secara tim atau hanya milik sendiri, lalu samakan guard sesuai pilihan |
| A12 | [API client](../../frontend/lib/api.ts) membuat idempotency key per pemanggilan; [middleware](../../backend/internal/platform/idempotency/middleware.go) hanya meliputi group operasi | Dua percobaan create dengan key berbeda bukan replay. Pertahankan key per intent submit/retry; tambah disable in-flight. Pisahkan kebutuhan idempotency untuk auth/master sesuai risikonya |
| A13 | [Settings Store](../../backend/internal/masterdata/adapter/postgres/settings.go) tidak memvalidasi semua semantik currency/timezone/warehouse aktif | Validasi input berdasarkan nilai yang didukung; jangan menerima timezone sembarang atau default warehouse nonaktif hanya karena FK valid |
| A14 | Tidak ada endpoint perubahan role existing user; migration [000010](../../backend/migrations/000010_three_roles.up.sql) memilih Super Admin dari ADMIN lama | Perlu pemeriksaan akun hasil migrasi dan prosedur perubahan role yang diaudit. Jangan menambahkan edit role bebas tanpa proteksi owner terakhir, sesi aktif, dan akses diri |

## P3 — penguatan yang tetap sederhana

- **Pagination dan pencarian server-side.** [Reporting Store](../../backend/internal/reporting/adapter/postgres/store.go) membatasi hasil; audit UI hanya mencari data yang sudah dimuat. Tanpa pagination, hasil bukan arsip lengkap. Prioritaskan audit dan daftar transaksi besar.
- **Readiness terpisah dari liveness.** [Healthz](../../backend/internal/platform/httpserver/router.go) hanya merespons ok; jangan memaknainya sebagai database sehat. Tambah readiness DB bila deployment memerlukannya.
- **Uji retry setelah crash.** Klaim/penyimpanan hasil idempotency terpisah dari transaksi bisnis. Uji crash di antara commit bisnis dan penyimpanan respons sebelum mengklaim exactly-once.
- **Master data minimum.** Produk memakai unit PCS default; belum ada pilihan unit pada create, edit nama, atau penghapusan. Prioritaskan edit nama/unit yang aman jika memang dibutuhkan; jangan menghapus master yang sudah direferensikan.
- **Pesan konflik yang dapat ditindaklanjuti.** Banyak invalid-state/version/stock guards menjadi conflict. UI sebaiknya membedakan data berubah, saldo kurang, creator mencoba review sendiri, dan dokumen belum pada tahap yang benar.

## Belum tersedia — bukan otomatis bug

Tidak ada bukti implementasi general ledger, jurnal debit/kredit, pajak, costing persediaan, multi-currency conversion, warehouse transfer, retur bisnis khusus, partial invoicing sebelum fulfillment penuh, integrasi bank, email invitation/PO otomatis, ataupun workflow approval bertingkat.

Untuk ERP sederhana, jangan menambah semuanya. Stock reversal bukan modul retur; payment recording bukan transfer bank; AP/AR bukan buku besar. BPMN baru tidak menambahkan proses fiktif untuk fitur-fitur tersebut.

## Verifikasi yang harus dilakukan sebelum dipakai operasional

Jalankan pada database, akun, dan base URL uji yang terisolasi. Kolom hasil sengaja kosong: ini **rencana uji**, bukan klaim PASS.

| Kasus | Hasil yang diharapkan | PASS/FAIL |
| --- | --- | --- |
| Migration akun ADMIN/OPERATOR lama | Tiga role baru; pemilik Super Admin sesuai keputusan; login valid | |
| Staff akses endpoint approval, AP/AR, users, settings | Approval/post/void/admin ditolak; AP/AR read/create diterima sesuai aksi | |
| Supervisor mencoba create PR/SO/adjustment | Ditolak; read dan review milik orang lain tetap bisa | |
| Creator Super Admin approve/reject PR/SO/adjustment sendiri | Diterima bila status/versi/syarat bisnis valid dan tercatat audit | |
| Super Admin edit SO milik Staff lalu approve | Diizinkan sebagai bypass, status/versi valid, actor tercatat di audit | |
| P2P: Staff PR/PO/dua GR/invoice/dua payment; Supervisor review/issue/post/match/post | Self-approval ditolak; stok, outstanding, PO CLOSED, PR COMPLETED, audit konsisten | |
| O2C: SO → review → confirm → dua delivery → invoice → pelunasan | Stok tidak negatif; SO COMPLETED; invoice PAID | |
| Dua delivery bersamaan memakai saldo yang sama | Tidak terjadi oversell; konflik ditangani | |
| Supplier invoice mismatch → resolve → match ulang | MATCH_EXCEPTION → MATCH_PENDING → POSTED | |
| Payment void → invoice void → GR/delivery reverse | Outstanding, parent state, stok dan riwayat dipulihkan sesuai guard | |
| Adjustment stale sebelum approve dan sesudah approve | Posting tidak memakai snapshot lama; jalur recovery jelas | |
| Same idempotency key/body; lalu same key/body berbeda | Replay hasil pertama; konflik untuk payload berbeda | |
| Refresh mendekati absolute expiry, lalu lewat deadline | Setelah A01 diperbaiki, family tetap kedaluwarsa di deadline awal | |
| Create product tanpa unit PCS | Setelah A03 diperbaiki, ditolak jelas atau unit valid dipilih; tidak sukses semu | |
| Invite → activate, reset, revoke, deactivate; admin activate INVITED | Token one-time, sesi dicabut sesuai aksi, transisi invalid ditolak setelah A05 diperbaiki | |
| Setiap role membuka seluruh menu Dashboard | Tidak ada shortcut/tab yang menjanjikan aksi terlarang | |

## Urutan pengerjaan yang disarankan

1. Undang dan aktifkan Supervisor pada aplikasi utama, lalu jalankan acceptance P2P/O2C/adjustment tiga akun. Migrasi data lama dan deployment Docker sudah diverifikasi secara terbatas.
2. Putuskan visibilitas angka keuangan Staff dan apakah dokumen draft boleh dikerjakan Staff lain; samakan guard bila kebijakannya owner-only.
3. Lanjutkan penguatan UI pengelolaan akun, retry/idempotency, dan pesan konflik sesuai kebutuhan demo; jangan menyebut seluruh kombinasi error telah diuji.

Sumber aksi/route ada di [manifest](manifest.json); bukti cakupan dan tautan ada di [validation.json](validation.json). Kelulusan validasi BPMN tidak menyatakan temuan aplikasi di atas sudah selesai.
