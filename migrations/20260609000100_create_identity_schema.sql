----------UP----------
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

CREATE TABLE clients (
  id bigserial PRIMARY KEY,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  client_id varchar(100) NOT NULL,
  client_secret varchar(255) NOT NULL,
  name varchar(140) NOT NULL,
  redirect_uris varchar(1000) NOT NULL
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

CREATE INDEX idx_users_department_id ON users(department_id);
CREATE INDEX idx_auth_codes_code_used ON auth_codes(code, used);
CREATE INDEX idx_client_roles_client_id ON client_roles(client_id);
CREATE UNIQUE INDEX idx_departments_name_active ON departments(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_roles_name_active ON roles(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_clients_client_id_active ON clients(client_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_client_roles_client_id_name_active ON client_roles(client_id, name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_auth_codes_code_active ON auth_codes(code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_signing_keys_key_id_active ON signing_keys(key_id) WHERE deleted_at IS NULL;

----------DOWN----------
DROP TABLE IF EXISTS signing_keys;
DROP TABLE IF EXISTS auth_codes;
DROP TABLE IF EXISTS user_client_roles;
DROP TABLE IF EXISTS user_authorized_clients;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS client_roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS departments;
