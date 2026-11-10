# Stage 0 report — infrastructure

## Goal

Deliver Docker Compose–based infrastructure for **Кадр** without product features (auth, uploads, feed, media pipeline business logic).

## Delivered

- **Compose stack:** `mysql`, `api`, `worker`, `media-runner`, `proxy`, optional `mailpit` (dev profile).
- **Go binary** `kadr` with `api`, `worker`, and embedded SQL migrations (`000001_baseline`).
- **HTTP:** `GET /health`, `GET /ready`, `GET /api/health` (version + DB/migration flags).
- **Worker:** single-threaded job poll stub against `jobs` table (concurrency env default `1`).
- **media-runner:** isolated image (no network), ffmpeg/libvips/libheif, `smoke` command for CI.
- **Frontend:** Vite React shell styled from mockups v0.2 tokens.
- **CI:** GitHub Actions — Go lint/vet/test `-race`, web lint/typecheck/test/build, compose integration, Playwright + axe, screenshots artifact.
- **Docs:** README, CONTRIBUTING, `resource-limits.md`, this file.

## How to verify locally

See [README.md](../README.md).

## Decisions made

| Topic | Decision |
|-------|------------|
| Go version | **1.22** (available on builder images; spec allows pinning at implementation start) |
| Reverse proxy | **nginx** in dedicated image (spec: Caddy or nginx) |
| Migrations | Embedded SQL in `internal/migrate`, applied on api/worker startup |
| Job queue in stage 0 | Minimal `jobs` table + worker poll stub; **SKIP LOCKED** wiring deferred to stage 1 |
| SMTP in dev | **Mailpit** via compose profile `dev` |
| Media storage | Named volumes `kadr_media_data`, `kadr_media_spool` on VPS disk |
| Frontend lint | **oxlint** (Vite 8 default) instead of ESLint |
| HEIC fixture | Generated PNG → HEIC via Debian `heif-enc` (committed small file) |
| Decimal units | Documented for later stages; not enforced in stage 0 |
| MySQL from Go services | Default DSN uses hostname `mysql`; `ci-integration.sh` falls back to bridge gateway + published `:3306` when container→container TCP is broken |
| API published port | `:8080` exposed for health/readiness smoke (proxy still serves UI in normal environments) |

## Known limitations

- No TLS termination in compose (production adds real certs on proxy).
- Migrations run from both `api` and `worker` on startup (idempotent baseline only).
- `media-runner` not invoked by worker yet (stage 4).
- Playwright/axe rules may need tuning when interactive UI lands.
- MySQL healthcheck uses dev-only root password literal (matches `.env.example`).

## HDR tone-mapping (stage 4 estimate, no code in stage 0)

Den: HDR video is tone-mapped to SDR in v1. On top of the baseline **2–3 day** assumption for allowlist + transcoding, tone-mapping adds about **+3 days** for stage 4:

- FFmpeg filter graph (e.g. zscale/tonemap) selection and presets
- Golden tests on iPhone HEVC HDR samples (accept / reject / map)
- UX copy for unsupported HDR stills vs mapped video
- Performance check on 1 vCPU / 768 MiB runner cap

**Total HDR delta:** ~**3 days** (range **2–4** depending on test asset availability).

## Open questions

1. Decimal vs binary units for 50 MB / 2 GB / 5 GB (spec recommendation: decimal).
2. HDR stills (HEIC PQ/HLG) — reject vs tone-map in v1.
3. SMTP provider and sending domain before production.
4. Exact MySQL `innodb_buffer_pool_size` on 2 GB VPS after idle measurements.
5. Proxy choice for production TLS (Caddy auto-TLS vs nginx + certbot).
6. Whether `api` or a one-shot init container should own migrations exclusively.
