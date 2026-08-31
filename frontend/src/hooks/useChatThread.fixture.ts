import { roomCapabilities } from "../domain/chat/roomPolicy";
import type { ChatThread } from "./useChatThread";

function noop(): void {}

function never(): Promise<void> {
    return Promise.resolve();
}

function neverFound(): Promise<boolean> {
    return Promise.resolve(false);
}

export function makeChatThread(overrides: Partial<ChatThread> = {}): ChatThread {
    return {
        thread: {
            data: null,
            id: "room-1",
            loading: false,
            set: noop,
            toggleMute: never,
            mutePending: false,
            voice: {
                status: "idle",
                room: null,
                participantIds: [],
                presenceCount: 0,
                error: "",
                join: noop,
                leave: noop,
                clearError: noop,
                enabled: true,
            },
        },
        session: {
            viewer: null,
            status: "idle",
            messages: [],
            hasMore: false,
            loadingMore: false,
            containerRef: noop,
            contentRef: noop,
            endRef: { current: null },
            onScroll: noop,
            toBottom: noop,
            editingMessageId: null,
            startEditing: noop,
            cancelEditing: noop,
            deleteMessage: never,
            editMessage: never,
            editLast: noop,
            replyingTo: null,
            setReplyingTo: noop,
            typingUserIds: [],
            notifyTyping: noop,
            matchesViewerMention: null,
            readReceipts: {},
            onSent: noop,
            toggleReaction: never,
            togglePin: never,
            history: {
                hasMore: false,
                loadingMore: false,
                setMessages: noop,
                addMessage: noop,
                seedMessages: noop,
                loadUntilMessage: neverFound,
                resync: never,
            },
        },
        capabilities: roomCapabilities(null, null),
        toast: {
            message: null,
            show: noop,
        },
        ...overrides,
    };
}
