-- Stage 4: media pipeline (spec v0.3 §5, §8–11)

CREATE TABLE IF NOT EXISTS media (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_id CHAR(26) NOT NULL,
  feed_id BIGINT UNSIGNED NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  upload_id BIGINT UNSIGNED NULL,
  kind ENUM('photo', 'video') NOT NULL,
  dedup_sha256 BINARY(32) NOT NULL,
  source_bytes BIGINT UNSIGNED NOT NULL,
  state ENUM('processing', 'waiting_quota', 'ready', 'failed') NOT NULL DEFAULT 'processing',
  stored_bytes BIGINT UNSIGNED NOT NULL DEFAULT 0,
  width INT UNSIGNED NULL,
  height INT UNSIGNED NULL,
  duration_ms INT UNSIGNED NULL,
  taken_sort DATETIME(6) NOT NULL,
  taken_date DATE NOT NULL,
  src_kind ENUM('exif', 'exif_floating', 'video_utc', 'upload') NOT NULL DEFAULT 'upload',
  error_code VARCHAR(64) NULL,
  error_message TEXT NULL,
  trashed_at TIMESTAMP NULL DEFAULT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_media_public (public_id),
  UNIQUE KEY uk_media_dedup (owner_id, dedup_sha256, source_bytes),
  KEY idx_media_timeline (feed_id, state, taken_sort, id),
  CONSTRAINT fk_media_feed FOREIGN KEY (feed_id) REFERENCES feeds (id) ON DELETE CASCADE,
  CONSTRAINT fk_media_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT fk_media_upload FOREIGN KEY (upload_id) REFERENCES uploads (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS media_variants (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  media_id BIGINT UNSIGNED NOT NULL,
  variant VARCHAR(32) NOT NULL,
  profile_hash CHAR(16) NOT NULL,
  storage_key VARCHAR(255) NOT NULL,
  bytes BIGINT UNSIGNED NOT NULL,
  sha256 BINARY(32) NOT NULL,
  width INT UNSIGNED NULL,
  height INT UNSIGNED NULL,
  mime VARCHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_variant_profile (media_id, variant, profile_hash),
  CONSTRAINT fk_variants_media FOREIGN KEY (media_id) REFERENCES media (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS source_files (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  upload_id BIGINT UNSIGNED NOT NULL,
  media_id BIGINT UNSIGNED NULL,
  incoming_key VARCHAR(255) NOT NULL,
  state ENUM('held', 'deleting', 'deleted') NOT NULL DEFAULT 'held',
  delete_after TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_source_upload (upload_id),
  CONSTRAINT fk_source_upload FOREIGN KEY (upload_id) REFERENCES uploads (id) ON DELETE CASCADE,
  CONSTRAINT fk_source_media FOREIGN KEY (media_id) REFERENCES media (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
