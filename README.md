# Mini ERP UMKM

Mini ERP berbasis BPMN untuk UMKM. Repository ini sedang disiapkan sebagai baseline statis penelitian impact mapping. BPMN berfungsi sebagai process specification dan sumber traceability; aplikasi tidak membaca BPMN saat runtime.

## Status

- Arsitektur: modular monolith dengan Clean Architecture per domain
- BPMN: `docs/bpmn/` (baseline awal) dan `docs/bpm-haqi/` (model Level 0–3 yang lebih rinci)
- Traceability: `docs/PROCESS-MAPPING.md`
- Struktur source: `frontend/` untuk Next.js dan `backend/` untuk Go modular monolith
- Status verifikasi: tes otomatis tersedia; penerimaan alur bisnis tiga role masih memerlukan uji manual. Checklist berisi akun demo disimpan lokal dan tidak di-commit.

## Dokumen utama

- `docs/bpm-haqi/README.md` — panduan BPMN Level 0–3; `docs/bpmn/` tetap menyimpan baseline awal
- `docs/RBAC-AUDIT.md` — hak akses Staff, Supervisor, Super Admin saat ini
- `docs/PROCESS-MAPPING.md` — mapping BPMN baseline ke software
- `docs/BUSINESS-RULES.md`, `docs/STATE-MACHINES.md`, `docs/ERD.md`, `docs/API-CONTRACT.md`, `docs/PHASE-1-DESIGN.md` — spesifikasi desain awal; beberapa bagian historis dan bukan bukti runtime terbaru

## Menjalankan secara lokal dengan Compose

Prasyarat: Docker/Compose. Port default yang digunakan adalah `3000`, `8080`, dan `5432`. Jika bentrok, atur `FRONTEND_HOST_PORT`, `BACKEND_HOST_PORT`, atau `POSTGRES_HOST_PORT` di `.env` tanpa mengubah port internal container.

1. Salin `.env.example` menjadi `.env`. Isi `POSTGRES_PASSWORD` dan `SESSION_PEPPER` dengan nilai lokal yang unik; jangan commit `.env`. `DATABASE_URL` hanya dipakai oleh command backend native dan password-nya harus sama dengan `POSTGRES_PASSWORD`.
2. Jalankan `docker compose up -d --build`. Service `migrate` menunggu PostgreSQL sehat dan menjalankan seluruh migration sebelum backend dimulai.
3. Khusus database baru, buat Super Admin pertama secara manual dan interaktif: `docker compose run --rm --entrypoint bootstrap-admin backend --email <email-admin>`. Tidak ada credential default dan bootstrap akan menolak database yang sudah memiliki user.
4. Buka `http://localhost:3000`. API tersedia pada `http://localhost:8080/api/v1`.

Untuk menghentikan stack, gunakan `make down`. Volume database dipertahankan; penghapusan volume harus menjadi tindakan terpisah yang disengaja.

Setup belum benar-benar tanpa interaksi karena bootstrap Admin sengaja dipisahkan dan meminta password melalui stdin. Alur Compose, migration, bootstrap, login, P2P, dan O2C harus diuji pada tahap runtime acceptance terpisah sebelum menyatakan aplikasi siap end-to-end.

## Menjalankan backend secara native di Windows

Prasyarat: Go 1.26+, Node.js 22+, npm, dan PostgreSQL 17 yang sudah tersedia. Di PowerShell, set `DATABASE_URL` ke host `localhost`, lalu dari `backend/` jalankan `go run ./cmd/migrate up`, `go run ./cmd/bootstrap-admin --email <email-admin>`, dan `go run ./cmd/api`. Dari `frontend/`, jalankan `npm ci` lalu `npm run dev`.

## Verifikasi

- `make verify` — gate lengkap untuk lingkungan POSIX; memerlukan dependency yang sudah terpasang dan sebagian target memakai tool Node melalui `npx`.
- `powershell -ExecutionPolicy Bypass -File scripts/verify-safe.ps1` — unit/static gate native PowerShell yang tidak memakai database, Docker, migration, bootstrap, API, atau browser.
- `make test-integration` — integration test dengan `TEST_DATABASE_URL` yang sudah diekspor.
- `make test-e2e` — browser E2E terhadap stack lokal yang sedang berjalan.
- `make migrate-verify` — integration test migrasi.

Production deployment, hosting, domain, TLS/edge configuration, monitoring produksi, dan live secrets tidak termasuk Phase 2 dan memerlukan rencana serta persetujuan terpisah.
