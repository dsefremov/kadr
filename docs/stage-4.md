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
| `fixtures/in/sample.heic` | `fixtures/generate-fixtures.sh` → `heif-enc` in media-runner | [CC0-1.0](https://creativecommons.org/publicdomain/zero/1.0/) |
| `fixtures/in/sample.mov` | ffmpeg `testsrc2` + `sine` (2 s) | [CC0-1.0](https://creativecommons.org/publicdomain/zero/1.0/) |
| `fixtures/in/benchmark-hdr-hlg.mov` | ffmpeg HLG BT.2020, 1080p×30 s | [CC0-1.0](https://creativecommons.org/publicdomain/zero/1.0/) |
| `fixtures/in/photo-24mp.jpg` | ffmpeg `testsrc2` 6000×4000 | [CC0-1.0](https://creativecommons.org/publicdomain/zero/1.0/) |
| `fixtures/in/sample.png` | ffmpeg solid color 64×64 | [CC0-1.0](https://creativecommons.org/publicdomain/zero/1.0/) |

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
| Runner RSS cap | `VIPS_CONCURRENCY=1`, libvips cache 64 MiB, ffmpeg/x264 **-threads 1** |

## Known limitations

- Peak memory from cgroup `memory.peak` + `VmHWM` in `derive.sh` `metrics.peak_rss_kb` (per one-off derive container).
- `preview-320` / `preview-1280` not generated (spec lists; stage 4 ships 640).
- Dedup unique constraint — second upload frees reserve silently (no duplicate UI).
- Video two-pass / nightly recompress not implemented.

## Processing memory / time

**Peak derive jobs** (`scripts/ci-media-benchmarks.sh`, compose **768 MiB** limit, `VIPS_CONCURRENCY=1`, ffmpeg `-threads 1`):

| Asset | Kind | Wall (s) | Peak RSS (MiB) | Under 768 MiB |
|-------|------|----------|----------------|---------------|
| `sample.heic` | photo | 1 | **34.0** | yes |
| `photo-24mp.jpg` (6000×4000 → 3200) | photo | 0 | **49.6** | yes |
| `benchmark-hdr-hlg.mov` (HLG 1080p, 30 s → SDR H.264) | video | 45 | **275.3** | yes |

Source: CI artifact `media-benchmarks` (`derive.sh` → `metrics.peak_rss_kb`, cgroup `memory.peak` in one-off `docker run --memory 768m`).

**Idle stack RSS** (`scripts/measure-memory.sh`, containers waiting): api ~41 MiB, worker ~2 MiB, media-runner ~1 MiB, mysql ~443 MiB (limit 512 MiB), proxy ~3 MiB — not transcode peaks.

## Tests

| Suite | Count (`cb4ca71`) |
|-------|-------------------|
| Go `-race` | **19** `Test*` functions (`go test ./... -race`) |
| Vitest | **3** |
| Playwright + axe | **28** passed (stage 2–4 specs, 1 worker) |

Integration: `ci-media-pipeline.sh`, `ci-timeline-smoke.sh` (timeline **items≥1**), `measure-memory.sh`.

## CI

**Head:** `cb4ca71fd0f9f6f05ea8d4b45fb95f05f9501da6`

**Green run:** https://github.com/dsefremov/kadr/actions/runs/42831699113 (`go-backend`, `web`, `docker-integration`).

## Open questions (with proposal)

1. **Preview ladder (320/640/1280)** — Proposal: add 320+1280 in stage 5 when timeline layout stabilizes.
2. **Duplicate upload UX** — Proposal: `409` + link to existing media id (spec §12).
3. **HEIC HDR stills** — Proposal: CC0 `sample.heic` is SDR smoke; HDR video covered by `benchmark-hdr-hlg.mov`; add HDR **photo** golden when libvips tone-map path is nailed down.
4. **Flatten consent endpoint** — Proposal: defer until designer flow for PNG preview (spec §17 optional).
