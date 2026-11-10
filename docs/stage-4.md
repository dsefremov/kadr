# Stage 4 report — media pipeline (spec v0.3 §8–11, not stage 5)

## Scope delivered

- **Schema** `000004_media`: `media`, `media_variants`, `source_files`; readiness **v4**.
- **`derive` jobs**: worker spool → **media-runner** (`watch` + `derive.sh`), single-thread, **no network**, **768 MiB** cap unchanged.
- **Photo**: JPEG q85 long side ≤3200, WebP preview 640; PNG alpha → white flatten; **animated** GIF/WebP/APNG rejected.
- **Video**: MP4 H.264 CRF 23, ≤1080p 30 fps, AAC 128k; **>60 min** rejected; **HDR → SDR** tone-map (`zscale`/`tonemap` when PQ/HLG).
- **Formats**: JPEG, PNG, WebP, HEIC, MP4, MOV (sniff + declared kind).
- **Quota**: decimal bytes; commit on `ready`; `waiting_quota` without overspend; reserve released on failure.
- **API**: timeline items with `/m/...` URLs; `GET /api/me/media?status=`; `GET /m/{id}/{variant}.{profileHash}`.
- **UI**: timeline grid, processing / error states, 3s poll.

**Not stage 4:** trash UI, admin, date override, buckets, dedup UX, stage 5+.

## Fixtures (licensed)

| File | Source | License |
|------|--------|---------|
| `fixtures/in/sample.heic` | [nokiatech/heif](https://github.com/nokiatech/heif) conformance examples | BSD-3-Clause |
| `fixtures/in/sample.mov` | [apple/media](https://github.com/apple/media) HEVC sample (`SampleHEVC`) | Apple sample terms (refer to repo) |
| `fixtures/in/sample.png` | Generated minimal PNG for CI | CC0 / project-owned |

## Screenshots (Playwright + axe)

| Desktop 1440 | Mobile 390 |
|---|---|
| ![Processing](screenshots/processing-desktop-1440.png) | ![Processing mobile](screenshots/processing-mobile-390.png) |
| ![Timeline](screenshots/timeline-ready-desktop-1440.png) | ![Timeline mobile](screenshots/timeline-ready-mobile-390.png) |

## Decisions made

| Topic | Decision |
|-------|----------|
| HDR (Den) | Tone-map photo/video to **SDR** in v1 |
| Profile hash | Constant `v1sdr20261110` for all variants |
| Spool | Shared `media_spool` volume; worker writes, runner watches |
| Permanent errors | Job completes (no retry) for animated/unsupported/length |
| Transient errors | Spool timeout → job retry via queue |
| Derive idempotency | On tus-complete, reset `derive:{upload_id}` to **pending**; retry if upload not yet `complete` |
| Preview sizes | WebP **640** in v1 (320/1280 deferred) |
| EXIF dates | `taken_sort` = upload time (EXIF in stage 16) |

## Known limitations

- Runner peak RSS logged in `result.json`; no continuous profiling in CI artifact yet.
- `preview-320` / `preview-1280` not generated (spec lists; stage 4 ships 640).
- Dedup unique constraint — second upload frees reserve silently (no duplicate UI).
- Video two-pass / nightly recompress not implemented.

## Processing memory / time (smoke + derive script)

Measured on CI runner via `scripts/measure-memory.sh` and `derive.sh` `metrics` field:

| Asset | Wall time (approx.) | Notes |
|-------|---------------------|-------|
| HEIC → JPEG+WebP (smoke) | <15 s | libvips |
| MOV → H.264 (smoke) | <30 s | ffmpeg single pass |
| sample.png derive | <5 s | CI `ci-media-pipeline.sh` |

Under **768 MiB** media-runner limit in compose; single job at a time.

**CI idle RSS** (integration stack, `measure-memory.sh` on run 42809412149): api **41.3** MiB, worker **2.1** MiB, media-runner **1.0** MiB, mysql **443** MiB (limit 512 MiB), proxy **3.4** MiB — derive/transcode peaks stay within media-runner **768 MiB** cap during smoke/derive jobs.

## Tests

| Suite | Count (`46d744f`) |
|-------|-------------------|
| Go `-race` | **19** `Test*` functions (`go test ./... -race`) |
| Vitest | **3** |
| Playwright + axe | **28** passed (stage 2–4 specs, 1 worker) |

Integration: `ci-media-pipeline.sh`, `ci-timeline-smoke.sh` (timeline **items≥1**), `measure-memory.sh`.

## CI

**Head:** `46d744f1d640d5403b2cd84f0da48223832ea5ea` (docs + screenshots; pipeline fix `e497bdf`)

**Green run:** https://github.com/dsefremov/kadr/actions/runs/42810729740 (`go-backend`, `web`, `docker-integration`).

## Open questions (with proposal)

1. **Preview ladder (320/640/1280)** — Proposal: add 320+1280 in stage 5 when timeline layout stabilizes.
2. **Duplicate upload UX** — Proposal: `409` + link to existing media id (spec §12).
3. **HEIC HDR stills** — Proposal: rely on libvips tone-map; add golden HDR fixture before release.
4. **Flatten consent endpoint** — Proposal: defer until designer flow for PNG preview (spec §17 optional).
