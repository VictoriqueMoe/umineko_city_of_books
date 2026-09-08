-- name: CreateTheory :one
WITH t AS (
    INSERT INTO theories (user_id, title, body, episode, series)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, user_id, title, body, episode, series, credibility_score, status, created_at
)
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM t
JOIN users u ON t.user_id = u.id;

-- name: GetTheoryByID :one
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status,
       t.refuted_by_response_id, t.refuted_at, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role,
       ru.id AS refuter_id,
       COALESCE(ru.username, '')::text AS refuter_username,
       COALESCE(ru.display_name, '')::text AS refuter_display_name,
       COALESCE(ru.avatar_url, '')::text AS refuter_avatar_url,
       COALESCE((SELECT rur.role FROM user_roles rur WHERE rur.user_id = ru.id LIMIT 1), '')::text AS refuter_role
FROM theories t
JOIN users u ON t.user_id = u.id
LEFT JOIN users ru ON t.refuted_by_user_id = ru.id
WHERE t.id = $1;

-- name: UpdateTheoryAsAdmin :execrows
UPDATE theories SET title = $1, body = $2, episode = $3, updated_at = NOW()
WHERE id = $4;

-- name: UpdateTheoryOwned :execrows
UPDATE theories SET title = $1, body = $2, episode = $3, updated_at = NOW()
WHERE id = $4 AND user_id = $5;

-- name: DeleteTheoryOwned :execrows
DELETE FROM theories WHERE id = $1 AND user_id = $2;

-- name: DeleteTheoryAsAdmin :execrows
DELETE FROM theories WHERE id = $1;

-- name: UpdateTheoryCredibilityScore :exec
UPDATE theories SET credibility_score = $1 WHERE id = $2;

-- name: RecomputeTheoryStatus :exec
UPDATE theories t SET status = CASE
    WHEN EXISTS (SELECT 1 FROM responses r WHERE r.theory_id = t.id AND r.parent_id IS NULL) THEN 'contested'::theory_status
    ELSE 'open'::theory_status
END
WHERE t.id = $1 AND t.status <> 'refuted';

-- name: MarkTheoryRefuted :execrows
UPDATE theories t SET status = 'refuted', refuted_by_response_id = r.id, refuted_by_user_id = r.user_id, refuted_at = NOW()
FROM responses r
WHERE t.id = $1 AND r.id = $2 AND r.theory_id = t.id AND r.parent_id IS NULL AND r.side = 'without_love' AND r.user_id <> t.user_id AND t.status <> 'refuted';

-- name: GetTheoryAuthorID :one
SELECT t.user_id FROM theories t WHERE t.id = $1;

-- name: GetTheorySeries :one
SELECT t.series FROM theories t WHERE t.id = $1;

-- name: GetTheoryTitle :one
SELECT t.title FROM theories t WHERE t.id = $1;

-- name: CountUserTheoriesToday :one
SELECT COUNT(*) FROM theories t WHERE t.user_id = $1 AND t.created_at > NOW() - INTERVAL '1 day';

-- name: CountUserTheoryResponsesToday :one
SELECT COUNT(*) FROM responses r WHERE r.user_id = $1 AND r.created_at > NOW() - INTERVAL '1 day';

-- name: CreateTheoryEvidence :one
INSERT INTO theory_evidence (theory_id, audio_id, quote_index, note, sort_order, lang)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, audio_id, quote_index, note, sort_order, lang;

-- name: AddTheoryEvidence :exec
INSERT INTO theory_evidence (theory_id, audio_id, quote_index, note, sort_order)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteTheoryEvidenceByTheory :exec
DELETE FROM theory_evidence WHERE theory_id = $1;

-- name: ListTheoryEvidence :many
SELECT te.id, te.audio_id, te.quote_index, te.note, te.sort_order, te.lang
FROM theory_evidence te
WHERE te.theory_id = $1
ORDER BY te.sort_order;

-- name: CreateTheoryResponse :one
WITH resp AS (
    INSERT INTO responses (theory_id, user_id, side, body, parent_id)
    VALUES ($1, $2, $3, $4, $5)
    RETURNING id, user_id, parent_id, side, body, created_at
)
SELECT resp.id, resp.parent_id, resp.side, resp.body, resp.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM resp
JOIN users u ON resp.user_id = u.id;

