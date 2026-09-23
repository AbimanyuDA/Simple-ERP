# Mini ERP UMKM

Mini ERP berbasis BPMN untuk UMKM. Phase 2 telah selesai melalui empat checkpoint internal dan berada pada status **release candidate untuk review lokal**; belum untuk production deployment.

## Status

- Arsitektur: modular monolith dengan Clean Architecture per domain
- Process source of truth: BPMN 2.0 di `docs/bpmn/`
- Traceability: `docs/PROCESS-MAPPING.md`
- Struktur source: `frontend/` untuk Next.js dan `backend/` untuk Go modular monolith
- Status implementasi: Checkpoint 1–4 lulus seluruh gate

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

## Menjalankan secara lokal

Prasyarat: Docker/Compose, Go 1.26.6+, Node.js 22+, dan npm.

1. Salin `.env.example` menjadi `.env`. Isi `POSTGRES_PASSWORD`, `DATABASE_URL`, dan `SESSION_PEPPER` dengan nilai lokal yang unik; jangan commit `.env`.
2. Jalankan PostgreSQL: `docker compose up -d postgres`.
3. Dari `backend/`, jalankan migrasi dengan `DATABASE_URL` yang menunjuk ke `localhost:5432`, lalu jalankan `go run ./cmd/bootstrap-admin --email <email-admin>`. Password dibaca dari stdin.
4. Dari root, jalankan `docker compose up -d --build backend frontend`.
5. Buka `http://localhost:3000`. API tersedia pada `http://localhost:8080/api/v1`.

Untuk menghentikan stack, gunakan `make down`. Volume database dipertahankan; penghapusan volume harus menjadi tindakan terpisah yang disengaja.

## Verifikasi

- `make verify` — format, vet, lint, typecheck, unit/race, BPMN, OpenAPI, dan traceability.
- `make test-integration` — integration test dengan `TEST_DATABASE_URL` yang sudah diekspor.
- `make test-e2e` — browser E2E terhadap stack lokal yang sedang berjalan.
- `make migrate-verify` — integration test migrasi.

Production deployment, hosting, domain, TLS/edge configuration, monitoring produksi, dan live secrets tidak termasuk Phase 2 dan memerlukan rencana serta persetujuan terpisah.
