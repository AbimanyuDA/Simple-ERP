# Role dan batas kewenangan

As-is working tree 2026-10-21. Role backend: `STAFF`, `SUPERVISOR`, `SUPER_ADMIN`. Ini bukan konfigurasi role dinamis.

| Kapabilitas | Staff | Supervisor | Super Admin |
| --- | --- | --- | --- |
| Login, refresh, logout; baca sesi | Ya | Ya | Ya |
| Baca PR, PO, GR, SO, delivery, adjustment, saldo/movement, master | Ya | Ya | Ya |
| Buat/submit PR | Ya | Tidak | Ya |
| Konversi approved PR ke PO draft; buat/cancel GR draft | Ya | Tidak | Ya |
| Issue/cancel PO; post GR | Tidak | Ya, bukan creator untuk issue/post | Ya, termasuk dokumen sendiri |
| Buat/edit/cek stok/submit/confirm SO | Ya | Tidak | Ya |
| Buat/post/cancel delivery draft | Ya | Tidak | Ya |
| Buat/submit/post adjustment | Ya | Tidak | Ya |
| Cancel PR/SO/adjustment sesuai status | Hanya creator | Tidak | Ya |
| Copy PR/SO REJECTED | Ya | Tidak | Ya |
| Approve/reject PR, SO, adjustment | Tidak | Ya, bukan creator | Ya, termasuk dokumen sendiri |
| Reverse GR, delivery, adjustment | Tidak | Ya | Ya |
| Buat/aktif/nonaktifkan master | Tidak | Ya | Ya |
| Baca AP; buat invoice supplier, resolve mismatch, buat payment supplier | Ya | Baca saja | Ya |
| Match/void invoice supplier; post/void payment supplier | Tidak | Ya, bukan creator untuk match/post | Ya, termasuk dokumen sendiri |
| Baca AR; buat draft invoice sales dan payment customer | Ya | Baca saja | Ya |
| Post/void invoice sales dan payment customer | Tidak | Ya, bukan creator untuk post | Ya, termasuk dokumen sendiri |
| Dashboard summary termasuk nilai AP/AR | Ya | Ya | Ya |
| Laporan dan audit log | Tidak | Ya | Ya |
| Users: baca, invite, status, reset token, revoke sesi/token, mark-shared | Tidak | Tidak | Ya |
| Baca/ubah organization settings | Tidak | Tidak | Ya |
| Ubah role user yang sudah ada | Tidak tersedia | Tidak tersedia | Tidak tersedia |

Guard status, saldo, alasan, versi, dan relasi dokumen tetap berlaku walaupun kolom bertuliskan “Ya”. Kebijakan owner tidak seragam: misalnya edit SO, submit, cancel PO/GR/delivery tidak semuanya memeriksa creator. Jangan menyimpulkan seluruh dokumen hanya bisa diolah pembuatnya.

Approval/rejection PR, SO, adjustment serta issue PO, post GR, match invoice supplier, post payment supplier, post invoice sales, dan post payment customer membandingkan actor dengan `creator_id` untuk Supervisor. Super Admin aktif dapat bypass pemeriksaan pembuat melalui role yang diverifikasi di database; status, versi, stok, dan audit tetap berlaku. Ini **bukan** pemeriksaan semua orang yang pernah mengubah dokumen.

## Perspektif BPMN

- [Staff](preview/role-staff.svg): fokus pekerjaan operasional utama; aksi koreksi lengkap tetap ada di L2/L3.
- [Supervisor](preview/role-supervisor.svg): review/commit P2P dan O2C, reversal, master, laporan; tidak membuat draft AP/AR.
- [Super Admin](preview/role-super-admin.svg): pengelola akun dan bypass operasional termasuk self-approval; bukan lane transaksi harian.

Penerima token undangan/reset belum login dan bukan role keempat. Pengelola bootstrap CLI adalah pelaku instalasi, juga bukan role aplikasi tambahan.

## API berbeda dari UI

| Area | Backend | UI snapshot |
| --- | --- | --- |
| User admin | Invite, status, reset, revoke, mark-shared | Panel users baru list dan invite; aksi admin lainnya belum memiliki kontrol lengkap |
| Logout | Endpoint tersedia | Tidak ada tombol logout eksplisit di AppShell |
| Dashboard | Summary semua role; reports/audit dibatasi reviewer | Tab reports/audit dan shortcut create belum difilter konsisten berdasarkan role |
| Role assignment | Saat invite; migration untuk akun lama | Bukan layar pengubahan role existing user |

Sumber: [role domain](../../backend/internal/auth/domain/role.go), [route/middleware assembly](../../backend/cmd/api/main.go), daftar aksi/route dalam [manifest](manifest.json), [users UI](../../frontend/features/users/users-panel.tsx), [dashboard UI](../../frontend/features/dashboard/dashboard-workspace.tsx).

## Migrasi akun lama

Migration [000010](../../backend/migrations/000010_three_roles.up.sql) memetakan OPERATOR ke STAFF. ADMIN menjadi SUPERVISOR, kecuali satu akun ADMIN yang dipilih menjadi SUPER_ADMIN: prioritaskan ADMIN aktif paling awal; fallback ADMIN paling awal. Bootstrap instalasi kosong membuat SUPER_ADMIN.

Sebelum deployment, tinjau siapa akun yang akan menjadi Super Admin. Jika database tidak memiliki ADMIN sama sekali, jangan berasumsi migration menciptakan pemilik baru. Pekerjaan ini tidak menjalankan migration atau mengubah akun.
