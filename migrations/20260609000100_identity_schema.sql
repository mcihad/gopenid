----------UP----------
-- ============================================================================
-- gOpenID identity schema (consolidated)
-- ============================================================================

CREATE TABLE departments (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  name varchar(120) NOT NULL,
  description varchar(500) NOT NULL DEFAULT ''
);

CREATE TABLE roles (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  name varchar(80) NOT NULL,
  description varchar(500) NOT NULL DEFAULT ''
);

CREATE TABLE groups (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  name varchar(120) NOT NULL,
  description varchar(500) NOT NULL DEFAULT ''
);

CREATE TABLE clients (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  client_id varchar(100) NOT NULL,
  client_secret varchar(255) NOT NULL,
  name varchar(140) NOT NULL,
  description varchar(500) NOT NULL DEFAULT '',
  home_url varchar(500) NOT NULL DEFAULT '',
  logo_url varchar(500) NOT NULL DEFAULT '',
  redirect_uris varchar(1000) NOT NULL,
  token_ttl_seconds integer NOT NULL DEFAULT 0,
  refresh_ttl_seconds integer NOT NULL DEFAULT 0
);

CREATE TABLE users (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  email varchar(180) NOT NULL,
  name varchar(140) NOT NULL,
  password_hash varchar(255) NOT NULL,
  active boolean NOT NULL DEFAULT true,
  blocked boolean NOT NULL DEFAULT false,
  blocked_reason varchar(500) NOT NULL DEFAULT '',
  phone varchar(40) NOT NULL DEFAULT '',
  title varchar(140) NOT NULL DEFAULT '',
  avatar_url varchar(500) NOT NULL DEFAULT '',
  last_login_at timestamptz,
  department_id bigint REFERENCES departments(id) ON DELETE SET NULL
);

CREATE TABLE client_roles (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  client_id bigint NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  name varchar(80) NOT NULL,
  description varchar(500) NOT NULL DEFAULT ''
);

CREATE TABLE user_roles (
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id bigint NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  PRIMARY KEY(user_id, role_id)
);

CREATE TABLE user_departments (
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  department_id bigint NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
  PRIMARY KEY(user_id, department_id)
);

CREATE TABLE user_groups (
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  group_id bigint NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  PRIMARY KEY(user_id, group_id)
);

CREATE TABLE user_authorized_clients (
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id bigint NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  PRIMARY KEY(user_id, client_id)
);

CREATE TABLE user_client_roles (
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_role_id bigint NOT NULL REFERENCES client_roles(id) ON DELETE CASCADE,
  PRIMARY KEY(user_id, client_role_id)
);

-- ----------------------------------------------------------------------------
-- Login policies (IP and time based, allow/deny, hierarchical assignment)
-- ----------------------------------------------------------------------------
CREATE TABLE policies (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  name varchar(140) NOT NULL,
  description varchar(500) NOT NULL DEFAULT '',
  type varchar(20) NOT NULL,
  effect varchar(20) NOT NULL,
  ip_cidrs varchar(1000) NOT NULL DEFAULT '',
  days_of_week varchar(40) NOT NULL DEFAULT '',
  start_time varchar(5) NOT NULL DEFAULT '',
  end_time varchar(5) NOT NULL DEFAULT ''
);

CREATE TABLE policy_assignments (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  policy_id bigint NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
  subject_type varchar(20) NOT NULL,
  subject_id bigint NOT NULL
);

-- ----------------------------------------------------------------------------
-- Tokens and sessions
-- ----------------------------------------------------------------------------
CREATE TABLE refresh_tokens (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  token_hash varchar(120) NOT NULL,
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id varchar(120) NOT NULL DEFAULT '',
  scope varchar(250) NOT NULL DEFAULT '',
  expires_at timestamptz NOT NULL,
  revoked boolean NOT NULL DEFAULT false,
  revoked_at timestamptz
);

CREATE TABLE revoked_tokens (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  jti varchar(120) NOT NULL,
  user_id bigint NOT NULL DEFAULT 0,
  reason varchar(250) NOT NULL DEFAULT '',
  expires_at timestamptz NOT NULL
);

-- ----------------------------------------------------------------------------
-- Audit log
-- ----------------------------------------------------------------------------
CREATE TABLE audit_logs (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  user_id bigint,
  email varchar(180) NOT NULL DEFAULT '',
  client_id varchar(120) NOT NULL DEFAULT '',
  event varchar(40) NOT NULL,
  success boolean NOT NULL DEFAULT true,
  message varchar(500) NOT NULL DEFAULT '',
  ip varchar(64) NOT NULL DEFAULT '',
  user_agent varchar(500) NOT NULL DEFAULT '',
  device varchar(80) NOT NULL DEFAULT '',
  browser varchar(80) NOT NULL DEFAULT '',
  os varchar(80) NOT NULL DEFAULT ''
);

CREATE TABLE auth_codes (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  code varchar(96) NOT NULL,
  user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  client_id varchar(120) NOT NULL,
  redirect_uri varchar(500) NOT NULL,
  scope varchar(250) NOT NULL DEFAULT '',
  nonce varchar(250) NOT NULL DEFAULT '',
  code_challenge varchar(250) NOT NULL DEFAULT '',
  code_challenge_method varchar(20) NOT NULL DEFAULT '',
  used boolean NOT NULL DEFAULT false
);

CREATE TABLE signing_keys (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  key_id varchar(120) NOT NULL,
  private_pem text NOT NULL,
  active boolean NOT NULL DEFAULT true
);

-- ----------------------------------------------------------------------------
-- Indexes
-- ----------------------------------------------------------------------------
CREATE INDEX idx_users_department_id ON users(department_id);
CREATE INDEX idx_auth_codes_code_used ON auth_codes(code, used);
CREATE INDEX idx_client_roles_client_id ON client_roles(client_id);
CREATE INDEX idx_policy_assignments_subject ON policy_assignments(subject_type, subject_id);
CREATE INDEX idx_policy_assignments_policy ON policy_assignments(policy_id);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);
CREATE INDEX idx_revoked_tokens_jti ON revoked_tokens(jti);
CREATE INDEX idx_revoked_tokens_expires ON revoked_tokens(expires_at);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_event ON audit_logs(event);

CREATE UNIQUE INDEX idx_departments_name_active ON departments(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_roles_name_active ON roles(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_groups_name_active ON groups(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_clients_client_id_active ON clients(client_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_client_roles_client_id_name_active ON client_roles(client_id, name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_policies_name_active ON policies(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_auth_codes_code_active ON auth_codes(code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_signing_keys_key_id_active ON signing_keys(key_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);

----------DOWN----------
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS revoked_tokens;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS policy_assignments;
DROP TABLE IF EXISTS policies;
DROP TABLE IF EXISTS signing_keys;
DROP TABLE IF EXISTS auth_codes;
DROP TABLE IF EXISTS user_client_roles;
DROP TABLE IF EXISTS user_authorized_clients;
DROP TABLE IF EXISTS user_groups;
DROP TABLE IF EXISTS user_departments;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS client_roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS departments;
