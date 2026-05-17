-- +goose Up

INSERT INTO vanity_roles (id, label, color, is_system, sort_order) VALUES
    ('system_top_othello', 'Discmaster', '#34d399', TRUE, 4);

-- +goose Down

DELETE FROM vanity_roles WHERE id = 'system_top_othello';
