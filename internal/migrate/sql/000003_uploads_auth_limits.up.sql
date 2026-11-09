CREATE TABLE IF NOT EXISTS uploads (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  public_id CHAR(26) NOT NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  kind_declared ENUM('photo', 'video') NOT NULL,
  upload_length BIGINT UNSIGNED NOT NULL,
  offset_bytes BIGINT UNSIGNED NOT NULL DEFAULT 0,
  incoming_key VARCHAR(255) NOT NULL,
  state ENUM('uploading', 'complete', 'expired', 'cancelled') NOT NULL DEFAULT 'uploading',
  last_activity_at TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_uploads_public (public_id),
  KEY idx_uploads_owner (owner_id, state),
  KEY idx_uploads_expire (state, expires_at),
  CONSTRAINT fk_uploads_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE,
  CONSTRAINT chk_upload_offset CHECK (offset_bytes <= upload_length)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auth_failures (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL,
  ip VARBINARY(16) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_auth_failures_email (email, created_at),
  KEY idx_auth_failures_ip (ip, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS auth_resend_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL,
  ip VARBINARY(16) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_resend_email (email, created_at),
  KEY idx_resend_ip (ip, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
