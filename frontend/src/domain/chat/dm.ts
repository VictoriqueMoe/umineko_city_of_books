import type { ChatMessage, ChatRoom, User } from "../../types/api";
import { formatTimeOfDay } from "../../utils/time";

export function getRoomDisplayName(room: ChatRoom, currentUser: User): string {
    if (room.type === "group") {
        return room.name || "Group Chat";
    }

    const other = room.members.find(m => m.id !== currentUser.id);
    if (other) {
        return other.display_name;
    }

    return "Direct Message";
}

export function getRoomAvatarUser(room: ChatRoom, currentUser: User): User | null {
    if (room.type === "dm") {
        return room.members.find(m => m.id !== currentUser.id) ?? null;
    }

    return null;
}

export function seenLabel(
    msg: ChatMessage,
    idx: number,
    messages: ChatMessage[],
    room: ChatRoom | undefined,
    selfId: string,
    receipts: Record<string, Record<string, string>>,
): string | null {
    if (!room) {
        return null;
    }

    for (let j = idx + 1; j < messages.length; j++) {
        if (messages[j].sender.id === selfId) {
            return null;
        }
    }

    const roomReceipts = receipts[room.id];
    if (!roomReceipts) {
        return null;
    }

    let latestReadAt = "";
    let seenByName = "";
    for (let i = 0; i < room.members.length; i++) {
        const member = room.members[i];
        if (member.id === selfId) {
            continue;
        }

        const readAt = roomReceipts[member.id];
        if (!readAt) {
            continue;
        }

        if (readAt < msg.created_at) {
            continue;
        }

        if (readAt > latestReadAt) {
            latestReadAt = readAt;
            seenByName = room.type === "dm" ? "" : member.display_name;
        }
    }

    if (!latestReadAt) {
        return null;
    }

    const time = formatTimeOfDay(latestReadAt);
    if (room.type === "dm") {
        return `seen ${time}`;
    }

    return `seen by ${seenByName} ${time}`;
}
