-- name: ListChatbots :many
SELECT c.id, c.user_id, u.username, u.display_name, u.avatar_url,
       c.system_prompt, c.base_prompt_id, COALESCE(b.prompt, '')::text AS base_prompt,
       c.model, c.reasoning_effort, c.verbosity, c.max_output_tokens, c.enabled
FROM chatbots c
JOIN users u ON u.id = c.user_id
LEFT JOIN chatbot_base_prompts b ON b.id = c.base_prompt_id
ORDER BY u.username;

-- name: GetChatbotByUserID :one
SELECT c.id, c.user_id, u.username, u.display_name, u.avatar_url,
       c.system_prompt, c.base_prompt_id, COALESCE(b.prompt, '')::text AS base_prompt,
       c.model, c.reasoning_effort, c.verbosity, c.max_output_tokens, c.enabled
FROM chatbots c
JOIN users u ON u.id = c.user_id
LEFT JOIN chatbot_base_prompts b ON b.id = c.base_prompt_id
WHERE c.user_id = $1;

-- name: CreateChatbot :one
WITH c AS (
    INSERT INTO chatbots (user_id, system_prompt, base_prompt_id, model, reasoning_effort, verbosity, max_output_tokens, enabled)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    RETURNING id, user_id, system_prompt, base_prompt_id, model, reasoning_effort, verbosity, max_output_tokens, enabled
)
SELECT c.id, c.user_id, u.username, u.display_name, u.avatar_url,
       c.system_prompt, c.base_prompt_id, COALESCE(b.prompt, '')::text AS base_prompt,
       c.model, c.reasoning_effort, c.verbosity, c.max_output_tokens, c.enabled
FROM c
JOIN users u ON u.id = c.user_id
LEFT JOIN chatbot_base_prompts b ON b.id = c.base_prompt_id;

-- name: UpdateChatbot :one
WITH c AS (
    UPDATE chatbots
       SET system_prompt = $2,
           base_prompt_id = $3,
           model = $4,
           reasoning_effort = $5,
           verbosity = $6,
           max_output_tokens = $7,
           enabled = $8,
           updated_at = NOW()
     WHERE chatbots.id = $1
    RETURNING id, user_id, system_prompt, base_prompt_id, model, reasoning_effort, verbosity, max_output_tokens, enabled
)
SELECT c.id, c.user_id, u.username, u.display_name, u.avatar_url,
       c.system_prompt, c.base_prompt_id, COALESCE(b.prompt, '')::text AS base_prompt,
       c.model, c.reasoning_effort, c.verbosity, c.max_output_tokens, c.enabled
FROM c
JOIN users u ON u.id = c.user_id
LEFT JOIN chatbot_base_prompts b ON b.id = c.base_prompt_id;

-- name: DeleteChatbot :execrows
DELETE FROM users u
WHERE u.is_bot AND u.id = (SELECT c.user_id FROM chatbots c WHERE c.id = $1);

-- name: CreateChatbotInvocation :one
INSERT INTO chatbot_invocations (bot_user_id, user_id, room_id, message_id, channel, model, status)
VALUES ($1, $2, $3, $4, $5, $6, 'pending')
RETURNING id, bot_user_id, user_id, room_id, message_id, channel, model, status;

-- name: CompleteChatbotInvocation :exec
UPDATE chatbot_invocations
   SET prompt_tokens = $2,
       cached_prompt_tokens = $3,
       cache_write_tokens = $4,
       completion_tokens = $5,
       reasoning_tokens = $6,
       status = $7
 WHERE chatbot_invocations.id = $1;

-- name: CountUserChatbotInvocationsToday :one
SELECT COUNT(*) FROM chatbot_invocations ci
WHERE ci.user_id = $1 AND ci.status <> 'failed' AND ci.created_at > NOW() - INTERVAL '1 day';

-- name: CountChatbotInvocationsToday :one
SELECT COUNT(*) FROM chatbot_invocations ci
WHERE ci.status <> 'failed' AND ci.created_at > NOW() - INTERVAL '1 day';

-- name: ChatbotStatsSince :many
SELECT ci.channel,
       COUNT(*) AS invocations,
       COALESCE(SUM(ci.prompt_tokens), 0)::bigint AS prompt_tokens,
       COALESCE(SUM(ci.cached_prompt_tokens), 0)::bigint AS cached_prompt_tokens,
       COALESCE(SUM(ci.cache_write_tokens), 0)::bigint AS cache_write_tokens,
       COALESCE(SUM(ci.completion_tokens), 0)::bigint AS completion_tokens,
       COALESCE(SUM(ci.reasoning_tokens), 0)::bigint AS reasoning_tokens,
       COUNT(*) FILTER (WHERE ci.status = 'failed') AS failed,
       COUNT(*) FILTER (WHERE ci.status = 'quota') AS quota
FROM chatbot_invocations ci
WHERE ci.created_at >= $1
GROUP BY ci.channel
ORDER BY COUNT(*) DESC, ci.channel;

-- name: GetOldestChatbotInvocationToday :one
SELECT COALESCE(MIN(created_at), '0001-01-01 00:00:00+00'::timestamptz)::timestamptz AS oldest
FROM chatbot_invocations
WHERE status <> 'failed' AND created_at > NOW() - INTERVAL '1 day';

-- name: GetOldestUserChatbotInvocationToday :one
SELECT COALESCE(MIN(created_at), '0001-01-01 00:00:00+00'::timestamptz)::timestamptz AS oldest
FROM chatbot_invocations
WHERE user_id = $1 AND status <> 'failed' AND created_at > NOW() - INTERVAL '1 day';
