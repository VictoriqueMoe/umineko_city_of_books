import type { ChatbotChannelUsage } from "../types/api";

const CHANNEL_LABELS: Record<string, string> = {
    group: "Group chats",
    dm: "DMs",
    post: "Posts",
    post_comment: "Post comments",
};
const CHANNEL_ORDER = Object.keys(CHANNEL_LABELS);

export const CHARS_PER_TOKEN = 4;
export const CACHING_TOKEN_THRESHOLD = 1000;

export function estimateTokens(text: string): number {
    return Math.ceil(text.length / CHARS_PER_TOKEN);
}

export function meetsCachingThreshold(tokens: number): boolean {
    return tokens >= CACHING_TOKEN_THRESHOLD;
}

export function tokensToCachingThreshold(tokens: number): number {
    return CACHING_TOKEN_THRESHOLD - tokens;
}

export function channelLabel(channel: string): string {
    return CHANNEL_LABELS[channel] ?? channel;
}

export function channelTokens(channel: ChatbotChannelUsage): number {
    return channel.prompt_tokens + channel.completion_tokens;
}

function emptyChannel(channel: string): ChatbotChannelUsage {
    return {
        channel,
        invocations: 0,
        prompt_tokens: 0,
        cached_prompt_tokens: 0,
        cache_write_tokens: 0,
        completion_tokens: 0,
        reasoning_tokens: 0,
    };
}

export function channelRows(channels: ChatbotChannelUsage[]): ChatbotChannelUsage[] {
    const returned = new Map<string, ChatbotChannelUsage>();
    for (const channel of channels) {
        returned.set(channel.channel, channel);
    }

    const rows: ChatbotChannelUsage[] = [];
    for (const name of CHANNEL_ORDER) {
        rows.push(returned.get(name) ?? emptyChannel(name));
        returned.delete(name);
    }

    for (const channel of returned.values()) {
        rows.push(channel);
    }

    return rows;
}
