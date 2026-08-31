import type { ChatRoom } from "../../types/api";

export type RosterEvent = "dm" | "otherRoom" | "unknown";

export function dmRoomsOf(rooms: ChatRoom[]): ChatRoom[] {
    return rooms.filter(room => room.type === "dm");
}

export function rosterEventFor(rooms: ChatRoom[], roomId: string): RosterEvent {
    const room = rooms.find(candidate => candidate.id === roomId);
    if (!room) {
        return "unknown";
    }

    return room.type === "dm" ? "dm" : "otherRoom";
}

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

export function patchRoom(rooms: ChatRoom[], roomId: string, patch: Partial<ChatRoom>): ChatRoom[] {
    return rooms.map(room => (room.id === roomId ? { ...room, ...patch } : room));
}

export function prependRoom(rooms: ChatRoom[], room: ChatRoom): ChatRoom[] {
    if (rooms.some(known => known.id === room.id)) {
        return rooms;
    }

    return [room, ...rooms];
}

export function removeRoom(rooms: ChatRoom[], roomId: string): ChatRoom[] {
    return rooms.filter(room => room.id !== roomId);
}
