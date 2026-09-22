# Mini ERP UMKM

Baseline **Phase 1 — architecture and process design** dan rencana **Phase 2 — MVP implementation** telah disetujui. Implementasi berjalan dalam satu phase dengan empat checkpoint internal.

## Status

- Arsitektur: modular monolith dengan Clean Architecture per domain
- Process source of truth: BPMN 2.0 di `docs/bpmn/`
- Traceability: `docs/PROCESS-MAPPING.md`
- Struktur source: `frontend/` untuk Next.js dan `backend/` untuk Go modular monolith
- Keputusan desain: Phase 1 dan rencana empat-checkpoint Phase 2 telah disetujui

## Dokumen utama

- `docs/PHASE-1-DESIGN.md` — ringkasan arsitektur dan keputusan
- `docs/BUSINESS-RULES.md` — actor, domain, dan business rules
- `docs/PROCESS-MAPPING.md` — mapping BPMN ke software
- `docs/ERD.md` — proposal model data
- `docs/STATE-MACHINES.md` — transisi seluruh business document
- `docs/API-CONTRACT.md` — proposal REST API
- `docs/REPOSITORY-STRUCTURE.md` — struktur modular monolith + Clean Architecture
- `docs/REVIEW-FINDINGS.md` — keputusan final dan hasil design review
- `docs/TRACEABILITY-COVERAGE.md` — laporan coverage final Phase 1
- `docs/superpowers/plans/2026-09-21-mini-erp-phase-2.md` — satu proposal Phase 2 dengan empat checkpoint internal

## Implementation gate

Setiap checkpoint Phase 2 harus melewati domain, API, integration, authorization, audit, UI, E2E, race/concurrency, migration, security, dan traceability verification sebelum checkpoint berikutnya dimulai. Production deployment tidak termasuk dalam Phase 2.
