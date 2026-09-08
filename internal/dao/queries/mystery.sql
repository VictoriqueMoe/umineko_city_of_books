-- name: CreateMystery :one
WITH m AS (
    INSERT INTO mysteries (user_id, title, body, difficulty, free_for_all, keep_open_after_solve,
        knox_culprit_named_early, knox_no_supernatural, knox_passages_declared, knox_no_unknown_poison, knox_no_outsider,
        knox_no_lucky_accident, knox_detective_not_culprit, knox_clues_shown, knox_narrator_hides_nothing, knox_no_unannounced_twins, knox_contract_published)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, TRUE)
    RETURNING id, user_id, title, body, difficulty, solved, paused, gm_away, free_for_all, keep_open_after_solve,
        knox_culprit_named_early, knox_no_supernatural, knox_passages_declared, knox_no_unknown_poison, knox_no_outsider,
        knox_no_lucky_accident, knox_detective_not_culprit, knox_clues_shown, knox_narrator_hides_nothing, knox_no_unannounced_twins,
        knox_contract_published, winner_id, solved_at, paused_at, paused_duration_seconds, created_at, updated_at
)
SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve,
       m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider,
       m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins,
       m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
       u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       w.id AS winner_id, w.username AS winner_username, w.display_name AS winner_display_name, w.avatar_url AS winner_avatar_url,
       COALESCE(wr.role, '')::text AS winner_role,
       (SELECT COUNT(*) FROM mystery_attempts ma WHERE ma.mystery_id = m.id AND ma.parent_id IS NULL AND ma.user_id != m.user_id) AS attempt_count,
       (SELECT COUNT(*) FROM mystery_clues mc WHERE mc.mystery_id = m.id) AS clue_count,
       (SELECT COUNT(DISTINCT ms.user_id) FROM mystery_attempts ms WHERE ms.mystery_id = m.id AND ms.is_winner = TRUE) AS solver_count
FROM m
JOIN users u ON m.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN users w ON m.winner_id = w.id
LEFT JOIN user_roles wr ON wr.user_id = w.id;

-- name: GetMysteryByID :one
SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve,
       m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider,
       m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins,
       m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
       u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       w.id AS winner_id, w.username AS winner_username, w.display_name AS winner_display_name, w.avatar_url AS winner_avatar_url,
       COALESCE(wr.role, '')::text AS winner_role,
       (SELECT COUNT(*) FROM mystery_attempts ma WHERE ma.mystery_id = m.id AND ma.parent_id IS NULL AND ma.user_id != m.user_id) AS attempt_count,
       (SELECT COUNT(*) FROM mystery_clues mc WHERE mc.mystery_id = m.id) AS clue_count,
       (SELECT COUNT(DISTINCT ms.user_id) FROM mystery_attempts ms WHERE ms.mystery_id = m.id AND ms.is_winner = TRUE) AS solver_count
FROM mysteries m
JOIN users u ON m.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN users w ON m.winner_id = w.id
LEFT JOIN user_roles wr ON wr.user_id = w.id
WHERE m.id = $1;

-- name: CountUserMysteries :one
SELECT COUNT(*) FROM mysteries WHERE user_id = $1;

-- name: ListMysteriesByUser :many
SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve,
       m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider,
       m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins,
       m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
       u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       w.id AS winner_id, w.username AS winner_username, w.display_name AS winner_display_name, w.avatar_url AS winner_avatar_url,
       COALESCE(wr.role, '')::text AS winner_role,
       (SELECT COUNT(*) FROM mystery_attempts ma WHERE ma.mystery_id = m.id AND ma.parent_id IS NULL AND ma.user_id != m.user_id) AS attempt_count,
       (SELECT COUNT(*) FROM mystery_clues mc WHERE mc.mystery_id = m.id) AS clue_count,
       (SELECT COUNT(DISTINCT ms.user_id) FROM mystery_attempts ms WHERE ms.mystery_id = m.id AND ms.is_winner = TRUE) AS solver_count
