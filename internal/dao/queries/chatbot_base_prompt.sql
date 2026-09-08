-- name: ListChatbotBasePrompts :many
SELECT b.id, b.name, b.prompt, b.created_at, b.updated_at,
       (SELECT COUNT(*) FROM chatbots c WHERE c.base_prompt_id = b.id) AS bot_count
FROM chatbot_base_prompts b
ORDER BY b.name;

-- name: GetChatbotBasePromptByID :one
SELECT b.id, b.name, b.prompt, b.created_at, b.updated_at,
       (SELECT COUNT(*) FROM chatbots c WHERE c.base_prompt_id = b.id) AS bot_count
FROM chatbot_base_prompts b
WHERE b.id = $1;

-- name: CreateChatbotBasePrompt :one
INSERT INTO chatbot_base_prompts (id, name, prompt)
VALUES (gen_random_uuid(), $1, $2)
RETURNING id, name, prompt, created_at, updated_at, 0::bigint AS bot_count;

-- name: UpdateChatbotBasePrompt :one
UPDATE chatbot_base_prompts SET name = $2, prompt = $3, updated_at = NOW()
WHERE chatbot_base_prompts.id = $1
RETURNING id, name, prompt, created_at, updated_at,
    (SELECT COUNT(*) FROM chatbots c WHERE c.base_prompt_id = chatbot_base_prompts.id) AS bot_count;

-- name: DeleteChatbotBasePrompt :execrows
DELETE FROM chatbot_base_prompts b WHERE b.id = $1;
