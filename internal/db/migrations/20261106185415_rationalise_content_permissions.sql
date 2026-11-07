-- +goose Up
-- +goose StatementBegin
INSERT INTO role_permissions (role, permission)
SELECT role,
       CASE permission
           WHEN 'edit_any_theory' THEN 'edit_any_work'
           WHEN 'delete_any_theory' THEN 'delete_any_work'
           WHEN 'edit_any_comment' THEN 'edit_any_post'
           WHEN 'delete_any_comment' THEN 'delete_any_post'
           WHEN 'delete_any_response' THEN 'delete_any_post'
       END
FROM role_permissions
WHERE permission IN ('edit_any_theory', 'delete_any_theory', 'edit_any_comment', 'delete_any_comment', 'delete_any_response')
ON CONFLICT DO NOTHING;

DELETE FROM role_permissions
WHERE permission IN ('edit_any_theory', 'delete_any_theory', 'edit_any_comment', 'delete_any_comment', 'delete_any_response', 'edit_any_journal');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
INSERT INTO role_permissions (role, permission)
SELECT role, mapped.permission
FROM role_permissions
JOIN (VALUES
    ('edit_any_work', 'edit_any_theory'),
    ('delete_any_work', 'delete_any_theory'),
    ('edit_any_post', 'edit_any_comment'),
    ('delete_any_post', 'delete_any_comment'),
    ('delete_any_post', 'delete_any_response')
) AS mapped(current_permission, permission) ON mapped.current_permission = role_permissions.permission
ON CONFLICT DO NOTHING;

DELETE FROM role_permissions
WHERE permission IN ('edit_any_work', 'delete_any_work');
-- +goose StatementEnd
