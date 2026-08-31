import { describe, expect, it } from "vitest";
import { makeChatMessage } from "../../test-utils/fixtures";
import type { ChatMessage } from "../../types/api";
import {
    appendIfUnknown,
    beforeCursor,
    dedupeById,
    emptyMessageList,
    MAX_MESSAGE_ID,
    mergeChronological,
    messageListReducer,
    trimToWindow,
    upsertById,
    type MessageListState,
} from "./messageStore";

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({ body: "hello", ...overrides });
}

function makeState(overrides: Partial<MessageListState> = {}): MessageListState {
    return {
        roomId: "room-1",
        messages: [],
        hasMore: false,
        ...overrides,
    };
}

function ids(messages: ChatMessage[]): string[] {
    return messages.map(m => m.id);
}

describe("beforeCursor", () => {
    it("pairs the creation time with the message id", () => {
        // given
        const message = makeMessage({ id: "m7", created_at: "2026-01-01T12:00:00Z" });

        // when
        const cursor = beforeCursor(message);

        // then
        expect(cursor).toBe("2026-01-01T12:00:00Z|m7");
    });

    it("pairs a bare timestamp with the sentinel id when there is no message to anchor on", () => {
        // given
        const createdAt = "2026-01-01T12:00:00Z";

        // when
        const cursor = beforeCursor({ created_at: createdAt, id: MAX_MESSAGE_ID });

        // then
        expect(cursor).toBe("2026-01-01T12:00:00Z|ffffffff-ffff-ffff-ffff-ffffffffffff");
    });
});

describe("dedupeById", () => {
    it("keeps only the incoming messages that are not already known", () => {
        // given
        const known = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];
        const incoming = [makeMessage({ id: "m2" }), makeMessage({ id: "m3" })];

        // when
        const unique = dedupeById(known, incoming);

        // then
        expect(ids(unique)).toEqual(["m3"]);
    });

    it("drops a message the incoming page repeats", () => {
        // given
        const incoming = [makeMessage({ id: "m1" }), makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];

        // when
        const unique = dedupeById([], incoming);

        // then
        expect(ids(unique)).toEqual(["m1", "m2"]);
    });

    it("preserves the incoming order", () => {
        // given
        const incoming = [makeMessage({ id: "m3" }), makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];

        // when
        const unique = dedupeById([], incoming);

        // then
        expect(ids(unique)).toEqual(["m3", "m1", "m2"]);
    });
});

describe("mergeChronological", () => {
    it("interleaves the incoming page into the current one by creation time", () => {
        // given
        const current = [
            makeMessage({ id: "m1", created_at: "2026-01-01T00:00:01Z" }),
            makeMessage({ id: "m3", created_at: "2026-01-01T00:00:03Z" }),
        ];
        const incoming = [makeMessage({ id: "m2", created_at: "2026-01-01T00:00:02Z" })];

        // when
        const merged = mergeChronological(current, incoming);

        // then
        expect(ids(merged)).toEqual(["m1", "m2", "m3"]);
    });

    it("breaks a tie on creation time with the message id", () => {
        // given
        const current = [makeMessage({ id: "b", created_at: "2026-01-01T00:00:01Z" })];
        const incoming = [makeMessage({ id: "a", created_at: "2026-01-01T00:00:01Z" })];

        // when
        const merged = mergeChronological(current, incoming);

        // then
        expect(ids(merged)).toEqual(["a", "b"]);
    });

    it("never duplicates a message the current list already holds", () => {
        // given
        const current = [makeMessage({ id: "m1", created_at: "2026-01-01T00:00:01Z" })];
        const incoming = [makeMessage({ id: "m1", created_at: "2026-01-01T00:00:01Z", body: "resent" })];

        // when
        const merged = mergeChronological(current, incoming);

        // then
        expect(ids(merged)).toEqual(["m1"]);
        expect(merged[0].body).toBe("hello");
    });
});

describe("trimToWindow", () => {
    it("leaves the list alone when there is no window", () => {
        // given
        const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];

        // when
        const result = trimToWindow(messages, undefined);

        // then
        expect(result.messages).toBe(messages);
        expect(result.trimmed).toBe(false);
    });

    it("leaves the list alone when it exactly fills the window", () => {
        // given
        const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];

        // when
        const result = trimToWindow(messages, 2);

        // then
        expect(result.messages).toBe(messages);
        expect(result.trimmed).toBe(false);
    });

    it("keeps the newest messages when the list outgrows the window", () => {
        // given
        const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" }), makeMessage({ id: "m3" })];

        // when
        const result = trimToWindow(messages, 2);

        // then
        expect(ids(result.messages)).toEqual(["m2", "m3"]);
        expect(result.trimmed).toBe(true);
    });
});

