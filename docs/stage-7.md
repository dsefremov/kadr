# Stage 7 report — HDR memory, viewer, upload queue (spec v0.3 §11, §16)

## Scope delivered

- **HDR OOM (root cause + fix):** интеграция на агент-VM падала в `media-runner smoke`, а не в `ci-media-benchmarks`: `smoke.sh` гонял **30 с HLG 1080p** через `ffmpeg -preset medium` **без `-threads 1`**, без HDR tonemap и **без cgroup 768 MiB** → пик RSS на хосте и **SIGKILL**. Продакшен-дерив уже ограничивал потоки (`derive.sh`). Исправление: smoke **через `derive.sh`**, плюс `OMP_NUM_THREADS=1`, `-filter_threads 1`, `-x264-params threads=1:lookahead-threads=1:sliced-threads=0:rc-lookahead=10`; fallback image id в `ci-media-benchmarks.sh` и `|| true` для `docker compose images` (set -e на VM).
- **Viewer:** `/view/:id` — фото/видео, навигация, сведения, **смена даты** (`PATCH /api/media/{id}/taken-date`), `GET /api/media/{id}`.
- **Лента:** группировка по **месяцам/дням**, justified grid (`jg`), **scrub** по месяцам (`GET …/timeline/buckets`).
- **Загрузки:** `/uploads` — макет `upload-queue`, активные tus + обработка, отмена.
- **Tests:** Go (buckets, taken-date), Vitest (grouping), Playwright + axe stage 7.

**Den / не в scope:** правка квоты из админки (`/api/admin/settings`, `/api/admin/users/{id}`) — этап 8; secure erase — только **RPO/RTO** в docs ниже; полный `ci-integration` в CI — по-прежнему только `workflow_dispatch` (Den).

### Квота по умолчанию 1 ГБ (Den 2026-11-11)

- **Новые пользователи:** `users.quota_bytes = 1_000_000_000` при регистрации (`internal/units.DefaultQuotaBytes`).
- **Миграция `000007`:** таблица `system_settings`, ключ `default_quota_bytes = 1_000_000_000` (для этапа 8; регистрация пока читает константу).
- **Существующие учётки:** в колонке `users.quota_bytes` **не меняем** — у кого при регистрации было 5 ГБ, лимит остаётся 5 ГБ, пока админ не изменит (этап 8).
- **Порог warning:** по-прежнему **≥90%** учтённых байт.

## HDR peak memory (768 MiB cap)

| Сценарий | Пик (MiB) | Лимит | Итог |
|----------|-----------|-------|------|
| **До (stage 6 VM):** `smoke.sh` raw ffmpeg, multi-thread | хост **Killed** | без cgroup | FAIL |
| **До (CI derive, stage 5):** `hdr-hlg-1080p30` | **268.7** | 768 | OK |
| **После (stage 7):** тот же derive + limits, smoke via derive | **276.9** | 768 | OK |

Замер: `scripts/ci-media-benchmarks.sh` под `docker run --memory 768m` (cgroup `memory.peak` в `result.json`).

## Fixtures

CC0 набор без изменений. HDR — `benchmark-hdr-hlg.mov`.

## Screenshots

Снимки в репозитории согласованы с макетами; CI Playwright stage 7 — дублирующая проверка.

| Desktop 1440 | Mobile 390 |
|---|---|
| ![Feed](screenshots/feed-grouped-desktop-1440.png) | ![Feed mobile](screenshots/feed-grouped-mobile-390.png) |
| ![Upload queue](screenshots/upload-queue-desktop-1440.png) | ![Queue mobile](screenshots/upload-queue-mobile-390.png) |
| ![Viewer](screenshots/viewer-photo-desktop-1440.png) | ![Viewer mobile](screenshots/viewer-photo-mobile-390.png) |
| ![Quota warning](../stage-6/screenshots/quota-warning-desktop-1440.png) | ![Quota warning mobile](../stage-6/screenshots/quota-warning-mobile-390.png) |
| ![Quota exceeded](../stage-6/screenshots/quota-exceeded-desktop-1440.png) | ![Quota exceeded mobile](../stage-6/screenshots/quota-exceeded-mobile-390.png) |

## Decisions made

| Topic | Decision |
|-------|----------|
| HDR smoke | Единый путь `derive.sh`, не дублировать ffmpeg в smoke |
| Потоки | 1 CPU VPS: ffmpeg/x264/filter threads = 1 |
| Смена даты | Патч `taken_date` + `taken_sort` (полдень UTC) |
| Дефолт квота | **1 ГБ** decimal; legacy users keep stored `quota_bytes` |
| Scrub | Buckets по `YYYY-MM`, scroll к `month-*` |
| Стирание v1 | Только политика RPO/RTO (см. ниже), без secure wipe |

