# Review khusus Procure to Pay (P2P)

Status: **cocok untuk alur utama di source dan lulus tes integrasi terisolasi**; Docker utama sudah diperbarui, tetapi transaksi P2P tiga akun belum diuji di data utama karena belum ada Supervisor. Review ini dibatasi ke PR, PO, goods receipt (GR), invoice supplier, payment supplier, dan koreksinya. O2C, adjustment, administrasi user, dan reporting adalah paket review terpisah.

Pembagian kerja: bagian yang saya tangani sampai tingkat source/UI adalah **P2P** (diagram L1 P2P, L2 PR/PO-GR/AP/koreksi P2P, dan L3 aksi terkait). Bagian yang dapat dilanjutkan teman adalah **O2C** (SO/delivery/AR dan koreksi), lalu inventory adjustment, akses/user, master/configuration, reporting, dan kontrol platform. Jangan menganggap validasi otomatis seluruh paket sebagai review semantik mendalam modul-modul lanjutan itu.

## Folder dan turunan level P2P

Folder `level-2-p2p` dan `level-3-p2p` berisi **diagram khusus P2P** dan menjadi lokasi utama pada katalog. Salinan P2P di folder `level-2`/`level-3` umum tetap ada untuk tautan lama, bersama modul lain dan kontrol lintas domain. Preview SVG terpusat di `preview`. Satu instansi L1 menggambarkan beberapa request/commit, bukan satu transaksi database panjang.

| Level | Diagram P2P | Turunan |
| --- | --- | --- |
| L1 | [P2P end-to-end](level-1/l1-p2p.bpmn) | L2 PR, PO/GR, AP, dan koreksi P2P |
| L2 utama | [PR](level-2-p2p/l2-pr.bpmn) | L3 `pr-create`, `pr-submit`, `pr-review` |
| L2 utama | [PO/GR](level-2-p2p/l2-po-gr.bpmn) | L3 `po-convert`, `po-issue`, `gr-create`, `gr-post` |
| L2 utama | [AP](level-2-p2p/l2-ap.bpmn) | L3 `ap-invoice`, `ap-match`, `ap-resolve`, `ap-payment`, `ap-payment-post` |
| L2 alternatif | [Koreksi P2P](level-2-p2p/l2-p2p-correction.bpmn) | [Koreksi Purchasing](level-2-p2p/l2-purchase-correction.bpmn) + L3 `ap-resolve`, `ap-match`, `ap-payment-void`, `ap-invoice-void` |
| L2 pendukung | [Koreksi Purchasing](level-2-p2p/l2-purchase-correction.bpmn) | L3 `pr-cancel`, `pr-copy`, `po-cancel`, `gr-cancel`, `gr-reverse` |

Total khusus P2P: **1 L1, 4 L2 langsung + 1 L2 pendukung, 19 L3 unik**. `ap-match` dan `ap-resolve` dipakai ulang, tidak diduplikasi. [L1 Finance](level-1/l1-finance.bpmn) dan [L2 koreksi Finance](level-2/l2-finance-correction.bpmn) adalah katalog gabungan AP/AR, bukan P2P khusus. Diagram L0, role, dan [kontrol HTTP](level-3/l3-kontrol-request.bpmn) bersifat lintas domain. Tautan L3 lengkap ada pada alur dan koreksi di bawah.