describe("appendIfUnknown", () => {
    it("appends a message the list has never seen", () => {
        // given
        const current = [makeMessage({ id: "m1" })];

        // when
        const next = appendIfUnknown(current, makeMessage({ id: "m2" }));

        // then
        expect(ids(next)).toEqual(["m1", "m2"]);
    });

    it("leaves the list untouched when the id is already present", () => {
        // given
        const current = [makeMessage({ id: "m1", body: "hello" })];

        // when
        const next = appendIfUnknown(current, makeMessage({ id: "m1", body: "duplicate delivery" }));

        // then
        expect(next).toBe(current);
    });
});

describe("upsertById", () => {
    it("replaces the message in place when the id is already present", () => {
        // given
        const current = [makeMessage({ id: "m1", body: "hello" }), makeMessage({ id: "m2" })];

        // when
        const next = upsertById(current, makeMessage({ id: "m1", body: "confirmed" }));

        // then
        expect(ids(next)).toEqual(["m1", "m2"]);
        expect(next[0].body).toBe("confirmed");
    });

    it("appends when the id is new", () => {
        // given
        const current = [makeMessage({ id: "m1" })];

        // when
        const next = upsertById(current, makeMessage({ id: "m2" }));

        // then
        expect(ids(next)).toEqual(["m1", "m2"]);
    });
});

