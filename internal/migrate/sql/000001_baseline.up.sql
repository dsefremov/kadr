-- Stage 0 baseline: schema tracking and minimal job queue stub (product schema comes in later stages).
CREATE TABLE IF NOT EXISTS schema_migrations (
  version BIGINT NOT NULL PRIMARY KEY,
  applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS jobs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  kind VARCHAR(64) NOT NULL,
  payload JSON NOT NULL,
  idem_key VARCHAR(191) NOT NULL,
  state ENUM('pending','running','done','failed','dead') NOT NULL DEFAULT 'pending',
  run_after TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  lease_until TIMESTAMP NULL DEFAULT NULL,
  attempts INT UNSIGNED NOT NULL DEFAULT 0,
  last_error TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_jobs_idem (idem_key),
  KEY idx_jobs_claim (state, run_after)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
