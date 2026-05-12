ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
ALTER TABLE users DROP COLUMN IF EXISTS phone_verified;
ALTER TABLE users DROP COLUMN IF EXISTS nickname;
DROP TABLE IF EXISTS email_verifications;
DROP TABLE IF EXISTS phone_verifications;
