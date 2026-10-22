# BPMN Simple ERP — bpm-haqi

Dokumentasi **as-is source code**, snapshot working tree **21 Oktober 2026**. Tujuannya menjelaskan seluruh kapabilitas aplikasi yang terimplementasi, dari gambaran umum sampai kontrol per aksi, tanpa menambahkan fitur ERP yang belum ada.

Mulai dari [katalog diagram](INDEX.md), [landscape L0](preview/l0-simple-erp.svg), lalu [P2P per level](P2P-REVIEW.md) atau [O2C L1](preview/l1-o2c.svg). [Audit kekurangan](AUDIT.md) membedakan perilaku sekarang dari rekomendasi perbaikan.

## Isi dan tingkat detail

| Level | Jumlah | Makna dalam paket ini |
| --- | ---: | --- |
| 0 | 1 | Landscape: akses, P2P, O2C, inventory, finance, administrasi, reporting |
| 1 | 10 | Tujuh domain; P2P/O2C end-to-end; tiga perspektif role |
| 2 | 17 | Lifecycle bisnis, proses pendukung, dan jalur koreksi |
| 3 | 64 | 63 kelompok aksi/API/CLI beserta guard dan efek; satu alur kontrol HTTP bersama |
| Total | 92 | Setiap diagram memiliki `.bpmn` dan preview `.svg` |

