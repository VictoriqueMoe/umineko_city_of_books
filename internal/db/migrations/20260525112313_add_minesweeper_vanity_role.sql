-- +goose Up

INSERT INTO vanity_roles (id, label, color, is_system, sort_order) VALUES
    ('system_top_minesweeper', 'Minemaster', '#f87171', TRUE, 5);

-- +goose Down

DELETE FROM vanity_roles WHERE id = 'system_top_minesweeper';
