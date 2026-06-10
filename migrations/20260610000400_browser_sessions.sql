----------UP----------
CREATE TABLE IF NOT EXISTS browser_sessions (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  token_hash varchar(120) NOT NULL,
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  auth_time timestamptz NOT NULL,
  expires_at timestamptz NOT NULL,
  revoked boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_browser_sessions_hash ON browser_sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_browser_sessions_user ON browser_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_browser_sessions_expires ON browser_sessions(expires_at);

----------DOWN----------
DROP TABLE IF EXISTS browser_sessions;
