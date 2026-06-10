----------UP----------
ALTER TABLE departments ADD COLUMN IF NOT EXISTS parent_id bigint REFERENCES departments(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id);

ALTER TABLE clients ADD COLUMN IF NOT EXISTS allow_password_grant boolean NOT NULL DEFAULT false;

ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_count integer NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until timestamptz;
CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users(locked_until);

----------DOWN----------
DROP INDEX IF EXISTS idx_users_locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS failed_login_count;

ALTER TABLE clients DROP COLUMN IF EXISTS allow_password_grant;

DROP INDEX IF EXISTS idx_departments_parent_id;
ALTER TABLE departments DROP COLUMN IF EXISTS parent_id;
