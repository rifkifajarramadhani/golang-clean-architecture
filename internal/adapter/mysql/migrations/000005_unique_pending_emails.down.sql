ALTER TABLE users
  DROP INDEX ux_users_pending_email,
  DROP COLUMN pending_email_unique;
