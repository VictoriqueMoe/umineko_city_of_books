import type { Chatbot, ChatbotPayload } from "../types/api";

export interface ChatbotFields {
    username: string;
    displayName: string;
    avatarURL: string;
    prompt: string;
    basePromptID: string;
    model: string;
    effort: string;
    verbosity: string;
    maxTokens: string;
}

export type UsernameCheck =
    | { state: "idle" }
    | { state: "checking" }
    | { state: "available"; username: string }
    | { state: "taken"; username: string }
    | { state: "failed" };

export const EMPTY_CHATBOT_FIELDS: ChatbotFields = {
    username: "",
    displayName: "",
    avatarURL: "",
    prompt: "",
    basePromptID: "",
    model: "",
    effort: "",
    verbosity: "",
    maxTokens: "",
};

export function fieldsFromBot(bot: Chatbot): ChatbotFields {
    return {
        username: bot.username,
        displayName: bot.display_name,
        avatarURL: bot.avatar_url,
        prompt: bot.system_prompt,
        basePromptID: bot.base_prompt_id ?? "",
        model: bot.model,
        effort: bot.reasoning_effort,
        verbosity: bot.verbosity,
        maxTokens: bot.max_output_tokens > 0 ? String(bot.max_output_tokens) : "",
    };
}

export function buildPayload(fields: ChatbotFields, enabled: boolean): ChatbotPayload {
    const maxTokens = parseInt(fields.maxTokens, 10);

    return {
        username: fields.username.trim(),
        display_name: fields.displayName.trim(),
        avatar_url: fields.avatarURL.trim(),
        system_prompt: fields.prompt,
        base_prompt_id: fields.basePromptID === "" ? null : fields.basePromptID,
        model: fields.model.trim(),
        reasoning_effort: fields.effort,
        verbosity: fields.verbosity,
        max_output_tokens: isNaN(maxTokens) ? 0 : maxTokens,
        enabled,
    };
}

export function usernameHint(check: UsernameCheck, editing: boolean): string {
    if (editing) {
        return "A bot's handle cannot be changed after it is created.";
    }

    switch (check.state) {
        case "checking":
            return "Checking whether that handle is free...";
        case "available":
            return `@${check.username} is free.`;
        case "taken":
            return `@${check.username} is already taken. Pick another.`;
        case "failed":
            return "Could not check that handle just now. Saving will still tell you if it is taken.";
        default:
            return "The handle members type to reach the bot. It has to be free, exactly like a human account.";
    }
}