-- name: ListTheoryResponses :many
SELECT r.id, r.parent_id, r.side, r.body, r.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM responses r
JOIN users u ON r.user_id = u.id
WHERE r.theory_id = $1
ORDER BY r.created_at ASC;

-- name: DeleteTheoryResponseOwned :execrows
DELETE FROM responses WHERE id = $1 AND user_id = $2;

-- name: DeleteTheoryResponseAsAdmin :execrows
DELETE FROM responses WHERE id = $1;

-- name: GetTheoryResponseInfo :one
SELECT r.user_id, r.theory_id FROM responses r WHERE r.id = $1;

-- name: GetTheoryResponseMeta :one
SELECT r.user_id, r.theory_id, r.side, r.parent_id FROM responses r WHERE r.id = $1;

-- name: CreateTheoryResponseEvidence :one
INSERT INTO response_evidence (response_id, audio_id, quote_index, note, sort_order, lang)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, audio_id, quote_index, note, sort_order, lang;

-- name: ListTheoryResponseEvidence :many
SELECT re.id, re.audio_id, re.quote_index, re.note, re.sort_order, re.lang
FROM response_evidence re
WHERE re.response_id = $1
ORDER BY re.sort_order;

-- name: ListTheoryResponseEvidenceBatch :many
SELECT re.response_id, re.id, re.audio_id, re.quote_index, re.note, re.sort_order, re.lang
FROM response_evidence re
WHERE re.response_id = ANY(string_to_array($1::text, ',')::uuid[])
ORDER BY re.response_id, re.sort_order;

-- name: UpdateTheoryResponseEvidenceTruthWeight :exec
UPDATE response_evidence SET truth_weight = $1 WHERE id = $2;

-- name: ListTheoryResponseEvidenceWeights :many
SELECT r.side, COALESCE(SUM(re.truth_weight), 0)::double precision AS truth_weight
FROM responses r
LEFT JOIN response_evidence re ON r.id = re.response_id
WHERE r.theory_id = $1 AND r.parent_id IS NULL
GROUP BY r.side;

-- name: GetUserTheoryVote :one
SELECT tv.value FROM theory_votes tv WHERE tv.user_id = $1 AND tv.theory_id = $2;

-- name: GetTheoryVoteCounts :one
SELECT COALESCE(SUM(CASE WHEN tv.value = 1 THEN 1 ELSE 0 END), 0)::bigint AS up_votes,
       COALESCE(SUM(CASE WHEN tv.value = -1 THEN 1 ELSE 0 END), 0)::bigint AS down_votes
FROM theory_votes tv
WHERE tv.theory_id = $1;

-- name: ListTheoryVoteScores :many
SELECT tv.theory_id AS target_id,
       COALESCE(SUM(CASE WHEN tv.value = 1 THEN 1 ELSE 0 END), 0)::bigint AS up_votes,
       COALESCE(SUM(CASE WHEN tv.value = -1 THEN 1 ELSE 0 END), 0)::bigint AS down_votes
FROM theory_votes tv
WHERE tv.theory_id = ANY(string_to_array($1::text, ',')::uuid[])
GROUP BY tv.theory_id;

-- name: ListTheoryResponseVoteScores :many
SELECT rv.response_id AS target_id,
       COALESCE(SUM(CASE WHEN rv.value = 1 THEN 1 ELSE 0 END), 0)::bigint AS up_votes,
       COALESCE(SUM(CASE WHEN rv.value = -1 THEN 1 ELSE 0 END), 0)::bigint AS down_votes
FROM response_votes rv
WHERE rv.response_id = ANY(string_to_array($1::text, ',')::uuid[])
GROUP BY rv.response_id;

-- name: ListUserTheoryVotes :many
SELECT tv.theory_id AS target_id, tv.value
FROM theory_votes tv
WHERE tv.user_id = $1 AND tv.theory_id = ANY(string_to_array($2::text, ',')::uuid[]);

-- name: ListUserTheoryResponseVotes :many
SELECT rv.response_id AS target_id, rv.value
FROM response_votes rv
WHERE rv.user_id = $1 AND rv.response_id = ANY(string_to_array($2::text, ',')::uuid[]);

