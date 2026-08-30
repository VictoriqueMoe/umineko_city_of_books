import { describe, expect, it } from "vitest";
import type { MysteryAttempt } from "../../types/api";
import { readCursorKey, unreadAuthorIds } from "./unread";

const battler = { id: "player-1", username: "battler", display_name: "Battler" };
const ange = { id: "player-2", username: "ange", display_name: "Ange" };

function makeAttempt(overrides: Partial<MysteryAttempt> = {}): MysteryAttempt {
    return {
        id: "attempt-1",
        author: battler,
        body: "The chain was fixed after the fact.",
        is_winner: false,
        vote_score: 0,
        created_at: "2026-07-01T11:00:00Z",
        ...overrides,
    };
}

describe("readCursorKey", () => {
    it("names the cursor after the mystery it belongs to", () => {
        // given
        const mysteryId = "mystery-1";

        // when
        const key = readCursorKey(mysteryId);

        // then
        expect(key).toBe("mystery-read-cursor-mystery-1");
    });

    it("gives two mysteries two separate cursors", () => {
        // given
        const first = "mystery-1";
        const second = "mystery-2";

        // when
        const keys = [readCursorKey(first), readCursorKey(second)];

        // then
        expect(keys[0]).not.toBe(keys[1]);
    });
});

describe("unreadAuthorIds", () => {
    it("marks nobody unread before a cursor has ever been laid down", () => {
        // given
        const attempts = [makeAttempt({ created_at: "2026-07-01T11:00:00Z" })];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: null, viewerId: "gm-1" });

        // then
        expect(unread.size).toBe(0);
    });

    it("marks nobody unread when the stored cursor is an empty string", () => {
        // given
        const attempts = [makeAttempt({ created_at: "2026-07-01T11:00:00Z" })];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "", viewerId: "gm-1" });

        // then
        expect(unread.size).toBe(0);
    });

    it("marks the authors who moved after the cursor", () => {
        // given
        const attempts = [
            makeAttempt({ id: "a1", created_at: "2026-07-01T09:00:00Z" }),
            makeAttempt({ id: "a2", author: ange, created_at: "2026-07-01T13:00:00Z" }),
        ];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01T12:00:00Z", viewerId: "gm-1" });

        // then
        expect(Array.from(unread)).toEqual(["player-2"]);
    });

    it("leaves an author read when their attempt lands exactly on the cursor", () => {
        // given
        const attempts = [makeAttempt({ created_at: "2026-07-01T12:00:00Z" })];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01T12:00:00Z", viewerId: "gm-1" });

        // then
        expect(unread.size).toBe(0);
    });

    it("never marks the viewer unread against their own attempt", () => {
        // given
        const attempts = [makeAttempt({ author: ange, created_at: "2026-07-01T13:00:00Z" })];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01T12:00:00Z", viewerId: ange.id });

        // then
        expect(unread.size).toBe(0);
    });

    it("names an author once however many attempts they made", () => {
        // given
        const attempts = [
            makeAttempt({ id: "a1", created_at: "2026-07-01T13:00:00Z" }),
            makeAttempt({ id: "a2", created_at: "2026-07-01T14:00:00Z" }),
        ];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01T12:00:00Z", viewerId: "gm-1" });

        // then
        expect(Array.from(unread)).toEqual(["player-1"]);
    });

    it("reads a cursor the server wrote without a zone as UTC", () => {
        // given
        const attempts = [makeAttempt({ created_at: "2026-07-01T13:00:00Z" })];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01 14:00:00", viewerId: "gm-1" });

        // then
        expect(unread.size).toBe(0);
    });

    it("treats an unreadable cursor as the beginning of time so nothing is missed", () => {
        // given
        const attempts = [makeAttempt({ created_at: "2026-07-01T11:00:00Z" })];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "not a date", viewerId: "gm-1" });

        // then
        expect(Array.from(unread)).toEqual(["player-1"]);
    });

    it("treats an attempt with an unreadable date as older than any cursor", () => {
        // given
        const attempts = [makeAttempt({ created_at: "who knows" })];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01T12:00:00Z", viewerId: "gm-1" });

        // then
        expect(unread.size).toBe(0);
    });

    it("marks every author unread for a signed out reader", () => {
        // given
        const attempts = [
            makeAttempt({ id: "a1", created_at: "2026-07-01T13:00:00Z" }),
            makeAttempt({ id: "a2", author: ange, created_at: "2026-07-01T13:00:00Z" }),
        ];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01T12:00:00Z", viewerId: null });

        // then
        expect(Array.from(unread)).toEqual(["player-1", "player-2"]);
    });

    it("finds nothing unread on an empty board", () => {
        // given
        const attempts: MysteryAttempt[] = [];

        // when
        const unread = unreadAuthorIds({ attempts, cursor: "2026-07-01T12:00:00Z", viewerId: "gm-1" });

        // then
        expect(unread.size).toBe(0);
    });
});
