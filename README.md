# Mini ERP UMKM

Mini ERP berbasis BPMN untuk UMKM. Repository ini sedang disiapkan sebagai baseline statis penelitian impact mapping. BPMN berfungsi sebagai process specification dan sumber traceability; aplikasi tidak membaca BPMN saat runtime.

## Status

- Arsitektur: modular monolith dengan Clean Architecture per domain
- Process source of truth: BPMN 2.0 di `docs/bpmn/`
- Traceability: `docs/PROCESS-MAPPING.md`
- Struktur source: `frontend/` untuk Next.js dan `backend/` untuk Go modular monolith
- Status verifikasi saat ini: source dan unit-level verification tersedia; penerimaan runtime lokal harus diuji ulang setelah baseline remediation

## Dokumen utama

- `docs/PHASE-1-DESIGN.md` — ringkasan arsitektur dan keputusan
- `docs/BUSINESS-RULES.md` — actor, domain, dan business rules
- `docs/PROCESS-MAPPING.md` — mapping BPMN ke software
- `docs/ERD.md` — proposal model data
- `docs/STATE-MACHINES.md` — transisi seluruh business document
- `docs/API-CONTRACT.md` — proposal REST API
- `docs/REPOSITORY-STRUCTURE.md` — struktur modular monolith + Clean Architecture
- `docs/REVIEW-FINDINGS.md` — keputusan final dan hasil design review
- `docs/TRACEABILITY-COVERAGE.md` — laporan coverage implementasi final
- `docs/SECURITY-REVIEW.md` — hasil security review
- `docs/RELEASE-CANDIDATE.md` — bukti dan batasan release candidate
- `docs/superpowers/plans/2026-09-21-mini-erp-phase-2.md` — satu proposal Phase 2 dengan empat checkpoint internal

## Menjalankan secara lokal dengan Compose

Prasyarat: Docker/Compose. Port default yang digunakan adalah `3000`, `8080`, dan `5432`. Jika bentrok, atur `FRONTEND_HOST_PORT`, `BACKEND_HOST_PORT`, atau `POSTGRES_HOST_PORT` di `.env` tanpa mengubah port internal container.

1. Salin `.env.example` menjadi `.env`. Isi `POSTGRES_PASSWORD` dan `SESSION_PEPPER` dengan nilai lokal yang unik; jangan commit `.env`. `DATABASE_URL` hanya dipakai oleh command backend native dan password-nya harus sama dengan `POSTGRES_PASSWORD`.
2. Jalankan `docker compose up -d --build`. Service `migrate` menunggu PostgreSQL sehat dan menjalankan seluruh migration sebelum backend dimulai.
3. Khusus database baru, buat Admin pertama secara manual dan interaktif: `docker compose run --rm --entrypoint bootstrap-admin backend --email <email-admin>`. Tidak ada credential default dan bootstrap akan menolak database yang sudah memiliki user.
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
