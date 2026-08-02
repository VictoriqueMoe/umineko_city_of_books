-- +goose Up
UPDATE audit_log
SET details = regexp_replace(details, '^room=[0-9a-fA-F-]{36} user=[0-9a-fA-F-]{36} ', '')
WHERE action LIKE 'chat_word_filter_%'
  AND details ~ '^room=[0-9a-fA-F-]{36} user=[0-9a-fA-F-]{36} ';

UPDATE audit_log
SET details = regexp_replace(details, '^target=[0-9a-fA-F-]{36} ', '')
WHERE action = 'chat_room_ban'
  AND details ~ '^target=[0-9a-fA-F-]{36} ';

UPDATE audit_log
SET details = ''
WHERE action = 'chat_room_unban'
  AND details ~ '^target=[0-9a-fA-F-]{36}$';

-- +goose Down
SELECT 1;
