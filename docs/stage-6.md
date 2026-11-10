# Stage 6 report — trash + dedup in trash (spec v0.3 §15, §12)

## Scope delivered

- **Migration `000005_trash`:** `purge_after` on `media`; schema readiness **v5**.
- **Trash APIs:** `POST /api/media/{id}/trash`, `POST /api/media/{id}/restore`, `GET /api/me/trash`, `DELETE /api/me/trash` (manual empty).
- **Quota:** `used_bytes` ↔ `trash_bytes` on trash/restore; trashed items stay in quota (Den).
- **Timeline:** trashed items excluded; owner can still fetch `/m/...` previews for trash UI.
- **Worker:** `purge_trash` job (`idem_key` `purge:{media_id}`) scheduled at `purge_after` (30 days).
- **Dedup (§12):** active duplicate → `409` `duplicate`; match in trash → `409` `duplicate_in_trash` + `restore_url`.
- **UI:** trash control on timeline, `/trash` page, restore + empty trash, duplicate-in-trash restore on home.
- **Tests:** Go retention check; Playwright stage 6 + screenshots + axe; `ci-timeline-smoke.sh` trash/restore.

**Not stage 6 (per Den / plan):** PNG flatten consent (§17); admin; public feed deep links; preview backfill `v1sdr20261110`; account deletion (§15.2); HDR HEIC golden.

## Fixtures

Unchanged CC0 set from stage 4–5. **HDR HEIC golden:** not added — no verifiable CC0 PQ/HLG HEIC still located (see limitations).

## Screenshots (Playwright + axe, 1440 / 390)

| Desktop 1440 | Mobile 390 |
|---|---|
| ![Trash](screenshots/trash-desktop-1440.png) | ![Trash mobile](screenshots/trash-mobile-390.png) |

## Decisions made

| Topic | Decision |
|-------|----------|
| Trash retention | 30 calendar days via `purge_after` |
| Trashable states | `ready`, `waiting_quota` only |
| Dedup while `processing` | Keep **409** same `media_id` (Den) |
| Tus `Location` | Relative (Den) |
| Public dup link | Deferred with public feed (not in stage 6) |
| §17 consent | Deferred to UX (Den) |
| Preview backfill | Admin job later; not in stage 6 (Den) |

## Known limitations

- **HDR HEIC golden** — still no CC0 HDR still with auditable PQ/HLG metadata in HEIC.
- **`row_version` / `If-Match`** — not added; change-date stage will introduce concurrency fields.
- **Per-item purge UX** — no separate “delete one forever” API; empty trash deletes all trashed items.
- **Automated purge test** — 30-day job covered by unit scheduling + worker handler; no time-travel e2e.

## Tests

| Suite | Count (pre-final head) |
|-------|------------------------|
| Go `-race` | 11 packages OK |
| Vitest | **3** |
| Playwright + axe | **44** listed (stage 2–6) |

Integration: `ci-timeline-smoke.sh` adds trash → list → restore.

## Local docker-integration (agent VM)

```bash
cd /path/to/kadr
export COMPOSE_PROFILES=dev
KADR_KEEP_UP=1 sudo -E ./scripts/ci-integration.sh
```

**Result (2026-11-10, agent VM, stage 6 code pre-doc):**

| Step | Result |
|------|--------|
| `docker compose build` + migrate v5 | OK (use gateway DSN when container→mysql fails) |
| `GET /ready` + `/api/health` | OK (`schema: true` @ v5) |
| `media-runner smoke` | OK |
| `ci-media-pipeline.sh` | OK (320/640/1280) |
| `ci-media-benchmarks.sh` | **Stopped** — HDR HLG transcode hit cgroup **768 MiB** (`Killed` mid-encode on this VM) |
| `ci-timeline-smoke.sh` | OK (timeline, duplicate 409, **trash/restore**) |
| `measure-memory.sh` (idle) | mysql **455.5**, media-runner **2.0**, worker **2.1**, proxy **11.6**, mailpit **18.8** MiB RSS |
| Playwright stage 6 | **8 passed** via `vite preview :8088` → API `:8080` (docker `proxy` stopped to free port) |

## CI (PR → develop)

| Item | Value |
|------|--------|
| **PR** | _TBD_ `grok_bot` → `develop` |
| **Green CI (final)** | Head **`_HEAD_`** — [run _RUN_](https://github.com/dsefremov/kadr/actions/runs/_RUN_) (`go-backend`, `web`) |

## Open questions (with proposal)

1. **Account deletion (§15.2)** — Proposal: dedicated stage 7 with async wipe job; trash purge reuses `PurgePermanent` helper.
2. **Single-item permanent delete** — Proposal: add `DELETE /api/media/{id}/trash` mirroring empty-trash for one `public_id`.
3. **Trash from processing card** — Proposal: only after cancel-upload API; until then trash remains on ready timeline items.
4. **Purge job sweep** — Proposal: optional nightly `purge_trash_sweep` job in addition to per-item `run_after` for clock skew.
5. **HDR HEIC fixture** — Proposal: add when a CC0 file with verifiable HDR metadata and license URL is identified (same as stage 5).
