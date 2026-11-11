-- Den 2026-11-11: default quota 1 GB (decimal) for new users; per-user quota_bytes unchanged on existing rows.
CREATE TABLE IF NOT EXISTS system_settings (
  k VARCHAR(64) NOT NULL,
  v BIGINT UNSIGNED NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (k)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO system_settings (k, v) VALUES ('default_quota_bytes', 1000000000);
