-- Stage 6: trash retention (spec v0.3 §15)

ALTER TABLE media
  ADD COLUMN purge_after TIMESTAMP NULL DEFAULT NULL AFTER trashed_at,
  ADD KEY idx_media_purge (trashed_at, purge_after);