FROM mysteries m
JOIN users u ON m.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN users w ON m.winner_id = w.id
LEFT JOIN user_roles wr ON wr.user_id = w.id
WHERE m.user_id = $1
ORDER BY m.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountMysteriesFiltered :one
SELECT COUNT(*) FROM mysteries m
WHERE (sqlc.narg(solved)::boolean IS NULL OR m.solved = sqlc.narg(solved)::boolean)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR m.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListMysteriesByNew :many
SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve,
       m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider,
       m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins,
       m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
       u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       w.id AS winner_id, w.username AS winner_username, w.display_name AS winner_display_name, w.avatar_url AS winner_avatar_url,
       COALESCE(wr.role, '')::text AS winner_role,
       (SELECT COUNT(*) FROM mystery_attempts ma WHERE ma.mystery_id = m.id AND ma.parent_id IS NULL AND ma.user_id != m.user_id) AS attempt_count,
       (SELECT COUNT(*) FROM mystery_clues mc WHERE mc.mystery_id = m.id) AS clue_count,
       (SELECT COUNT(DISTINCT ms.user_id) FROM mystery_attempts ms WHERE ms.mystery_id = m.id AND ms.is_winner = TRUE) AS solver_count
FROM mysteries m
JOIN users u ON m.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN users w ON m.winner_id = w.id
LEFT JOIN user_roles wr ON wr.user_id = w.id
WHERE (sqlc.narg(solved)::boolean IS NULL OR m.solved = sqlc.narg(solved)::boolean)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR m.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY m.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListMysteriesByOld :many
SELECT m.id, m.user_id, m.title, m.body, m.difficulty, m.solved, m.paused, m.gm_away, m.free_for_all, m.keep_open_after_solve,
       m.knox_culprit_named_early, m.knox_no_supernatural, m.knox_passages_declared, m.knox_no_unknown_poison, m.knox_no_outsider,
       m.knox_no_lucky_accident, m.knox_detective_not_culprit, m.knox_clues_shown, m.knox_narrator_hides_nothing, m.knox_no_unannounced_twins,
       m.knox_contract_published, m.solved_at, m.paused_at, m.paused_duration_seconds, m.created_at, m.updated_at,
       u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       w.id AS winner_id, w.username AS winner_username, w.display_name AS winner_display_name, w.avatar_url AS winner_avatar_url,
       COALESCE(wr.role, '')::text AS winner_role,
       (SELECT COUNT(*) FROM mystery_attempts ma WHERE ma.mystery_id = m.id AND ma.parent_id IS NULL AND ma.user_id != m.user_id) AS attempt_count,
       (SELECT COUNT(*) FROM mystery_clues mc WHERE mc.mystery_id = m.id) AS clue_count,
       (SELECT COUNT(DISTINCT ms.user_id) FROM mystery_attempts ms WHERE ms.mystery_id = m.id AND ms.is_winner = TRUE) AS solver_count
