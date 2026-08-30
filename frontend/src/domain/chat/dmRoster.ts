import type { ChatRoom } from "../../types/api";

export function moveRoomToFront(rooms: ChatRoom[], roomId: string, patch: Partial<ChatRoom>): ChatRoom[] {
    const index = rooms.findIndex(room => room.id === roomId);
    if (index === -1) {
        return rooms;
    }

    const updated: ChatRoom = { ...rooms[index], ...patch };

    const next = rooms.slice();
    next.splice(index, 1);
    next.unshift(updated);

    return next;
}
