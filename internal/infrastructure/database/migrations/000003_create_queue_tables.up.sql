CREATE TABLE queue_jobs (
  id VARCHAR(255) PRIMARY KEY,
  queue VARCHAR(100) NOT NULL,
  type VARCHAR(255) NOT NULL,
  payload JSON NOT NULL,
  status VARCHAR(20) NOT NULL,
  attempts INT NOT NULL DEFAULT 0,
  max_retry INT NOT NULL DEFAULT 25,
  timeout_seconds INT NOT NULL DEFAULT 0,
  retention_secs INT NOT NULL DEFAULT 0,
  available_at DATETIME(6) NOT NULL,
  lease_token VARCHAR(36) NULL,
  leased_until DATETIME(6) NULL,
  last_error TEXT NULL,
  last_failed_at DATETIME(6) NULL,
  completed_at DATETIME(6) NULL,
  expires_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  INDEX idx_queue_jobs_reserve (queue, status, available_at),
  INDEX idx_queue_jobs_lease (status, leased_until),
  INDEX idx_queue_jobs_expiry (status, expires_at)
);

CREATE TABLE queue_locks (
  lock_key VARCHAR(191) PRIMARY KEY,
  job_id VARCHAR(255) NOT NULL,
  expires_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  INDEX idx_queue_locks_job_id (job_id),
  INDEX idx_queue_locks_expires_at (expires_at)
);

CREATE TABLE queue_stats (
  queue VARCHAR(100) PRIMARY KEY,
  processed_total BIGINT NOT NULL DEFAULT 0,
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6)
);
