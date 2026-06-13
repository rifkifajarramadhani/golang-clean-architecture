DROP TABLE IF EXISTS email_verification_tokens;

ALTER TABLE users
  DROP COLUMN token_version,
  DROP COLUMN pending_email,
  DROP COLUMN email_verified_at,
  DROP COLUMN role;
