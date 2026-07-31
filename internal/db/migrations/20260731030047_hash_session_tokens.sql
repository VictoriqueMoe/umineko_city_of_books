-- +goose Up

DELETE FROM sessions WHERE expires_at < NOW();

UPDATE sessions SET token = encode(sha256(token::bytea), 'hex');

-- +goose Down

DELETE FROM sessions;
