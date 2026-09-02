export interface Chatbot {
    id: string;
    user_id: string;
    username: string;
    display_name: string;
    avatar_url: string;
    system_prompt: string;
    base_prompt_id: string | null;
    model: string;
    reasoning_effort: string;
    verbosity: string;
    max_output_tokens: number;
    enabled: boolean;
}

export interface ChatbotPayload {
    username: string;
    display_name: string;
    avatar_url: string;
    system_prompt: string;
    base_prompt_id: string | null;
    model: string;
    reasoning_effort: string;
    verbosity: string;
    max_output_tokens: number;
    enabled: boolean;
}

export interface ChatbotBasePrompt {
    id: string;
    name: string;
    prompt: string;
    bot_count: number;
    created_at: string;
    updated_at: string;
}

export interface ChatbotBasePromptPayload {
    name: string;
    prompt: string;
}

export interface ChatbotBasePromptListResponse {
    base_prompts: ChatbotBasePrompt[];
}

export interface ChatbotChannelUsage {
    channel: string;
    invocations: number;
    prompt_tokens: number;
    cached_prompt_tokens: number;
    cache_write_tokens: number;
    completion_tokens: number;
    reasoning_tokens: number;
}

export interface ChatbotUsage {
    invocations: number;
    prompt_tokens: number;
    cached_prompt_tokens: number;
    cache_write_tokens: number;
    completion_tokens: number;
    reasoning_tokens: number;
    billed_usd: number | null;
    failed: number;
    quota: number;
    channels: ChatbotChannelUsage[];
}

export interface ChatbotSummary {
    user_id: string;
    username: string;
    display_name: string;
    avatar_url: string;
}

export interface ChatbotListResponse {
    chatbots: ChatbotSummary[];
}

export interface ChatbotModels {
    models: string[];
    error?: string;
}

export interface UsernameAvailability {
    username: string;
    available: boolean;
}

export interface ChatbotTestResult {
    ok: boolean;
    error?: string;
}
