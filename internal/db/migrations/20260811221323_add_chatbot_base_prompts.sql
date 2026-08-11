-- +goose Up
CREATE TABLE chatbot_base_prompts (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    prompt TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE chatbots ADD COLUMN base_prompt_id UUID REFERENCES chatbot_base_prompts(id) ON DELETE RESTRICT;

CREATE INDEX idx_chatbots_base_prompt_id ON chatbots(base_prompt_id) WHERE base_prompt_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_chatbots_base_prompt_id;

ALTER TABLE chatbots DROP COLUMN base_prompt_id;

DROP TABLE chatbot_base_prompts;
