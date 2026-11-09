# Stage 3 report — visibility confirmation and resumable tus UI

## Scope delivered (spec §7, §13, §21; not stage 4)

- **Public feed confirmation** when switching private → public: modal copy per Den («Ваша лента станет видна всем по ссылке. Продолжить?»), **«Сделать публичной»** / **«Отмена»**, focus trap, **Esc** cancels; private switch stays one-click.
- **`tus-js-client`** in the upload UI: chunked uploads, retries, progress bar, **local fingerprint resume**, server **unfinished list** via `GET /api/me/uploads` + **Продолжить** (re-pick same file).
- Pre-upload note: original not stored (spec §21).
- **`X-Robots-Tag: noindex`** on public **timeline** responses (feed metadata already had it in stage 2).

**Not stage 3:** `derive`/media-runner, timeline media, trash/admin (stage 4+).

## Screenshots (Playwright + axe, committed)

### Desktop (1440)

| Screen | |
|--------|---|
| Public visibility confirm | ![Confirm desktop](screenshots/public-confirm-desktop-1440.png) |
| Home (public) | ![Home public desktop](screenshots/home-public-desktop-1440.png) |
| Upload progress | ![Progress desktop](screenshots/upload-progress-desktop-1440.png) |
| Upload complete | ![Done desktop](screenshots/upload-done-desktop-1440.png) |

### Mobile (390)

| Screen | |
|--------|---|
| Public visibility confirm | ![Confirm mobile](screenshots/public-confirm-mobile-390.png) |
| Home (public) | ![Home public mobile](screenshots/home-public-mobile-390.png) |
| Upload progress | ![Progress mobile](screenshots/upload-progress-mobile-390.png) |
| Upload complete | ![Done mobile](screenshots/upload-done-mobile-390.png) |

## Decisions made

| Topic | Decision |
|-------|----------|
| Modal copy | Den 2026-11-09 (§13) |
| tus chunk size | 256 KiB default (`VITE_TUS_CHUNK_SIZE` override) |
| Resume UX | tus fingerprint + explicit resume for server partials |
| Going private | No confirmation (spec: public is the sensitive transition) |
| VPS / worker | Unchanged: 1 CPU, 2 GB, `KADR_WORKER_CONCURRENCY=1` |

## Known limitations

- Resume still requires the same file bytes client-side (tus protocol).
- No dedicated pause/cancel control in UI (tus retries on network errors).
- Timeline remains empty until stage 4.

## Tests

| Suite | Count (final head `6dda011`) |
|-------|--------------------------------|
| Go `go test ./... -race` | **12** test functions, **10** packages with tests |
| Vitest | **3** files / **3** tests |
| Playwright + axe | **20** passed |

## CI

**Green run:** https://github.com/dsefremov/kadr/actions/runs/42693437265 (`go-backend` + `-race`, `web`, `docker-integration` with Playwright + axe).

## Open questions

1. Production `KADR_PUBLIC_URL` / email link format before release?
2. Login rate limits at scale (middleware vs edge)?
3. Admin bootstrap beyond CLI?
4. Argon2id 19 MiB revisit in stage 9 (unchanged until then)?
