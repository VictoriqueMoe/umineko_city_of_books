import { describe, expect, it } from "vitest";
import { makeDmRoom } from "../../test-utils/fixtures";
import type { ChatRoom } from "../../types/api";
import { moveRoomToFront } from "./dmRoster";

function makeRoom(id: string, overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({ id, last_message_at: "2026-01-01T00:00:00Z", unread: false, ...overrides });
}

function ids(rooms: ChatRoom[]): string[] {
    return rooms.map(room => room.id);
}

describe("moveRoomToFront", () => {
    it("lifts a room from the middle to the front", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b"), makeRoom("c")];

        // when
        const next = moveRoomToFront(rooms, "b", {});

        // then
        expect(ids(next)).toEqual(["b", "a", "c"]);
    });

    it("lifts the last room to the front", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b"), makeRoom("c")];

        // when
        const next = moveRoomToFront(rooms, "c", {});

        // then
        expect(ids(next)).toEqual(["c", "a", "b"]);
    });

    it("leaves the order alone when the room is already at the front", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b")];

        // when
        const next = moveRoomToFront(rooms, "a", {});

        // then
        expect(ids(next)).toEqual(["a", "b"]);
    });

    it("applies the patch to the room it lifts", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b")];

        // when
        const next = moveRoomToFront(rooms, "b", { last_message_at: "2026-08-28T12:00:00Z", unread: true });

        // then
        expect(next[0].last_message_at).toBe("2026-08-28T12:00:00Z");
        expect(next[0].unread).toBe(true);
    });

    it("keeps the fields the patch does not name", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b", { name: "Beatrice", viewer_muted: true })];

        // when
        const next = moveRoomToFront(rooms, "b", { unread: true });

        // then
        expect(next[0].name).toBe("Beatrice");
        expect(next[0].viewer_muted).toBe(true);
    });

    it("marks a room read when the patch says so, since the viewer is the sender", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b", { unread: true })];

        // when
        const next = moveRoomToFront(rooms, "b", { unread: false });

        // then
        expect(next[0].unread).toBe(false);
    });

    it("returns the very same array when the room is not in the list, so the caller can tell", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b")];

        // when
        const next = moveRoomToFront(rooms, "missing", { unread: true });

        // then
        expect(next).toBe(rooms);
    });

    it("returns the very same array when the list is empty", () => {
        // given
        const rooms: ChatRoom[] = [];

        // when
        const next = moveRoomToFront(rooms, "a", {});

        // then
        expect(next).toBe(rooms);
    });

    it("does not mutate the list it was given", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b", { unread: false })];

        // when
        moveRoomToFront(rooms, "b", { unread: true });

        // then
        expect(ids(rooms)).toEqual(["a", "b"]);
        expect(rooms[1].unread).toBe(false);
    });

    it("leaves the rooms it did not touch identical, so their rows do not re-render", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b"), makeRoom("c")];

        // when
        const next = moveRoomToFront(rooms, "b", { unread: true });

        // then
        expect(next[1]).toBe(rooms[0]);
        expect(next[2]).toBe(rooms[2]);
        expect(next[0]).not.toBe(rooms[1]);
    });

    it("lifts the first match when the list holds a duplicate id", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b", { name: "first" }), makeRoom("b", { name: "second" })];

        // when
        const next = moveRoomToFront(rooms, "b", {});

        // then
        expect(ids(next)).toEqual(["b", "a", "b"]);
        expect(next[0].name).toBe("first");
    });
});
