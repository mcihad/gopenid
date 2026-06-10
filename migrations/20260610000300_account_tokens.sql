----------UP----------
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS account_tokens (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash varchar(120) NOT NULL,
  type varchar(40) NOT NULL,
  expires_at timestamptz NOT NULL,
  used_at timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_account_tokens_hash ON account_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_account_tokens_user_type ON account_tokens(user_id, type);
CREATE INDEX IF NOT EXISTS idx_account_tokens_expires ON account_tokens(expires_at);

----------DOWN----------
DROP TABLE IF EXISTS account_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
