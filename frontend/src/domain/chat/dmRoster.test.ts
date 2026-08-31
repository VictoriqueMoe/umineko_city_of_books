import { describe, expect, it } from "vitest";
import { makeChatRoom, makeDmRoom } from "../../test-utils/fixtures";
import type { ChatRoom } from "../../types/api";
import {
    dmRoomsOf,
    moveRoomToFront,
    patchRoom,
    prependRoom,
    removeRoom,
    rosterEventFor,
    type RosterEvent,
} from "./dmRoster";

function makeRoom(id: string, overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({ id, last_message_at: "2026-01-01T00:00:00Z", unread: false, ...overrides });
}

function makeGroupRoom(id: string, overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({ id, name: "Rokkenjima", ...overrides });
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

describe("dmRoomsOf", () => {
    it("keeps the direct messages and drops every group room", () => {
        // given
        const rooms = [makeRoom("a"), makeGroupRoom("b"), makeRoom("c")];

        // when
        const next = dmRoomsOf(rooms);

        // then
        expect(ids(next)).toEqual(["a", "c"]);
    });

    it("keeps the order the server gave", () => {
        // given
        const rooms = [makeRoom("c"), makeRoom("a"), makeRoom("b")];

        // when
        const next = dmRoomsOf(rooms);

        // then
        expect(ids(next)).toEqual(["c", "a", "b"]);
    });

    it("hands back nothing when the viewer belongs to group rooms alone", () => {
        // given
        const rooms = [makeGroupRoom("a"), makeGroupRoom("b")];

        // when
        const next = dmRoomsOf(rooms);

        // then
        expect(next).toEqual([]);
    });
});

describe("rosterEventFor", () => {
    interface EventCase {
        name: string;
        rooms: ChatRoom[];
        roomId: string;
        expected: RosterEvent;
    }

    const eventCases: EventCase[] = [
        {
            name: "a conversation already on the roster",
            rooms: [makeRoom("a"), makeGroupRoom("b")],
            roomId: "a",
            expected: "dm",
        },
        {
            name: "a group room the viewer is also in",
            rooms: [makeRoom("a"), makeGroupRoom("b")],
            roomId: "b",
            expected: "otherRoom",
        },
        {
            name: "a room nobody has heard of",
            rooms: [makeRoom("a"), makeGroupRoom("b")],
            roomId: "c",
            expected: "unknown",
        },
        {
            name: "any room at all while the list has not arrived",
            rooms: [],
            roomId: "a",
            expected: "unknown",
        },
    ];

    it.each(eventCases)("calls a message in $name $expected", ({ rooms, roomId, expected }) => {
        // given
        const list = rooms;

        // when
        const event = rosterEventFor(list, roomId);

        // then
        expect(event).toBe(expected);
    });
});

describe("patchRoom", () => {
    it("applies the patch to the room it names and leaves the order alone", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b", { unread: true })];

        // when
        const next = patchRoom(rooms, "b", { unread: false });

        // then
        expect(ids(next)).toEqual(["a", "b"]);
        expect(next[1].unread).toBe(false);
    });

    it("leaves every other room identical, so their rows do not re-render", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b")];

        // when
        const next = patchRoom(rooms, "b", { viewer_muted: true });

        // then
        expect(next[0]).toBe(rooms[0]);
        expect(next[1]).not.toBe(rooms[1]);
    });

    it("changes nothing when the room is not in the list", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b")];

        // when
        const next = patchRoom(rooms, "missing", { unread: true });

        // then
        expect(next).toEqual(rooms);
    });

    it("does not mutate the list it was given", () => {
        // given
        const rooms = [makeRoom("a", { viewer_muted: false })];

        // when
        patchRoom(rooms, "a", { viewer_muted: true });

        // then
        expect(rooms[0].viewer_muted).toBe(false);
    });
});

describe("prependRoom", () => {
    it("puts a conversation the roster has never seen at the front", () => {
        // given
        const rooms = [makeRoom("a")];

        // when
        const next = prependRoom(rooms, makeRoom("b"));

        // then
        expect(ids(next)).toEqual(["b", "a"]);
    });

    it("returns the very same array when the roster already holds the conversation", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b")];

        // when
        const next = prependRoom(rooms, makeRoom("b", { name: "a second copy" }));

        // then
        expect(next).toBe(rooms);
    });

    it("does not mutate the list it was given", () => {
        // given
        const rooms = [makeRoom("a")];

        // when
        prependRoom(rooms, makeRoom("b"));

        // then
        expect(ids(rooms)).toEqual(["a"]);
    });
});

describe("removeRoom", () => {
    it("drops the conversation the viewer deleted", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b"), makeGroupRoom("c")];

        // when
        const next = removeRoom(rooms, "b");

        // then
        expect(ids(next)).toEqual(["a", "c"]);
    });

    it("changes nothing when the conversation is already gone", () => {
        // given
        const rooms = [makeRoom("a")];

        // when
        const next = removeRoom(rooms, "missing");

        // then
        expect(ids(next)).toEqual(["a"]);
    });

    it("does not mutate the list it was given", () => {
        // given
        const rooms = [makeRoom("a"), makeRoom("b")];

        // when
        removeRoom(rooms, "b");

        // then
        expect(ids(rooms)).toEqual(["a", "b"]);
    });
});
