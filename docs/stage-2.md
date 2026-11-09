# Stage 2 report — auth completion, tus upload, feed API

## Scope delivered

### Stage 1 leftovers (Den 2026-11-09)
- **Password reset** API `POST /api/auth/password/reset` (request + complete) and UI (`/forgot`, `/reset`).
- Shared **«Ссылка недействительна»** component (`InvalidLink`) for verify + reset.
- **Login rate limit:** 5 failures / 15 min per email (middleware via `auth_failures`).
- **Resend limits:** 60 s + 5/hour per address (existing) + **20/hour per IP** (`auth_resend_log`).
- **`cleanup_unverified`** worker job (sweep every 6 h, delete &gt; 7 days unconfirmed).

### Stage 2 product slice (spec §5–7, §16–17, ADR-005)
- Migration **`000003_uploads_auth_limits`**: `uploads`, `auth_failures`, `auth_resend_log`.
- **tus 1.0** owner API: Creation, HEAD, PATCH, DELETE, `GET /api/me/uploads`.
- **Quota reserve** on `Upload-Length`; **expire_upload** job (+24 h).
- **Feed:** `GET /api/u/{handle}/feed`, `GET /api/feeds/{id}/timeline` (empty until stage 4), `PATCH /api/me/feed`.
- **`GET /api/me`** returns real quota counters + handle/visibility.
- **CLI:** `kadr admin create --email … --password …` (bootstrap admin, no web UI).
- **React:** upload zone (minimal tus client), visibility toggle, forgot/reset flows.

**Not stage 2:** `derive`/media-runner (stage 4), admin APIs, trash/timeline media, stage 3 features.

## Screenshots (Playwright + axe, committed)

Captured in CI (`web/e2e/screenshots.spec.ts`) at **1440×900** and **390×844**.

### Desktop (1440)

| Screen | |
|--------|---|
| Home — upload zone + visibility | ![Home desktop](screenshots/home-desktop-1440.png) |
| Upload accepted (tus) | ![Upload done desktop](screenshots/upload-done-desktop-1440.png) |
| `/forgot` | ![Forgot desktop](screenshots/forgot-desktop-1440.png) |
| `/reset` (new password form) | ![Reset desktop](screenshots/reset-desktop-1440.png) |
| «Ссылка недействительна» | ![Invalid link desktop](screenshots/invalid-link-desktop-1440.png) |

### Mobile (390)

| Screen | |
|--------|---|
| Home — upload zone + visibility | ![Home mobile](screenshots/home-mobile-390.png) |
| Upload accepted (tus) | ![Upload done mobile](screenshots/upload-done-mobile-390.png) |
| `/forgot` | ![Forgot mobile](screenshots/forgot-mobile-390.png) |
| `/reset` | ![Reset mobile](screenshots/reset-mobile-390.png) |
| «Ссылка недействительна» | ![Invalid link mobile](screenshots/invalid-link-mobile-390.png) |

## Stage 9 notes (unchanged)

- `innodb_buffer_pool_size`, TLS (Caddy vs nginx), **Argon2id 19 MiB** revisit in stage 9.

## Decisions made

| Topic | Decision |
|-------|----------|
| `KADR_PUBLIC_URL` | Env-driven; production domain before release |
| Login limits | DB-backed per email (§7) |
| Resend IP cap | 20/hour per IP (§7 table) |
| First admin | `kadr admin create` in container only |
| tus client (web) | Single-chunk helper for stage 2; full resume later |
| `derive` jobs | Acknowledged in worker; pipeline in stage 4 |
| Accent / argon2 | Keep stage 1 values (#c73e22, 19 MiB) |
| Schema version | **3** required for readiness |
| Transparent PNG | Flatten onto **white** in processing (no consent endpoint; Den) |
| Animated images | **Rejected** at ingest (Den) |
| Video duration | **60 min** cap (Den) |

## Known limitations

- Timeline empty until `media` rows exist (stage 4).
- `derive` jobs no-op success (placeholder).
- No checksum extension; checksum optional per spec.
- Upload kind from metadata defaults to `photo` if missing.
- Visibility toggle without confirmation dialog (spec UX polish later).

## Tests

| Suite | Count (CI head `721c1e1`) |
|-------|---------------------------|
| Go `go test ./... -race` | **11** test functions, **9** packages with tests |
| Vitest | **2** files / **2** tests |
| Playwright + axe | **14** passed (2 viewports × auth, placeholder, stage2 flows, screenshot suite) |

## CI

**Green run:** https://github.com/dsefremov/kadr/actions/runs/42685535490 (`go-backend` + `-race`, `web`, `docker-integration` with Playwright + axe).

## Open questions

1. Confirmation dialog copy for public visibility (spec §13)?
2. When to add full tus-js-client with resume UX on mobile?
3. Production `KADR_PUBLIC_URL` / email link format before release?
4. Login rate limits at scale (app middleware vs reverse proxy)?
5. Admin bootstrap beyond CLI (runbook only)?
6. Argon2id 19 MiB revisit in stage 9 (scheduled; unchanged until then)?
