import type { ChatMessage } from "../../types/api";

export const MAX_MESSAGE_ID = "ffffffff-ffff-ffff-ffff-ffffffffffff";

export interface MessageCursorSource {
    created_at: string;
    id: string;
}

export function beforeCursor(message: MessageCursorSource): string {
    return `${message.created_at}|${message.id}`;
}

export function dedupeById(known: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
    const existing = new Set(known.map(message => message.id));
    const unique: ChatMessage[] = [];

    for (const message of incoming) {
        if (existing.has(message.id)) {
            continue;
        }

        unique.push(message);
        existing.add(message.id);
    }

    return unique;
}

export function mergeChronological(current: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
    const merged = [...current, ...dedupeById(current, incoming)];
    merged.sort((a, b) => {
        const ta = Date.parse(a.created_at);
        const tb = Date.parse(b.created_at);
        if (ta !== tb) {
            return ta - tb;
        }

        return a.id.localeCompare(b.id);
    });

    return merged;
}

export interface TrimResult {
    messages: ChatMessage[];
    trimmed: boolean;
}

export function trimToWindow(current: ChatMessage[], limit: number | undefined): TrimResult {
    if (limit === undefined || current.length <= limit) {
        return { messages: current, trimmed: false };
    }

    return { messages: current.slice(current.length - limit), trimmed: true };
}

export function appendIfUnknown(current: ChatMessage[], message: ChatMessage): ChatMessage[] {
    if (current.some(m => m.id === message.id)) {
        return current;
    }

    return [...current, message];
}

export function upsertById(current: ChatMessage[], message: ChatMessage): ChatMessage[] {
    const idx = current.findIndex(m => m.id === message.id);
    if (idx === -1) {
        return [...current, message];
    }

    const next = current.slice();
    next[idx] = message;

    return next;
}

export type MessagePatch = (current: ChatMessage[]) => ChatMessage[];

export interface MessageListState {
    roomId: string | undefined;
    messages: ChatMessage[];
    hasMore: boolean;
}

export type MessageListAction =
    | { type: "roomOpened"; roomId: string | undefined }
    | { type: "historyLoaded"; roomId: string; messages: ChatMessage[]; total: number }
    | { type: "historyFailed"; roomId: string }
    | { type: "seeded"; roomId: string; messages: ChatMessage[] }
    | { type: "messageReceived"; roomId: string | undefined; message: ChatMessage; limit?: number }
    | { type: "messageUpserted"; roomId: string | undefined; message: ChatMessage; limit?: number }
    | { type: "olderLoaded"; roomId: string | undefined; messages: ChatMessage[]; limit?: number }
    | { type: "gapLoaded"; roomId: string | undefined; messages: ChatMessage[]; limit?: number }
    | { type: "resynced"; roomId: string | undefined; messages: ChatMessage[]; limit?: number }
    | { type: "messagesPatched"; roomId: string | undefined; patch: MessagePatch; limit?: number }
    | { type: "hasMoreChanged"; hasMore: boolean };

export function emptyMessageList(roomId: string | undefined): MessageListState {
    return { roomId, messages: [], hasMore: false };
}

function messagesFor(state: MessageListState, roomId: string | undefined): ChatMessage[] {
    return state.roomId === roomId ? state.messages : [];
}

function windowed(
    state: MessageListState,
    roomId: string | undefined,
    next: ChatMessage[],
    limit: number | undefined,
): MessageListState {
    const { messages, trimmed } = trimToWindow(next, limit);

    return {
        roomId,
        messages,
        hasMore: trimmed || (state.roomId === roomId && state.hasMore),
    };
}

export function messageListReducer(state: MessageListState, action: MessageListAction): MessageListState {
    switch (action.type) {
        case "roomOpened": {
            if (state.roomId === action.roomId) {
                return state;
            }

            return emptyMessageList(action.roomId);
        }
        case "historyLoaded": {
            return {
                roomId: action.roomId,
                messages: action.messages,
                hasMore: action.messages.length < action.total,
            };
        }
        case "historyFailed": {
            return emptyMessageList(action.roomId);
        }
        case "seeded": {
            return { roomId: action.roomId, messages: action.messages, hasMore: false };
        }
        case "messageReceived": {
            const base = messagesFor(state, action.roomId);

            return windowed(state, action.roomId, appendIfUnknown(base, action.message), action.limit);
        }
        case "messageUpserted": {
            const base = messagesFor(state, action.roomId);

            return windowed(state, action.roomId, upsertById(base, action.message), action.limit);
        }
        case "olderLoaded": {
            const base = messagesFor(state, action.roomId);

            return windowed(state, action.roomId, [...dedupeById(base, action.messages), ...base], action.limit);
        }
        case "gapLoaded": {
            const base = messagesFor(state, action.roomId);

            return windowed(state, action.roomId, mergeChronological(base, action.messages), action.limit);
        }
        case "resynced": {
            const base = messagesFor(state, action.roomId);
            const fresh = dedupeById(base, action.messages);
            const next = fresh.length === 0 ? base : [...base, ...fresh];

            return windowed(state, action.roomId, next, action.limit);
        }
        case "messagesPatched": {
            const base = messagesFor(state, action.roomId);

            return windowed(state, action.roomId, action.patch(base), action.limit);
        }
        case "hasMoreChanged": {
            return { ...state, hasMore: action.hasMore };
        }
    }
}
