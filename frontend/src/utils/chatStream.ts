import type { Dispatch, SetStateAction } from "react";
import type { ChatMessage } from "../types/api";
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