FROM mysteries m
JOIN users u ON m.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
LEFT JOIN users w ON m.winner_id = w.id
LEFT JOIN user_roles wr ON wr.user_id = w.id
WHERE (sqlc.narg(solved)::boolean IS NULL OR m.solved = sqlc.narg(solved)::boolean)
  AND (sqlc.arg(exclude_user_ids)::text = '' OR m.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY m.created_at ASC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: UpdateMystery :execrows
UPDATE mysteries SET title = $1, body = $2, difficulty = $3, updated_at = CURRENT_TIMESTAMP WHERE id = $4 AND user_id = $5;

-- name: UpdateMysteryAsAdmin :exec
UPDATE mysteries SET title = $1, body = $2, difficulty = $3, free_for_all = $4, keep_open_after_solve = $5,
    knox_culprit_named_early = $6, knox_no_supernatural = $7, knox_passages_declared = $8, knox_no_unknown_poison = $9, knox_no_outsider = $10,
    knox_no_lucky_accident = $11, knox_detective_not_culprit = $12, knox_clues_shown = $13, knox_narrator_hides_nothing = $14, knox_no_unannounced_twins = $15, knox_contract_published = TRUE,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $16;

-- name: AddMysteryClue :one
INSERT INTO mystery_clues (mystery_id, body, truth_type, sort_order, player_id) VALUES ($1, $2, $3, $4, $5)
RETURNING id, body, truth_type, sort_order, player_id;

-- name: GetMysteryClues :many
SELECT id, body, truth_type, sort_order, player_id FROM mystery_clues WHERE mystery_id = $1 ORDER BY sort_order ASC;

-- name: DeleteMysteryClues :exec
DELETE FROM mystery_clues WHERE mystery_id = $1 AND player_id IS NULL;

-- name: DeleteMysteryClue :exec
DELETE FROM mystery_clues WHERE id = $1;

-- name: UpdateMysteryClue :exec
UPDATE mystery_clues SET body = $1 WHERE id = $2;

-- name: CountMysteryClues :one
SELECT COUNT(*) FROM mystery_clues WHERE mystery_id = $1;

-- name: CreateMysteryAttempt :one
WITH a AS (
    INSERT INTO mystery_attempts (mystery_id, user_id, parent_id, body) VALUES ($1, $2, $3, $4)
    RETURNING id, mystery_id, user_id, parent_id, body, is_winner, created_at
)
SELECT a.id, a.mystery_id, a.user_id, a.parent_id, a.body, a.is_winner, a.created_at,
       u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       0::bigint AS vote_score,
       0::int AS user_vote
FROM a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id;

-- name: GetMysteryAttempts :many
SELECT a.id, a.mystery_id, a.user_id, a.parent_id, a.body, a.is_winner, a.created_at,
       u.username AS author_username, u.display_name AS author_display_name, u.avatar_url AS author_avatar_url,
       COALESCE(r.role, '')::text AS author_role,
       COALESCE((SELECT SUM(v.value) FROM mystery_attempt_votes v WHERE v.attempt_id = a.id), 0)::bigint AS vote_score,
       COALESCE((SELECT uv.value FROM mystery_attempt_votes uv WHERE uv.attempt_id = a.id AND uv.user_id = $1), 0)::int AS user_vote
FROM mystery_attempts a
JOIN users u ON a.user_id = u.id
LEFT JOIN user_roles r ON r.user_id = u.id
WHERE a.mystery_id = $2
ORDER BY a.created_at ASC;

-- name: DeleteMysteryAttempt :execrows
DELETE FROM mystery_attempts WHERE id = $1 AND user_id = $2;

-- name: DeleteMysteryAttemptAsAdmin :exec
DELETE FROM mystery_attempts WHERE id = $1;

-- name: GetMysteryAttemptAuthorID :one
SELECT user_id FROM mystery_attempts WHERE id = $1;

-- name: GetMysteryAttemptMysteryID :one
SELECT mystery_id FROM mystery_attempts WHERE id = $1;

-- name: GetMysteryAttemptOwner :one
SELECT user_id, mystery_id FROM mystery_attempts WHERE id = $1;

-- name: CountMysteryAttempts :one
SELECT COUNT(*) FROM mystery_attempts WHERE mystery_id = $1;

-- name: SetMysteryWinner :exec
UPDATE mysteries SET solved = TRUE, winner_id = $1, solved_at = NOW() WHERE id = $2;

-- name: SetMysteryAttemptWinner :exec
UPDATE mystery_attempts SET is_winner = TRUE WHERE id = $1;

-- name: MarkMysteryPermanentlySolved :execrows
UPDATE mysteries SET solved = TRUE, solved_at = NOW() WHERE id = $1 AND solved = FALSE;

-- name: UserHasWinningMysteryAttempt :one
SELECT EXISTS(SELECT 1 FROM mystery_attempts a WHERE a.mystery_id = $1 AND a.user_id = $2 AND a.is_winner = TRUE);

-- name: GetMysterySolverIDs :many
SELECT DISTINCT user_id FROM mystery_attempts WHERE mystery_id = $1 AND is_winner = TRUE;

-- name: GetMysteryPlayerIDs :many
SELECT DISTINCT ma.user_id FROM mystery_attempts ma
JOIN mysteries m ON m.id = ma.mystery_id
WHERE ma.mystery_id = $1 AND ma.user_id != m.user_id;

-- name: IsMysterySolved :one
SELECT solved FROM mysteries WHERE id = $1;

-- name: IsMysteryPaused :one
SELECT paused FROM mysteries WHERE id = $1;

-- name: SetMysteryPaused :exec
UPDATE mysteries
SET paused = TRUE,
    paused_at = CASE WHEN paused = TRUE THEN paused_at ELSE NOW() END
WHERE id = $1;

-- name: SetMysteryUnpaused :exec
UPDATE mysteries
SET paused = FALSE,
    paused_duration_seconds = paused_duration_seconds + CASE
        WHEN paused_at IS NOT NULL
        THEN EXTRACT(EPOCH FROM (NOW() - paused_at))::INTEGER
        ELSE 0
    END,
    paused_at = NULL
WHERE id = $1;

-- name: SetMysteryGmAway :exec
UPDATE mysteries SET gm_away = $1 WHERE id = $2;

-- name: GetMysteryLeaderboard :many
SELECT lb.user_id, lb.username, lb.display_name, lb.avatar_url, lb.role, lb.score, lb.easy_solved, lb.medium_solved, lb.hard_solved, lb.nightmare_solved, lb.score_adjustment
FROM (
    SELECT u.id AS user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')::text AS role,
        (COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
            CASE WHEN m.difficulty = 'easy' THEN 2
                 WHEN m.difficulty = 'medium' THEN 4
                 WHEN m.difficulty = 'hard' THEN 6
                 WHEN m.difficulty = 'nightmare' THEN 8
                 ELSE 4 END
        ELSE 0 END), 0) + u.mystery_score_adjustment)::bigint AS score,
        COALESCE(SUM(CASE WHEN m.difficulty = 'easy' THEN 1 ELSE 0 END), 0)::bigint AS easy_solved,
        COALESCE(SUM(CASE WHEN m.difficulty = 'medium' THEN 1 ELSE 0 END), 0)::bigint AS medium_solved,
        COALESCE(SUM(CASE WHEN m.difficulty = 'hard' THEN 1 ELSE 0 END), 0)::bigint AS hard_solved,
        COALESCE(SUM(CASE WHEN m.difficulty = 'nightmare' THEN 1 ELSE 0 END), 0)::bigint AS nightmare_solved,
        u.mystery_score_adjustment::int AS score_adjustment
    FROM users u
    LEFT JOIN mystery_attempts a ON a.user_id = u.id AND a.is_winner = TRUE
    LEFT JOIN mysteries m ON m.id = a.mystery_id
    LEFT JOIN user_roles r ON r.user_id = u.id
    GROUP BY u.id, r.role
    HAVING COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
            CASE WHEN m.difficulty = 'easy' THEN 2
                 WHEN m.difficulty = 'medium' THEN 4
                 WHEN m.difficulty = 'hard' THEN 6
                 WHEN m.difficulty = 'nightmare' THEN 8
                 ELSE 4 END
        ELSE 0 END), 0) + u.mystery_score_adjustment > 0
) AS lb
ORDER BY lb.score DESC, lb.display_name ASC
LIMIT $1;

