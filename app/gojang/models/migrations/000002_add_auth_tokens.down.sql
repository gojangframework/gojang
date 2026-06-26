DROP INDEX IF EXISTS idx_users_password_reset_token;
DROP INDEX IF EXISTS idx_users_email_verification_token;
ALTER TABLE users DROP COLUMN password_reset_expiry;
ALTER TABLE users DROP COLUMN password_reset_token;
ALTER TABLE users DROP COLUMN email_verification_expiry;
ALTER TABLE users DROP COLUMN email_verification_token;
ALTER TABLE users DROP COLUMN is_email_verified;
