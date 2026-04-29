-- +goose Up

INSERT INTO vanity_roles (id, label, color, is_system, sort_order) VALUES
    ('system_top_chess', 'Grandmaster', '#facc15', TRUE, 2),
    ('system_top_checkers', 'King of the Board', '#a78bfa', TRUE, 3);

-- +goose Down

DELETE FROM vanity_roles WHERE id IN ('system_top_chess', 'system_top_checkers');
