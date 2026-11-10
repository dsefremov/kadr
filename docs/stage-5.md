# Stage 5 report — preview ladder + duplicate upload (spec v0.3 §12)

## Scope delivered

- **WebP previews** `preview-320`, `preview-640`, `preview-1280` in `derive.sh` (vips thumbnail, q80).
- **Profile hash** `v1sdr20261111` (new ladder; media derived under `v1sdr20261110` keeps old variant paths until re-upload).
- **Timeline** `srcSet` 320w / 640w / 1280w; default `src` = 640 (`data-testid="timeline-thumb"`).
- **Duplicate upload (§12):** final tus PATCH → SHA-256 + size → **409** JSON `{ error, media_id, preview_url, timeline_url }`; reserve released; upload row `cancelled`; no derive job.
- **UI:** `upload-duplicate` + `upload-duplicate-link` to existing preview.
- **CI hooks:** pipeline asserts three preview files; timeline smoke checks ladder URLs + duplicate 409.
- **Tests:** Go (`dedup`, timeline URLs), Playwright stage 5 + screenshots + axe.

**Deferred (Den):** PNG flatten consent API (§17); trash/admin; HDR HEIC golden in repo.

## Fixtures

CC0 set unchanged from stage 4. **HDR HEIC golden** not added — HDR coverage via `benchmark-hdr-hlg.mov` video path.

## Screenshots (Playwright + axe, 1440 / 390)

| Desktop 1440 | Mobile 390 |
|---|---|
| ![Timeline srcset](screenshots/timeline-srcset-desktop-1440.png) | ![Timeline mobile](screenshots/timeline-srcset-mobile-390.png) |
| ![Duplicate upload](screenshots/duplicate-upload-desktop-1440.png) | ![Duplicate mobile](screenshots/duplicate-upload-mobile-390.png) |

## Decisions made

| Topic | Decision |
|-------|----------|
| Preview ladder | 320 + 1280 alongside 640 (Den) |
| Duplicate | 409 on final PATCH; JSON link to `preview_url` + feed `timeline_url` |
| Upload row on dup | `cancelled` (ENUM has no `duplicate` state) |
| Tus `Location` | Relative `/api/uploads/{id}` so PATCH stays on page origin |
| Tus 409 in browser | `onShouldRetry` false when body is `duplicate`; parse body from `DetailedError` |
| Profile hash | `v1sdr20261111` for new variant set |
| Dedup while `processing` | Included in lookup (`processing`, `ready`, `waiting_quota`) |
| §17 consent API | Deferred to UX design (Den) |

## Known limitations

- **HDR HEIC golden** deferred — no CC0 HDR still with verifiable PQ/HLG metadata in HEIC in repo.
- **Backfill** — existing ready media only expose 640 in timeline until re-upload or a re-derive job.
- **`timeline_url` in 409** — API feed path (session), not a public slug deep link.
- **Cloud agent VM** — container→container TCP to `api:8080` from nginx can fail; `scripts/ci-integration.sh` already documents mysql/mailpit gateway fallback (proxy on :8088 may need same pattern on some hosts).

## Processing memory / time (768 MiB cap)

Measured locally (`docker run --memory 768m`, `derive.sh`, profile `v1sdr20261111`):

| Asset | Kind | Wall (s) | Peak RSS (MiB) | ≤768 MiB |
|-------|------|----------|----------------|----------|
| `sample.heic` | photo | 1 | 41.2 | yes |
| `photo-24mp.jpg` | photo | 1 | 49.7 | yes |
| `benchmark-hdr-hlg.mov` | video | 33 | 276.7 | yes |

Adding 320/1280 WebP did not change HDR video peak materially vs stage 4.

## Tests (local on final head)

| Suite | Result |
|-------|--------|
| Go `vet` + `test -race` | 11 packages OK |
| `go test` functions | 21 |
| Vitest | 3 passed |
| Playwright stage 5 (+ axe in screenshot spec) | 4 passed (2 flow + 2 screenshot viewports) |
| `ci-media-pipeline.sh` | OK |
| `ci-timeline-smoke.sh` | OK (timeline ladder URLs + duplicate 409) |

Full Playwright matrix (all stages) not re-run locally in this turn; stage 5 specs green on `:8088` vite preview → API `:8080`.

## CI / GitHub Actions

| Item | Value |
|------|--------|
| **Workflow** | Unchanged (`.github/workflows/ci.yml`) |
| **Last full green CI on repo** | Merge **PR #6** → `develop` @ **`c722293`** — [run 42837503467](https://github.com/dsefremov/kadr/actions/runs/42837503467) (go-backend, web, docker-integration) |
| **CI on this stage-5 head** | **Not run** — Actions jobs for `dsefremov` currently fail in ~2–3s without a runner (billing; Den investigating). Verified **locally** on commit below instead. |

**Final head (stage 5 PR):** `a30efc1`

## Open questions (with proposal)

1. **Re-derive backfill** for `v1sdr20261110` media — Proposal: optional admin/job in a later stage; until then only 640 in API for old items.
2. **Duplicate while first copy still `processing`** — Now 409 with same `media_id`; Proposal: optional 202 + poll UX if uploads race often.
3. **HDR HEIC golden** — Proposal: add CC0 PQ/HLG HEIF when a vetted source is identified (stage 6+).
4. **Absolute tus `Location` with `KADR_PUBLIC_URL`** — Proposal: keep relative for same-site; add absolute only if off-site tus clients appear.