-- name: GetTheoryResponseSideCounts :one
SELECT COALESCE(SUM(CASE WHEN r.side = 'with_love' THEN 1 ELSE 0 END), 0)::bigint AS with_love_count,
       COALESCE(SUM(CASE WHEN r.side = 'without_love' THEN 1 ELSE 0 END), 0)::bigint AS without_love_count
FROM responses r
WHERE r.theory_id = $1 AND r.parent_id IS NULL;

-- name: ListTheoryResponseSideCounts :many
SELECT r.theory_id,
       COALESCE(SUM(CASE WHEN r.side = 'with_love' THEN 1 ELSE 0 END), 0)::bigint AS with_love_count,
       COALESCE(SUM(CASE WHEN r.side = 'without_love' THEN 1 ELSE 0 END), 0)::bigint AS without_love_count
FROM responses r
WHERE r.theory_id = ANY(string_to_array($1::text, ',')::uuid[]) AND r.parent_id IS NULL
GROUP BY r.theory_id;

-- name: CountUserTheoryActivity :one
SELECT ((SELECT COUNT(*) FROM theories t WHERE t.user_id = $1) + (SELECT COUNT(*) FROM responses r WHERE r.user_id = $1))::bigint AS total;

-- name: ListUserTheoryActivity :many
SELECT combined.type, combined.theory_id, combined.theory_title, combined.side, combined.body, combined.created_at
FROM (
    SELECT 'theory'::text AS type, t.id AS theory_id, t.title AS theory_title, ''::text AS side, t.body, t.created_at
    FROM theories t
    WHERE t.user_id = $1
    UNION ALL
    SELECT 'response'::text AS type, r.theory_id AS theory_id, th.title AS theory_title, r.side, r.body, r.created_at
    FROM responses r
    JOIN theories th ON r.theory_id = th.id
    WHERE r.user_id = $1
) combined
ORDER BY combined.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountTheoriesFiltered :one
SELECT COUNT(*)
FROM theories t
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]));

-- name: ListTheoriesByNew :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY t.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListTheoriesByOld :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY t.created_at ASC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListTheoriesByPopular :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COALESCE(SUM(tv.value), 0) FROM theory_votes tv WHERE tv.theory_id = t.id) DESC, t.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListTheoriesByPopularAsc :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COALESCE(SUM(tv.value), 0) FROM theory_votes tv WHERE tv.theory_id = t.id) ASC, t.created_at ASC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListTheoriesByControversial :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM theory_votes tv WHERE tv.theory_id = t.id) DESC, t.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListTheoriesByControversialAsc :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY (SELECT COUNT(*) FROM theory_votes tv WHERE tv.theory_id = t.id) ASC, t.created_at ASC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListTheoriesByCredibility :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY t.credibility_score DESC, t.created_at DESC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: ListTheoriesByCredibilityAsc :many
SELECT t.id, t.title, t.body, t.episode, t.series, t.credibility_score, t.status, t.created_at,
       u.id AS author_id,
       u.username AS author_username,
       u.display_name AS author_display_name,
       u.avatar_url AS author_avatar_url,
       COALESCE((SELECT ur.role FROM user_roles ur WHERE ur.user_id = u.id LIMIT 1), '')::text AS author_role
FROM theories t
JOIN users u ON t.user_id = u.id
WHERE (sqlc.arg(series)::text = '' OR t.series = sqlc.arg(series)::text)
  AND (sqlc.arg(episode)::int <= 0 OR t.episode = sqlc.arg(episode)::int)
  AND (sqlc.arg(author_id)::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR t.user_id = sqlc.arg(author_id)::uuid)
  AND (sqlc.arg(search)::text = '' OR (t.title LIKE '%' || sqlc.arg(search)::text || '%' OR t.body LIKE '%' || sqlc.arg(search)::text || '%'))
  AND (sqlc.arg(exclude_user_ids)::text = '' OR t.user_id <> ALL(string_to_array(sqlc.arg(exclude_user_ids)::text, ',')::uuid[]))
ORDER BY t.credibility_score ASC, t.created_at ASC
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;
