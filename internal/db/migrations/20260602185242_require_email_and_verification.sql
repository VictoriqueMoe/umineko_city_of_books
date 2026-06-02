-- +goose Up
ALTER TABLE users ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN verify_grace_until TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE users SET verify_grace_until = NOW() + INTERVAL '14 days';

CREATE UNIQUE INDEX idx_users_email_unique ON users (LOWER(email)) WHERE email <> '';

CREATE TABLE email_verification_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens (user_id);

-- +goose Down
DROP TABLE email_verification_tokens;
DROP INDEX idx_users_email_unique;
ALTER TABLE users DROP COLUMN verify_grace_until;
ALTER TABLE users DROP COLUMN email_verified;
