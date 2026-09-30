-- +goose Up
CREATE TABLE IF NOT EXISTS email_codes (
    user_id    BIGINT       NOT NULL,
    purpose    VARCHAR(16)  NOT NULL CHECK (purpose IN ('register', 'login')),
    code_hash  BYTEA        NOT NULL,
    attempts   INTEGER      NOT NULL DEFAULT 0,
    expires_at TIMESTAMP(0) NOT NULL,
    created_at TIMESTAMP(0) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_email_codes_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

COMMENT ON TABLE email_codes IS 'Temporary email verification and login codes';

COMMENT ON COLUMN email_codes.user_id IS 'User associated with the email code';
COMMENT ON COLUMN email_codes.purpose IS 'Purpose of the code: register or login';
COMMENT ON COLUMN email_codes.code_hash IS 'Hash of the email verification or login code';
COMMENT ON COLUMN email_codes.attempts IS 'Number of failed attempts to verify the code';
COMMENT ON COLUMN email_codes.expires_at IS 'Timestamp when the code expires';
COMMENT ON COLUMN email_codes.created_at IS 'Timestamp when the code was created';
COMMENT ON CONSTRAINT fk_email_codes_user ON email_codes IS 'Links the email code to its user and deletes codes when the user is deleted';

CREATE UNIQUE INDEX idx_email_codes_user_purpose ON email_codes (user_id, purpose);

COMMENT ON INDEX idx_email_codes_user_purpose IS 'Ensures at most one code per user and purpose';

-- +goose Down
DROP INDEX IF EXISTS idx_email_codes_user_purpose;
DROP TABLE IF EXISTS email_codes;