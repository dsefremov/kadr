# Stage 6 report — корзина и квоты (spec v0.3 §7, §12, §15)

## Scope delivered (план Den «Корзина и квоты», 4–6 дн.)

- **Квота 5 ГБ (десятичные):** `GET /api/me` → `quota_*`, `quota_status` (`ok` | `warning` ≥90% | `exceeded`); индикатор + сегмент корзины (макеты `quota-warning`, `quota-exceeded`).
- **Корзина:** trash / restore / список; **DELETE** навсегда (один элемент + очистить всё); квота `used` ↔ `trash`; авто **purge_trash** через 30 дней; файлы с диска снимаются в `PurgePermanent` / `PurgeAny`.
- **Дедуп + корзина:** `409` `duplicate_in_trash`, UI `upload-duplicate-trash`, восстановление (макет `upload-duplicate-trash`).
- **Удаление аккаунта:** `POST /api/me/delete-account` (пароль) → `deleting`, отзыв сессий, отмена uploads, job **`delete_account`**; UI `account-delete`, `account-delete-progress`.
- **Migrations:** `000005_trash`, `000006_account_deleting`; readiness **v6**.
- **Tests:** quota helpers; restore/quota race fixture; disk removal; Playwright stage 6 + quota/account + axe 1440/390; `ci-timeline-smoke.sh` trash → purge → re-upload.

**Без изменений (ранние этапы / Den):** tus relative Location; §17 flatten; admin; viewer/buckets; backfill `v1sdr20261110`; HDR HEIC без CC0.

## Fixtures

CC0 набор stage 4–5. **HDR HEIC golden** — не добавлен (нет проверяемого CC0 PQ/HLG HEIC).

## Screenshots (Playwright + axe)

| Desktop 1440 | Mobile 390 |
|---|---|
| ![Trash](screenshots/trash-desktop-1440.png) | ![Trash mobile](screenshots/trash-mobile-390.png) |
| ![Quota warning](screenshots/quota-warning-desktop-1440.png) | ![Quota warning mobile](screenshots/quota-warning-mobile-390.png) |
| ![Quota exceeded](screenshots/quota-exceeded-desktop-1440.png) | ![Quota exceeded mobile](screenshots/quota-exceeded-mobile-390.png) |

## Decisions made

| Topic | Decision |
|-------|----------|
| Warning threshold | **90%** accounted bytes (mockups ~92%) |
| Trash retention | 30 days, `purge_after` + worker job |
| Permanent delete | `DELETE /api/media/{id}/trash` + `DELETE /api/me/trash` |
| Account delete | Sync lock + async worker wipe (`PurgeAny` + user row) |
| Dedup in trash | `duplicate_in_trash` + restore CTA |
| §17 / public dup link | Deferred (Den) |

## Known limitations

- **HDR HEIC golden** — нет CC0 источника с проверяемой лицензией.
- **`commit_quota` / quarantine UI** — waiting_quota без отдельной страницы очереди (макет `upload-queue` частично).
- **Account wipe e2e** — progress UI без ожидания worker в CI (job покрыт unit/worker path).
- **30-day purge e2e** — без time-travel; scheduling + worker handler only.

## Покрытие плана (этапы 0–9)

Оценка **остатка** — грубая трудоёмкость в человеко-днях для команды (не календарь агента).

| # | Этап (утверждённый план) | Статус | Остаток (дн.) |
|---|--------------------------|--------|----------------|
| 0 | Инфраструктура (Compose, CI, shell) | **Готово** | — |
| 1 | Контракты / health / migrations baseline | **Готово** | — |
| 2 | Аккаунты, tus, feed API, quota reserve | **Готово** | 0.5 (polish `/account` nav в shell) |
| 3 | Загрузки, resend limits, reset password | **Готово** | 0.5 |
| 4 | Обработка медиа, quota commit | **Готово** | 1 (`waiting_quota` UX, quarantine) |
| 5 | Превью 320/640/1280, дедуп в ленте | **Готово** | — |
| 6 | **Корзина и квоты** | **Готово** (этот PR) | 1 (полировка uploads list UI) |
| 7 | Администрирование (§14) | **Нет** | **8–12** |
| 8 | Сквозная приёмка (тест-план §19, MFA rec.) | **Частично** (CI smoke) | **4–6** |
| 9 | VPS hardening, релиз `main`, бэкапы | **Частично** (limits doc) | **3–5** |

**Лента и просмотрщик (между 5 и 7):** timeline grid есть; **viewer**, buckets/seek, смена даты — **не сделано** (~**6–10** дн.).

## Tests

| Suite | Count (pre-final doc) |
|-------|------------------------|
| Go packages | 12+ (incl. `quota`, trash disk/quota tests) |
| Vitest | **3** |
| Playwright + axe | **22** (stage 6 trash/quota/account, 1440+390, axe) |

Integration: `ci-timeline-smoke.sh` — duplicate, trash, **purge forever**, re-upload.

## Local docker-integration

```bash
export COMPOSE_PROFILES=dev
KADR_KEEP_UP=1 sudo -E ./scripts/ci-integration.sh
```

| Step | Result (agent VM, 2026-11-10) |
|------|-------------------------------|
| `ci-integration.sh` (compose up) | OK; **:8088** занят vite → proxy skip; gateway DSN для migrate |
| `media-runner smoke` | OK до HDR ffmpeg; **HDR transcode Killed** (OOM на VM) |
| `ci-media-pipeline.sh` | OK |
| `ci-media-benchmarks.sh` | **Не завершён** (тот же HDR job; cap 768 MiB) |
| `ci-timeline-smoke.sh` | OK (trash → **purge forever** → re-upload) |
| Playwright stage 6 (все спеки) | **не в CI** на PR→`develop`; те же локальные **21/45** на `:8088` (2026-11-11, head **`0c1f440c2981c984cfad5c92e5208b11a40ebc80`**) — заявление «22/22» относилось к частичному прогону без полного suite |
| `measure-memory.sh` | api ~42 MiB, worker ~2 MiB idle (см. `/tmp/kadr-memory.txt`) |

## CI (PR → develop)

| Item | Value |
|------|--------|
| **PR** | [#12](https://github.com/dsefremov/kadr/pull/12) |
| **Green CI (final)** | Head **`59a3b07`** — [run 42911354496](https://github.com/dsefremov/kadr/actions/runs/42911354496) (`go-backend`, `web`) |

## Open questions (with proposal)

1. **Uploads list UI** (`upload-queue` mockup) — Proposal: stage 7 lite or fold into viewer milestone.
2. **Purge sweep job** — Proposal: nightly job if clock skew observed in prod.
3. **Lower quota admin** — Proposal: admin quota-edit in stage 7 for support.
4. **HDR HEIC** — Proposal: add CC0 when source found.
5. **Physical wipe guarantees** — Proposal: document RPO/RTO only; no secure erase in v1.