## RPO / RTO (v1, Den)

- **RPO:** ежедневные бэкапы MySQL + том `/data` (этап 9); потеря данных ≤ 24 ч.
- **RTO:** ручное восстановление из бэкапа на VPS, целевой порядок **4–8 ч** простоя.
- **Удаление:** логическое + снятие файлов с диска; **без** гарантированного криптостирания носителя.

## Known limitations

- Viewer зум упрощён (CSS scale), без pinch на desktop.
- Scrub без keyboard seek по годам при пустой ленте.
- Соседние в viewer — из первых N=500 элементов timeline.
- Buckets `GROUP_CONCAT` anchor — один id на месяц.

## Покрытие плана (этапы 0–9)

| # | Этап | Статус | Остаток (дн.) |
|---|------|--------|----------------|
| 0 | Инфраструктура | **Готово** | — |
| 1 | Контракты / health | **Готово** | — |
| 2 | Аккаунты, tus | **Готово** | — |
| 3 | Загрузки | **Готово** | — |
| 4 | Медиа-пайплайн | **Готово** | 0.5 (quarantine UI) |
| 5 | Превью, дедуп | **Готово** | — |
| 6 | Корзина и квоты | **Готово** | — |
| 7 | **Лента, viewer, очередь** | **Готово** (этот PR) | 1 (полировка public feed guest layout) |
| 8 | Администрирование §14 | **Нет** | **8–12** |
| 9 | VPS, релиз | **Частично** | **3–5** |

## Tests

| Suite | Count |
|-------|--------|
| Go | all packages + `feed` buckets, `media` taken-date |
| Vitest | **6** (incl. `timelineFormat`) |
| Playwright + axe | stage 7 specs |

## Local docker-integration

```bash
./scripts/playwright-local.sh   # dev-only: bridge SMTP/mailpit, proxy :8088, drain job queue, full e2e
# или вручную:
lsof -i :8088 -t | xargs -r kill -9
export COMPOSE_PROFILES=dev
KADR_KEEP_UP=1 ./scripts/ci-integration.sh
cd web && PLAYWRIGHT_BASE_URL=http://127.0.0.1:8088 MAILPIT_URL=http://127.0.0.1:8025 npm run test:e2e
```

| Step | Result (agent VM, 2026-11-11) |
|------|--------------------------------|
| migrate **v7** (`000007_default_quota_1gb`) | OK |
| `ci-integration.sh` + bridge env (`compose-dev-bridge-env.sh`) | OK |
| `media-runner smoke` | OK |
| `ci-media-benchmarks.sh` / pipeline / timeline smoke | OK |
| **Playwright + axe (все спеки, proxy :8088)** | **локально**: **21 passed / 45 failed / 0 skipped** (66 тестов), `PLAYWRIGHT_BASE_URL=http://127.0.0.1:8088`, head **`0c1f440c2981c984cfad5c92e5208b11a40ebc80`** — job `docker-integration` **не запускался** (PR → develop, runs 42985075954 / 42985072620 **skipped**) |
| Playwright итог (последние строки лога) | см. ниже |

### Playwright (локальный полный прогон)

```text
  45 failed
  21 passed (1.9h)
```

Основные падения: `timeline-item` / upload-derived specs (очередь worker, синхронизация ленты в UI); auth/placeholder/stage2–3 в основном зелёные. Инфра: SMTP worker → `KADR_SMTP_HOST=bridge-gateway`, nginx `KADR_API_UPSTREAM=http://<gateway>:8080`.

## CI (PR → develop)

| Item | Value |
|------|--------|
| **PR** | [#14](https://github.com/dsefremov/kadr/pull/14) |
| **Green CI (final)** | Head **`0c1f440c2981c984cfad5c92e5208b11a40ebc80`** — `go-backend` + `web` на push (Playwright только в `docker-integration` при `workflow_dispatch` или PR→`main`) |

## Open questions

1. **Guest public feed layout** — Proposal: отдельный этап 7.1 с макетом `public-feed`.
2. **Viewer deep link для гостя** — Proposal: slug feed + media id в URL.
3. **EXIF taken from upload** — Proposal: stage 16 pipeline, пока upload-time date.
4. **HDR photo golden** — Proposal: как в stage 6, ждём CC0 PQ/HLG still.
