UPDATE users SET status = 'active' WHERE status = 'deleting';
ALTER TABLE users
  MODIFY COLUMN status ENUM('active', 'blocked') NOT NULL DEFAULT 'active';
