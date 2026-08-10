-- +goose Up
ALTER TABLE mysteries ADD COLUMN knox_contract_published BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE mysteries DROP COLUMN knox_contract_published;