-- name: GetTopMysteryDetectiveIDs :many
WITH ranked AS (
    SELECT u.id AS user_id,
        (COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
            CASE WHEN m.difficulty = 'easy' THEN 2
                 WHEN m.difficulty = 'medium' THEN 4
                 WHEN m.difficulty = 'hard' THEN 6
                 WHEN m.difficulty = 'nightmare' THEN 8
                 ELSE 4 END
        ELSE 0 END), 0) + u.mystery_score_adjustment)::bigint AS score
    FROM users u
    LEFT JOIN mystery_attempts a ON a.user_id = u.id AND a.is_winner = TRUE
    LEFT JOIN mysteries m ON m.id = a.mystery_id
    GROUP BY u.id
    HAVING COALESCE(SUM(CASE WHEN m.id IS NOT NULL THEN
            CASE WHEN m.difficulty = 'easy' THEN 2
                 WHEN m.difficulty = 'medium' THEN 4
                 WHEN m.difficulty = 'hard' THEN 6
                 WHEN m.difficulty = 'nightmare' THEN 8
                 ELSE 4 END
        ELSE 0 END), 0) + u.mystery_score_adjustment > 0
)
SELECT rk.user_id FROM ranked rk
WHERE rk.score = (SELECT MAX(mx.score) FROM ranked mx);

