import { useCallback } from "react";
import type { ChatMessage, UserProfile } from "../types/api";
import { deleteChatMessage, editChatMessage } from "../api/endpoints";
import { applyChatMessageEdited } from "../utils/chatStream";

interface UseChatMessageHandlersOptions {
    user: UserProfile | null;
    messages: ChatMessage[];
    setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>;
    setEditingMessageId: (id: string | null) => void;
    onError?: (message: string) => void;
    editLastBlocked?: boolean;
}

interface UseChatMessageHandlersResult {
    handleDeleteMessage: (message: ChatMessage) => Promise<void>;
    handleEditMessage: (message: ChatMessage, newBody: string) => Promise<void>;
    handleEditLast: () => void;
}

export function useChatMessageHandlers({
    user,
    messages,
    setMessages,
    setEditingMessageId,
    onError,
    editLastBlocked = false,
}: UseChatMessageHandlersOptions): UseChatMessageHandlersResult {
    const handleDeleteMessage = useCallback(
        async (message: ChatMessage) => {
            try {
                await deleteChatMessage(message.id);
                setMessages(prev => prev.filter(m => m.id !== message.id));
            } catch (err) {
                if (onError) {
                    onError(err instanceof Error ? err.message : "Failed to delete message");
                }
            }
        },
        [setMessages, onError],
    );

    const handleEditMessage = useCallback(
        async (message: ChatMessage, newBody: string) => {
            try {
                const updated = await editChatMessage(message.id, newBody);
                applyChatMessageEdited(updated, setMessages);
            } catch (err) {
                if (onError) {
                    onError(err instanceof Error ? err.message : "Failed to edit message");
                }
                throw err;
            }
        },
        [setMessages, onError],
    );

    const handleEditLast = useCallback(() => {
        if (!user || editLastBlocked) {
            return;
        }
        for (let i = messages.length - 1; i >= 0; i--) {
            const candidate = messages[i];
            if (candidate.sender.id === user.id && !candidate.is_system) {
                setEditingMessageId(candidate.id);
                requestAnimationFrame(() => {
                    const el = document.getElementById(`chat-msg-${candidate.id}`);
                    if (el) {
                        el.scrollIntoView({ behavior: "smooth", block: "center" });
                    }
                });
                return;
            }
        }
    }, [user, editLastBlocked, messages, setEditingMessageId]);

    return {
        handleDeleteMessage,
        handleEditMessage,
        handleEditLast,
    };
}