Level 0–3 adalah **konvensi dekomposisi dokumentasi ini**, bukan empat level normatif yang diwajibkan standar [OMG BPMN 2.0.2](https://www.omg.org/spec/BPMN/2.0.2/About-BPMN).

Folder `docs/bpmn` lama tidak diubah. Di paket baru ini, **L3 benar-benar diagram BPMN**, bukan hanya matriks pemetaan kode. Dokumen lama yang masih menyebut ADMIN/OPERATOR tidak menjadi sumber otorisasi paket ini.

Diagram P2P memiliki folder khusus `level-2-p2p` dan `level-3-p2p`; salinannya di `level-2`/`level-3` umum **tetap dipertahankan** untuk tautan lama. Katalog memakai folder khusus sebagai lokasi utama, jadi jumlah 92 berarti model unik, bukan jumlah fisik file. [Daftar dan hierarki P2P](P2P-REVIEW.md) menjelaskan pembagiannya. Lima Markdown utama dipertahankan: README, INDEX, P2P-REVIEW, ROLES, dan AUDIT; detail aksi/route tetap tersedia di `manifest.json` dan model BPMN.

## Cara membaca

1. Pilih domain di L0, lalu buka L1 terkait dari [INDEX](INDEX.md).
2. Pada aktivitas bergaris tebal (`callActivity`), buka diagram anak. Pemetaan anak juga tersedia di `manifest.json`.
3. Baca L2 untuk urutan, keputusan, penerimaan/pengiriman/pembayaran parsial, dan mismatch invoice.
4. Baca L3 untuk role, syarat status/versi, perubahan data, serta audit. Dokumentasi elemen XML menyimpan catatan yang lebih rinci daripada label gambar.
5. Cocokkan dengan [ROLES](ROLES.md), [review P2P](P2P-REVIEW.md), [AUDIT](AUDIT.md), serta daftar route dan sumber dalam [manifest](manifest.json).

SVG dapat dibuka langsung di browser; gunakan zoom atau scroll untuk diagram panjang. `.bpmn` memakai BPMN XML + Diagram Interchange sehingga bisa dibuka di modeler yang mendukung BPMN 2.0. Impor lintas file bersifat relatif: pertahankan susunan folder. Tidak semua modeler otomatis membuka diagram anak ketika aktivitas diklik; gunakan katalog bila diperlukan.

## Aturan pemodelan

- Model bersifat deskriptif (`isExecutable=false`), bukan deployment workflow engine.
- Semua diagram berada dalam pool Simple ERP. Lane menunjukkan pelaku manusia atau sistem; label **“atau” berarti alternatif aktor**, bukan dua persetujuan.
- Lane generik pada katalog tingkat atas bukan pemberian izin. Otorisasi aktual mengikuti lane L3 dan matriks role.
- Gateway bertanda X memilih satu jalur. Katalog kapabilitas memilih satu kebutuhan per instansi, bukan mengeksekusi semua modul sekaligus.
- P2P/O2C L1 menampilkan urutan end-to-end. Proses panjang ini terdiri dari beberapa request dan transaksi terpisah, bukan satu transaksi database.
- L2 menguji hasil setelah pemanggilan L3. End event pada L3 hanya berarti request selesai; parent tetap memeriksa status bisnisnya.
- Gateway hasil posting memilih status parent yang ditulis dalam transaksi posting yang sama. Jalur parsial kembali ke pembuatan GR, delivery, atau payment berikutnya.
- Match supplier memiliki tiga hasil: POSTED, MATCH_EXCEPTION yang dapat diperbaiki lalu dicocokkan ulang, atau request gagal. Jalur reject review juga dipisah dari kegagalan request.
- Reject adalah hasil bisnis, berbeda dari request invalid. Copy membuat dokumen baru; reverse/void mempertahankan riwayat. Cancel bukan penghapusan.
- Kondisi gateway berupa deskripsi manusia, bukan ekspresi executable.
- Guard pada gambar diringkas. Middleware global dijelaskan di [kontrol request](preview/l3-kontrol-request.svg); detail sumber ada di `manifest.json` dan XML.
- Error infrastruktur dan rollback tidak diekspansi menjadi boundary event untuk setiap statement SQL. Detail validasi field mengikuti kode, bukan hanya label task.
- Supervisor tidak boleh menyetujui PR/SO/adjustment, issue PO, post GR, match invoice supplier, post payment supplier, post invoice sales, atau post payment customer buatannya sendiri. Super Admin boleh self-approval sebagai bypass; guard status, versi, stok, dan audit tetap berlaku.
- Supplier/customer/bank tidak digambar sebagai proses eksternal otomatis: tidak ada bukti integrasi tersebut. Pencatatan pembayaran bukan transfer bank.

## Batas cakupan dan bukti

“Lengkap” di sini berarti seluruh **89 endpoint terdaftar**, bootstrap CLI, tujuh domain, tiga role, dan lifecycle/koreksi yang ditemukan pada kode snapshot. Ini bukan klaim mencakup seluruh kemungkinan fitur ERP atau semua kombinasi error, race condition, dan konfigurasi deployment.

Sumber utama: registrasi route, middleware, handler/application/store, migration/constraint, serta komponen UI. [Manifest](manifest.json) menyimpan SHA256 sumber untuk mendeteksi perubahan setelah snapshot.

Dokumen menggambarkan kode RBAC tiga role yang ada di working tree. Migration `000010_three_roles` **tidak dijalankan dalam pekerjaan dokumentasi ini**. Database, Docker image, akun aktif, dan UI runtime belum diverifikasi; dokumentasi tidak membuktikan runtime sudah memakai role baru.

## Verifikasi dan regenerasi

Jalankan dari root repository, Python 3 dengan `lxml`:

```powershell
python docs/bpm-haqi/tools/build.py
python docs/bpm-haqi/tools/validate.py --require-schema
node docs/bpm-haqi/tools/render.cjs
```

Lima XSD resmi OMG disertakan di `tools/schema`; sumber unduhan dicatat di [schema README](tools/schema/README.md). Untuk menyegarkannya secara eksplisit:

```powershell
python docs/bpm-haqi/tools/validate.py --fetch-schema --require-schema
```

`build.py` hanya menulis paket dokumentasi ini. Ubah katalog setelah meninjau kode, bukan sekadar regenerate untuk menutupi drift. Validasi memeriksa XSD, referensi flow, lane, keterjangkauan start/end, DI, overlap node, link antarfile, hierarki, cakupan route, dan hash sumber. Pemetaan action wildcard dan route inline memiliki daftar eksplisit di validator; tinjau daftar tersebut bila routing berubah.

Hasil tersimpan di [validation.json](validation.json) dan [render-validation.json](qa/render-validation.json). Render memakai Playwright di `frontend/node_modules`; screenshot visual QA di folder `qa` dibuat lokal dan tidak di-commit. Jika browser default Playwright belum terpasang, arahkan environment variable `BPMN_CHROMIUM_PATH` ke executable Chromium yang sudah tersedia. Validasi statis dan render bukan pengganti uji transaksi pada database terisolasi.
