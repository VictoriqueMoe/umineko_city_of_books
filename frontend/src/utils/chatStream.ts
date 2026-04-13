import type { Dispatch, SetStateAction } from "react";
import type { ChatMessage, PostMedia } from "../types/api";
import { markChatRoomRead } from "../api/endpoints";

export function handleIncomingChatMessage(
    chatMsg: ChatMessage,
    activeRoomId: string | null,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
    scrollToBottom: () => void,
): boolean {
    if (chatMsg.room_id !== activeRoomId) {
        return false;
    }
    setMessages(prev => {
        if (prev.some(m => m.id === chatMsg.id)) {
            return prev;
        }
        return [...prev, chatMsg];
    });
    scrollToBottom();
    if (document.visibilityState === "visible" && document.hasFocus()) {
        markChatRoomRead(chatMsg.room_id).catch(() => {});
    }
    return true;
}

export interface ChatMessageMediaAddedPayload {
    room_id: string;
    message_id: string;
    media: PostMedia;
}

export function handleIncomingChatMessageMedia(
    payload: ChatMessageMediaAddedPayload,
    activeRoomId: string | null,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): boolean {
    if (payload.room_id !== activeRoomId) {
        return false;
    }
    setMessages(prev => {
        let changed = false;
        const next = prev.map(m => {
            if (m.id !== payload.message_id) {
                return m;
            }
            const existing = m.media ?? [];
            if (existing.some(x => x.id === payload.media.id)) {
                return m;
            }
            changed = true;
            return { ...m, media: [...existing, payload.media] };
        });
        return changed ? next : prev;
    });
    return true;
}
