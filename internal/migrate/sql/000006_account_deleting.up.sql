-- Stage 6: account deletion lifecycle (spec §15)

ALTER TABLE users
  MODIFY COLUMN status ENUM('active', 'blocked', 'deleting') NOT NULL DEFAULT 'active';
