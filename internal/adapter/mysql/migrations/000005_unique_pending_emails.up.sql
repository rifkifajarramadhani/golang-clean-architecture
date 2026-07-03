ALTER TABLE users
  ADD COLUMN pending_email_unique VARCHAR(255)
    GENERATED ALWAYS AS (NULLIF(pending_email, '')) STORED,
  ADD UNIQUE INDEX ux_users_pending_email (pending_email_unique);
