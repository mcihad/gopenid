----------UP----------
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret varchar(80) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled boolean NOT NULL DEFAULT false;

----------DOWN----------
ALTER TABLE users DROP COLUMN IF EXISTS mfa_enabled;
ALTER TABLE users DROP COLUMN IF EXISTS totp_secret;