describe("messageListReducer", () => {
    it("empties the list when a different room is opened", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" })], hasMore: true });

        // when
        const next = messageListReducer(state, { type: "roomOpened", roomId: "room-2" });

        // then
        expect(next).toEqual(emptyMessageList("room-2"));
    });

    it("keeps the list when the room that is opened is the one already open", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" })], hasMore: true });

        // when
        const next = messageListReducer(state, { type: "roomOpened", roomId: "room-1" });

        // then
        expect(next).toBe(state);
    });

    it("reports more history when the first page is short of the total", () => {
        // given
        const state = emptyMessageList(undefined);

        // when
        const next = messageListReducer(state, {
            type: "historyLoaded",
            roomId: "room-1",
            messages: [makeMessage({ id: "m1" })],
            total: 4,
        });

        // then
        expect(next.hasMore).toBe(true);
        expect(next.roomId).toBe("room-1");
    });

    it("reports no more history when the first page is the whole room", () => {
        // given
        const state = emptyMessageList(undefined);

        // when
        const next = messageListReducer(state, {
            type: "historyLoaded",
            roomId: "room-1",
            messages: [makeMessage({ id: "m1" })],
            total: 1,
        });

        // then
        expect(next.hasMore).toBe(false);
    });

    it("empties the list when the first page fails", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" })], hasMore: true });

        // when
        const next = messageListReducer(state, { type: "historyFailed", roomId: "room-1" });

        // then
        expect(next).toEqual(emptyMessageList("room-1"));
    });

    it("seeds a room with a caller supplied list and no further history", () => {
        // given
        const state = makeState({ hasMore: true });

        // when
        const next = messageListReducer(state, {
            type: "seeded",
            roomId: "room-2",
            messages: [makeMessage({ id: "m1" })],
        });

        // then
        expect(next).toEqual({ roomId: "room-2", messages: [makeMessage({ id: "m1" })], hasMore: false });
    });

    it("appends an inbound message", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" })] });

        // when
        const next = messageListReducer(state, {
            type: "messageReceived",
            roomId: "room-1",
            message: makeMessage({ id: "m2" }),
        });

        // then
        expect(ids(next.messages)).toEqual(["m1", "m2"]);
    });

    it("ignores an inbound message that has already been delivered", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1", body: "hello" })] });

        // when
        const next = messageListReducer(state, {
            type: "messageReceived",
            roomId: "room-1",
            message: makeMessage({ id: "m1", body: "duplicate delivery" }),
        });

        // then
        expect(next.messages).toBe(state.messages);
    });

    it("replaces an optimistic message when the confirmed one is upserted", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1", body: "sending" })] });

        // when
        const next = messageListReducer(state, {
            type: "messageUpserted",
            roomId: "room-1",
            message: makeMessage({ id: "m1", body: "sent" }),
        });

        // then
        expect(next.messages[0].body).toBe("sent");
    });

    it("prepends an older page and drops what it already holds", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m3" })] });

        // when
        const next = messageListReducer(state, {
            type: "olderLoaded",
            roomId: "room-1",
            messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m3" })],
        });

        // then
        expect(ids(next.messages)).toEqual(["m1", "m3"]);
    });

    it("orders a page loaded to close a gap by creation time", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m3", created_at: "2026-01-01T00:00:03Z" })] });

        // when
        const next = messageListReducer(state, {
            type: "gapLoaded",
            roomId: "room-1",
            messages: [
                makeMessage({ id: "m1", created_at: "2026-01-01T00:00:01Z" }),
                makeMessage({ id: "m2", created_at: "2026-01-01T00:00:02Z" }),
            ],
        });

        // then
        expect(ids(next.messages)).toEqual(["m1", "m2", "m3"]);
    });

    it("appends only what a resync brings back that is genuinely new", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" })] });

        // when
        const next = messageListReducer(state, {
            type: "resynced",
            roomId: "room-1",
            messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })],
        });

        // then
        expect(ids(next.messages)).toEqual(["m1", "m2"]);
    });

    it("keeps the same list when a resync brings nothing new", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" })] });

        // when
        const next = messageListReducer(state, {
            type: "resynced",
            roomId: "room-1",
            messages: [makeMessage({ id: "m1" })],
        });

        // then
        expect(next.messages).toBe(state.messages);
    });

    it("runs a patch against the messages the room currently holds", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })] });

        // when
        const next = messageListReducer(state, {
            type: "messagesPatched",
            roomId: "room-1",
            patch: current => current.filter(m => m.id !== "m1"),
        });

        // then
        expect(ids(next.messages)).toEqual(["m2"]);
    });

    it("patches an empty list and retags the state when the action belongs to another room", () => {
        // given
        const state = makeState({ roomId: "room-1", messages: [makeMessage({ id: "m1" })] });

        // when
        const next = messageListReducer(state, {
            type: "messageReceived",
            roomId: "room-2",
            message: makeMessage({ id: "m2", room_id: "room-2" }),
        });

        // then
        expect(next.roomId).toBe("room-2");
        expect(ids(next.messages)).toEqual(["m2"]);
    });

    it("drops the oldest messages and admits there is more history once the window overflows", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })] });

        // when
        const next = messageListReducer(state, {
            type: "messageReceived",
            roomId: "room-1",
            message: makeMessage({ id: "m3" }),
            limit: 2,
        });

        // then
        expect(ids(next.messages)).toEqual(["m2", "m3"]);
        expect(next.hasMore).toBe(true);
    });

    it("keeps every message when the caller passes no window", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })] });

        // when
        const next = messageListReducer(state, {
            type: "messageReceived",
            roomId: "room-1",
            message: makeMessage({ id: "m3" }),
        });

        // then
        expect(ids(next.messages)).toEqual(["m1", "m2", "m3"]);
        expect(next.hasMore).toBe(false);
    });

    it("carries an existing has more flag through a message that changes nothing about it", () => {
        // given
        const state = makeState({ messages: [makeMessage({ id: "m1" })], hasMore: true });

        // when
        const next = messageListReducer(state, {
            type: "messageReceived",
            roomId: "room-1",
            message: makeMessage({ id: "m2" }),
        });

        // then
        expect(next.hasMore).toBe(true);
    });

    it("forgets the has more flag when the action belongs to another room", () => {
        // given
        const state = makeState({ roomId: "room-1", hasMore: true });

        // when
        const next = messageListReducer(state, {
            type: "messageReceived",
            roomId: "room-2",
            message: makeMessage({ id: "m2", room_id: "room-2" }),
        });

        // then
        expect(next.hasMore).toBe(false);
    });

    it("takes the has more flag from the end of a page load", () => {
        // given
        const state = makeState({ hasMore: true });

        // when
        const next = messageListReducer(state, { type: "hasMoreChanged", hasMore: false });

        // then
        expect(next.hasMore).toBe(false);
        expect(next.roomId).toBe("room-1");
    });
});
