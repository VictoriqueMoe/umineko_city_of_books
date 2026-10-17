import type { ChatMessage, ChatRoom } from "../../types/api";

export type ChatThreadKind = "room" | "dm";

export interface ChatMessageTarget {
    id: string;
    createdAt?: string;
}

const THREAD_BASES: Record<ChatThreadKind, string> = {
    room: "/rooms",
    dm: "/chat",
};

export function chatThreadPath(kind: ChatThreadKind, roomId: string, target?: ChatMessageTarget): string {
    const path = `${THREAD_BASES[kind]}/${roomId}`;
    if (!target) {
        return path;
    }

    const query = target.createdAt ? `?${new URLSearchParams({ at: target.createdAt })}` : "";

    return `${path}${query}#msg-${target.id}`;
}

export function messagePermalink(
    room: Pick<ChatRoom, "id" | "type">,
    message: Pick<ChatMessage, "id" | "created_at">,
): string {
    const kind: ChatThreadKind = room.type === "dm" ? "dm" : "room";

    return chatThreadPath(kind, room.id, { id: message.id, createdAt: message.created_at });
}
