# Contributing to Kadr

## Branches

- **`develop`** — integration branch for completed stages.
- **`grok_bot`** — agent implementation branch (Stage 0 work lands here).
- **`main`** — release line; do not target directly from feature work.

**Rules (hard):**

1. Do all implementation on `grok_bot` (or short-lived branches cut from it if needed).
2. Open **one pull request** from `grok_bot` → `develop` per stage unless maintainers say otherwise.
3. Never open a PR into `main`, never push to `main` or `develop` directly, never merge from the agent.

## Checks before push

```bash
make test
COMPOSE_PROFILES=dev ./scripts/ci-integration.sh
```

Every PR must pass the full GitHub Actions suite (Go lint/vet/test, frontend lint/typecheck/test/build, Docker Compose integration, Playwright smoke).

## Secrets

Do not commit credentials, production hostnames, or real SMTP settings. Use `.env.example` and local overrides only.
