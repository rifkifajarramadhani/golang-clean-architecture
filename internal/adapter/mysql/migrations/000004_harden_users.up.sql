ALTER TABLE users
  ADD COLUMN role VARCHAR(16) NOT NULL DEFAULT 'user' AFTER password,
  ADD COLUMN email_verified_at TIMESTAMP NULL AFTER role,
  ADD COLUMN pending_email VARCHAR(255) NOT NULL DEFAULT '' AFTER email_verified_at,
  ADD COLUMN token_version INT NOT NULL DEFAULT 1 AFTER pending_email;

CREATE TABLE email_verification_tokens (
  id INT PRIMARY KEY AUTO_INCREMENT,
  user_id INT NOT NULL UNIQUE,
  token_hash VARCHAR(64) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_email_verification_tokens_expires_at (expires_at),
  CONSTRAINT fk_email_verification_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