-- name: GetMysteryGMLeaderboard :many
SELECT gm_lb.user_id, gm_lb.username, gm_lb.display_name, gm_lb.avatar_url, gm_lb.role, gm_lb.score, gm_lb.mystery_count, gm_lb.player_count
FROM (
    SELECT u.id AS user_id, u.username, u.display_name, u.avatar_url, COALESCE(r.role, '')::text AS role,
        (SUM(
            CASE m.difficulty
                WHEN 'easy' THEN 2
                WHEN 'medium' THEN 4
                WHEN 'hard' THEN 6
                WHEN 'nightmare' THEN 8
                ELSE 4
            END
            + LEAST((SELECT COUNT(DISTINCT ap.user_id) FROM mystery_attempts ap WHERE ap.mystery_id = m.id), 5)
        ) + u.gm_score_adjustment)::bigint AS score,
        COUNT(m.id)::bigint AS mystery_count,
        SUM(LEAST((SELECT COUNT(DISTINCT ac.user_id) FROM mystery_attempts ac WHERE ac.mystery_id = m.id), 5))::bigint AS player_count
    FROM mysteries m
    JOIN users u ON m.user_id = u.id
    LEFT JOIN user_roles r ON r.user_id = u.id
    WHERE m.solved = TRUE
    GROUP BY u.id, r.role
    HAVING SUM(
            CASE m.difficulty
                WHEN 'easy' THEN 2
                WHEN 'medium' THEN 4
                WHEN 'hard' THEN 6
                WHEN 'nightmare' THEN 8
                ELSE 4
            END
            + LEAST((SELECT COUNT(DISTINCT ah.user_id) FROM mystery_attempts ah WHERE ah.mystery_id = m.id), 5)
        ) + u.gm_score_adjustment > 0
) AS gm_lb
ORDER BY gm_lb.score DESC, gm_lb.display_name ASC
LIMIT $1;

-- name: GetTopMysteryGMIDs :many
WITH ranked AS (
    SELECT u.id AS user_id,
        (SUM(
            CASE m.difficulty
                WHEN 'easy' THEN 2
                WHEN 'medium' THEN 4
                WHEN 'hard' THEN 6
                WHEN 'nightmare' THEN 8
                ELSE 4
            END
            + LEAST((SELECT COUNT(DISTINCT ap.user_id) FROM mystery_attempts ap WHERE ap.mystery_id = m.id), 5)
        ) + u.gm_score_adjustment)::bigint AS score
    FROM mysteries m
    JOIN users u ON m.user_id = u.id
    WHERE m.solved = TRUE
    GROUP BY u.id
    HAVING SUM(
            CASE m.difficulty
                WHEN 'easy' THEN 2
                WHEN 'medium' THEN 4
                WHEN 'hard' THEN 6
                WHEN 'nightmare' THEN 8
                ELSE 4
            END
            + LEAST((SELECT COUNT(DISTINCT ah.user_id) FROM mystery_attempts ah WHERE ah.mystery_id = m.id), 5)
        ) + u.gm_score_adjustment > 0
)
SELECT rk.user_id FROM ranked rk
WHERE rk.score = (SELECT MAX(mx.score) FROM ranked mx);

-- name: AddMysteryAttachment :one
INSERT INTO mystery_attachments (mystery_id, file_url, file_name, file_size) VALUES ($1, $2, $3, $4) RETURNING id;

-- name: DeleteMysteryAttachment :execrows
DELETE FROM mystery_attachments WHERE id = $1 AND mystery_id = $2;

-- name: GetMysteryAttachments :many
SELECT id, file_url, file_name, file_size FROM mystery_attachments WHERE mystery_id = $1 ORDER BY created_at;

-- name: GetMysteryAttachmentPaths :many
SELECT file_url FROM mystery_attachments WHERE mystery_id = $1 ORDER BY created_at, id;
