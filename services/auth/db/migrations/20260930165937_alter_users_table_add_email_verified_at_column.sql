-- +goose Up
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

COMMENT ON COLUMN users.email_verified_at IS 'Timestamp when the user email was verified';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;

