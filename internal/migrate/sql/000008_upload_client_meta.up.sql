-- Client filename / mtime from tus Upload-Metadata (taken_date fallbacks).

ALTER TABLE uploads
  ADD COLUMN client_filename VARCHAR(255) NULL AFTER kind_declared,
  ADD COLUMN client_mtime DATETIME(3) NULL AFTER client_filename;
