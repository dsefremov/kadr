# Resource limits (Stage 0)

Target VPS: **1 vCPU @ 3.3 GHz, 2 GB RAM, 30 GB NVMe** (per project brief).

Compose `deploy.resources.limits` (idle design targets):

| Service | CPU limit | RAM limit | Notes |
|---------|-----------|-----------|-------|
| MySQL 8.4 | 0.35 | 512 MiB | Primary data + job queue table |
| API | 0.25 | 128 MiB | HTTP + migrations on start |
| Worker | 0.15 | 128 MiB | **Single-threaded** poll (`KADR_WORKER_CONCURRENCY=1`) |
| media-runner | 0.50 | **768 MiB** | Strict cap; one ffmpeg job at a time in v1 |
| proxy (nginx) | 0.10 | 64 MiB | Static + `/api` proxy |
| Mailpit (dev) | 0.05 | 64 MiB | Not used in production |

**Sum of limits:** ~1.65 vCPU, ~1664 MiB — leaves headroom for kernel/page cache on a 2 GB host. On an overloaded host, OOM kills the most memory-heavy **bounded** service first (`media-runner`).

## Memory budget narrative

- **~512 MiB** — MySQL (InnoDB buffer pool kept modest on small VPS).
- **~256 MiB** — Go API + worker combined (128 MiB each).
- **≤768 MiB** — media-runner peak (ffmpeg/libvips); must not run overlapping transcodes in v1.
- **~64–128 MiB** — proxy + optional Mailpit.
- Remaining RAM — OS, filesystem cache for media volume I/O.

Measure idle RSS after `docker compose up`:

```bash
./scripts/measure-memory.sh
```

Production overrides belong in host-specific compose overrides (not committed).
