# Кадр (Kadr)

Private photo and video feed (memories.gallery-style). **Stage 1** adds accounts: register, email verification, login, transactional outbox, and a `send_email` worker backed by Mailpit in dev. **Stage 0** laid down Docker, migrations, media-runner smoke, and CI; upload and feed arrive in later stages.

## Stack

| Layer | Tech |
|-------|------|
| API / worker | Go 1.22 (`cmd/kadr` — `api`, `worker`, `migrate` one-shot in Compose) |
| UI | React 19 + TypeScript (Vite) |
| DB | MySQL 8.4 |
| Media | `media-runner` — ffmpeg, libvips, libheif (isolated, `network_mode: none`) |
| Edge | nginx reverse proxy (static SPA + `/api` upstream) |
| Dev mail | Mailpit (`docker compose --profile dev`) |

## Quick start (Docker)

```bash
cp .env.example .env
docker compose --profile dev up --build
```

- App (proxy): http://127.0.0.1:8088  
- API health: http://127.0.0.1:8088/api/health  
- Mailpit UI (dev profile): http://127.0.0.1:8025  

## Tests

```bash
# Go
go vet ./...
go test ./... -race -count=1

# Frontend
cd web && npm ci && npm run lint && npm run typecheck && npm run test && npm run build

# Full integration (compose + DB + media smoke)
./scripts/ci-integration.sh

# Playwright (stack + Mailpit; set MAILPIT_URL in CI)
cd web && PLAYWRIGHT_BASE_URL=http://127.0.0.1:8088 MAILPIT_URL=http://127.0.0.1:8025 npm run test:e2e
```

## Layout

- `cmd/kadr` — single binary, multiple commands (per architecture ADR-001)
- `internal/` — config, DB, migrations, API, worker
- `web/` — frontend shell
- `deploy/` — Dockerfiles, nginx, media-runner scripts
- `fixtures/` — licensed/sample media for CI smoke
- `docs/stage-0.md` — stage 0 report
- `docs/stage-1.md` — stage 1 report
- `docs/resource-limits.md` — VPS memory/CPU budget

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch rules.
