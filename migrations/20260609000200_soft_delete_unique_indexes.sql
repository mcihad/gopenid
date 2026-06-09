----------UP----------
ALTER TABLE departments DROP CONSTRAINT IF EXISTS departments_name_key;
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_key;
ALTER TABLE clients DROP CONSTRAINT IF EXISTS clients_client_id_key;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;
ALTER TABLE client_roles DROP CONSTRAINT IF EXISTS client_roles_client_id_name_key;
ALTER TABLE auth_codes DROP CONSTRAINT IF EXISTS auth_codes_code_key;
ALTER TABLE signing_keys DROP CONSTRAINT IF EXISTS signing_keys_key_id_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_name_active ON departments(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_name_active ON roles(name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_clients_client_id_active ON clients(client_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active ON users(email) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_client_roles_client_id_name_active ON client_roles(client_id, name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_codes_code_active ON auth_codes(code) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_signing_keys_key_id_active ON signing_keys(key_id) WHERE deleted_at IS NULL;

----------DOWN----------
DROP INDEX IF EXISTS idx_signing_keys_key_id_active;
DROP INDEX IF EXISTS idx_auth_codes_code_active;
DROP INDEX IF EXISTS idx_client_roles_client_id_name_active;
DROP INDEX IF EXISTS idx_users_email_active;
DROP INDEX IF EXISTS idx_clients_client_id_active;
DROP INDEX IF EXISTS idx_roles_name_active;
DROP INDEX IF EXISTS idx_departments_name_active;

ALTER TABLE departments ADD CONSTRAINT departments_name_key UNIQUE(name);
ALTER TABLE roles ADD CONSTRAINT roles_name_key UNIQUE(name);
ALTER TABLE clients ADD CONSTRAINT clients_client_id_key UNIQUE(client_id);
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE(email);
ALTER TABLE client_roles ADD CONSTRAINT client_roles_client_id_name_key UNIQUE(client_id, name);
ALTER TABLE auth_codes ADD CONSTRAINT auth_codes_code_key UNIQUE(code);
ALTER TABLE signing_keys ADD CONSTRAINT signing_keys_key_id_key UNIQUE(key_id);