Pada dua L2 koreksi, lima pilihan sekarang bertemu di satu **XOR merge → satu end event** karena hasil akhirnya sama. Pilihan koreksi dan pilihan proses baru/koreksi di L1 harus eksplisit; tidak ada default yang diam-diam menjalankan aksi terakhir. Pada L1/L2 lifecycle, end event setelah gateway tetap terpisah jika status berhentinya berbeda (misalnya PR belum disetujui versus PO belum diterima); labelnya dibuat spesifik. Gateway → end untuk cabang gagal/berhenti memang sah, bukan tugas tambahan yang harus dipaksakan. Preview SVG kini menampilkan tanda slash pada alur default dan tanda `+` pada call activity. Ini mengikuti [spesifikasi OMG BPMN 2.0.2](https://www.omg.org/spec/BPMN/2.0.2/PDF) tentang gateway, end event, default flow, dan collapsed call activity. Seluruh 25 model P2P diperiksa secara struktural; `conditionExpression` adalah deskripsi manusia dan proses ditandai `isExecutable=false`, jadi paket ini tidak boleh dipakai sebagai workflow engine tanpa pendefinisian data dan ekspresi yang dapat dieksekusi.

## Peran bisnis dan batas kode saat ini

| Bagian | Staff | Supervisor | Kondisi sebenarnya |
| --- | --- | --- | --- |
| PR | Buat, submit, cancel draft milik sendiri, copy rejected | Approve/reject PR bukan buatannya | Super Admin boleh self-review sebagai bypass; review tetap mencatat actor dan status. |
| PO dan GR | Konversi PR approved menjadi PO draft; buat/cancel GR draft | Issue/cancel PO; post/reverse GR | Supervisor tidak boleh issue/post dokumennya sendiri; Super Admin boleh bypass. Pembatalan PO adalah keputusan korektif Supervisor. |
| Invoice dan payment supplier | Buat invoice, resolve mismatch, buat draft payment | Match/void invoice; post/void payment | Supervisor tidak boleh match/post dokumennya sendiri; Super Admin boleh bypass. Kedua role dapat membaca AP; hanya reviewer dapat membaca AR. |
| Super Admin | — | — | Di luar *perspektif pelaku bisnis* harian P2P. [Guard role](../../backend/internal/auth/domain/role.go) memberi akses darurat ke kedua sisi, termasuk self-approval. |

Super Admin mengelola akun dan dapat bypass pembatasan role maupun self-approval saat diperlukan; ia bukan lane proses normal. Guard status/versi/stok dan audit tetap berlaku. Diagram L3 mendokumentasikan hak akses *as-is*.

## Cocokkan alur utama dengan kode

| Urutan | Diagram rinci | Efek status dan keputusan yang benar-benar ada | Bukti source |
| --- | --- | --- | --- |
| 1. Staff membuat PR | [create](level-3-p2p/l3-pr-create.bpmn) | PR `DRAFT`; purpose dan lines divalidasi. | [CreateRequest](../../backend/internal/purchasing/adapter/postgres/store.go) |
| 2. Staff submit | [submit](level-3-p2p/l3-pr-submit.bpmn) | `DRAFT -> SUBMITTED`, dengan versi cocok. | [TransitionRequest](../../backend/internal/purchasing/adapter/postgres/store.go) |
| 3. Supervisor review | [review](level-3-p2p/l3-pr-review.bpmn) | `SUBMITTED -> APPROVED` atau `REJECTED`; Supervisor bukan creator. Super Admin boleh bypass self-review. Reject memerlukan alasan. Request gagal bukan status `REJECTED`. | [TransitionRequest](../../backend/internal/purchasing/adapter/postgres/store.go) |
| 4. Staff konversi PO | [convert](level-3-p2p/l3-po-convert.bpmn) | PR `APPROVED -> CONVERTED_TO_PO`; PO baru `DRAFT`, supplier dan warehouse harus aktif. | [ConvertToOrder](../../backend/internal/purchasing/adapter/postgres/store.go) |
| 5. Supervisor issue PO | [issue](level-3-p2p/l3-po-issue.bpmn) | PO `DRAFT -> ISSUED`; Supervisor bukan creator PO, Super Admin boleh bypass. Tidak ada pengiriman PO otomatis ke supplier. | [IssueOrder](../../backend/internal/purchasing/adapter/postgres/store.go) |
| 6. Staff buat GR; Supervisor post | [create](level-3-p2p/l3-gr-create.bpmn), [post](level-3-p2p/l3-gr-post.bpmn) | Dalam alur normal actor berbeda; Super Admin boleh bypass. GR `DRAFT -> POSTED`; stok dan stock movement bertambah. PO menjadi `PARTIALLY_RECEIVED` atau `RECEIVED`. Bila parsial, buat GR berikutnya; invoice belum boleh dibuat. | [CreateReceipt/PostReceipt](../../backend/internal/purchasing/adapter/postgres/store.go) |
| 7. Staff catat invoice | [create](level-3-p2p/l3-ap-invoice.bpmn) | Hanya dari PO `RECEIVED`; invoice menjadi `MATCH_PENDING`. Bukan tiga-way match GR: fungsi match membandingkan invoice dengan line PO setelah prasyarat penerimaan penuh. | [CreateSupplierInvoice](../../backend/internal/finance/adapter/postgres/store.go) |
| 8. Supervisor match | [match](level-3-p2p/l3-ap-match.bpmn) | Supervisor bukan creator; Super Admin boleh bypass. Exact match menjadi `POSTED`; mismatch menjadi `MATCH_EXCEPTION` dan menyimpan alasan. Staff [resolve](level-3-p2p/l3-ap-resolve.bpmn) ke `MATCH_PENDING`, lalu Supervisor match ulang. | [MatchSupplierInvoice/ResolveSupplierInvoice](../../backend/internal/finance/adapter/postgres/store.go) |
| 9. Staff catat payment; Supervisor post | [create](level-3-p2p/l3-ap-payment.bpmn), [post](level-3-p2p/l3-ap-payment-post.bpmn) | Dalam alur normal actor berbeda; Super Admin boleh bypass. Payment `DRAFT -> POSTED`; invoice `PARTIALLY_PAID` atau `PAID`. Payment berikutnya menangani sisa. Bila **semua invoice non-VOID** pada PO lunas, PO `RECEIVED -> CLOSED` dan PR `CONVERTED_TO_PO -> COMPLETED`. | [CreateSupplierPayment/PostSupplierPayment](../../backend/internal/finance/adapter/postgres/store.go) |

Payment API dapat mengalokasikan satu payment ke beberapa invoice milik supplier yang sama; UI saat ini membuat **satu alokasi per payment**. Diagram L2 menggambarkan satu lintasan PO sampai lunas, bukan membatasi kemampuan API untuk alokasi lintas invoice/PO.

## Jalur alternatif dan koreksi

- PR `DRAFT` dapat [cancel](level-3-p2p/l3-pr-cancel.bpmn). PR `REJECTED` tidak di-submit ulang: [copy](level-3-p2p/l3-pr-copy.bpmn) membuat PR `DRAFT` **baru**.
- PO `DRAFT/ISSUED` dapat [cancel](level-3-p2p/l3-po-cancel.bpmn) oleh Supervisor bila belum ada GR `POSTED`. Dalam transaksi yang sama PR asal kembali `APPROVED`; PO batal tetap historis, dan Staff dapat membuat satu PO pengganti aktif setelah migration `000011`.
- GR `DRAFT` dapat [cancel](level-3-p2p/l3-gr-cancel.bpmn). GR `POSTED` dapat [reverse](level-3-p2p/l3-gr-reverse.bpmn) oleh Supervisor jika tidak ada invoice non-VOID dan stok akhir tidak negatif; movement lama tidak dihapus.
- Invoice `POSTED` dapat [void](level-3-p2p/l3-ap-invoice-void.bpmn) jika tidak ada alokasi dari payment `POSTED`. Bila sudah dibayar, [void payment](level-3-p2p/l3-ap-payment-void.bpmn) dahulu. Void payment memulihkan outstanding dan bisa membuka PO `CLOSED -> RECEIVED` serta PR `COMPLETED -> CONVERTED_TO_PO`.
- Setelah invoice non-VOID tidak ada, reversal GR dapat diperiksa ulang. Tiap operasi tetap punya guard status dan alasan; ini bukan tombol “undo seluruh P2P”.

## Kesesuaian web dan gap yang tersisa

- [UI PR](../../frontend/features/purchasing/purchase-request-panel.tsx), [PO](../../frontend/features/purchasing/purchase-order-panel.tsx), [GR](../../frontend/features/purchasing/goods-receipt-panel.tsx), dan [Finance](../../frontend/features/finance/finance-workspace.tsx) menyediakan alur utama. Backend adalah otoritas final untuk role/status/versi.
- UI PR mengisi alasan reject/cancel dengan teks demo tetap, bukan alasan yang diketik pengguna. Ini tidak menggagalkan guard backend, tetapi mengurangi kegunaan audit.
- UI invoice menampilkan sumber berdasarkan belum adanya invoice non-VOID, tanpa selalu menyaring status PO `RECEIVED`; pilihan PO yang belum siap akhirnya ditolak backend. UI payment hanya satu invoice per draft, sementara API mendukung beberapa alokasi.
- Staff membuat draft AP; Supervisor match/post tanpa dapat membuat draft AP. Backend menolak Supervisor memproses dokumen buatannya sendiri; Super Admin aktif dikecualikan.
- Super Admin tetap memiliki bypass role untuk keadaan khusus, tetapi bukan pelaku P2P harian. Audit penggunaan bypass belum dibuktikan melalui uji runtime.
- Pemulihan PO cancel sudah ada di source dan diagram; migration `000011` dan skenario replacement PO lulus pada database uji terisolasi. Database aplikasi utama sudah dimigrasi ke versi 12, tetapi skenario replacement tidak dijalankan pada data pengguna.

## Bukti dan batas “100%”

Struktur P2P, referensi child, gateway, arah panah, label, dan status kritis diperiksa oleh [validator](validation.json) serta [render QA](qa/render-validation.json). Review source di atas memeriksa fungsi P2P, route/role backend, dan kontrol UI. Ini **100% cakupan model P2P yang terdaftar**, bukan jaminan 100% perilaku runtime.

Tes unit tanpa database `go test ./internal/... ./cmd/...` dari folder `backend` lulus, termasuk guard role HTTP. Migrasi database uji terpisah berhasil dan `go test -count=1 -tags=integration ./test/integration` lulus, termasuk P2P hingga pelunasan, pemulihan PO, dan regresi sesi/product/undangan. Validator dan render Chromium lulus 93/93 diagram, termasuk penambahan invoice-post O2C.

Migrasi data akun lama dan smoke browser login/navigasi/Finance dua akun pada Docker utama sudah lulus. Belum ada Supervisor untuk uji browser tiga akun pada database utama, dan skenario retry/crash belum dilakukan. [Integration test](../../backend/test/integration/operations_test.go) memakai `TRUNCATE` pada tabel bisnis; jangan jalankan ke database pengguna. Tes integrasi membuktikan alur transaksi terhadap database uji, bukan seluruh perilaku UI pada data utama.
