import { describe, expect, it, vi } from "vitest";
import { makeWatchPartySession } from "../../test-utils/fixtures";
import type { User, WatchPartyParticipant, WatchPartySession } from "../../types/api";
import { appendOrReplaceParticipant, mapSession, upsertSession } from "./sessions";

function makeUser(id: string): User {
    return { id, username: id, display_name: id };
}

function participant(id: string, hasControl = false): WatchPartyParticipant {
    return { user: makeUser(id), has_control: hasControl, joined_at: "2026-08-01T10:00:00Z" };
}

function makeSession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return makeWatchPartySession({ participants: [participant("user-viewer")], ...overrides });
}

describe("appendOrReplaceParticipant", () => {
    it("appends a watcher the list does not have", () => {
        // given
        const participants = [participant("user-viewer")];

        // when
        const next = appendOrReplaceParticipant(participants, participant("user-other"));

        // then
        expect(next).toEqual([participant("user-viewer"), participant("user-other")]);
        expect(participants).toEqual([participant("user-viewer")]);
    });

    it("replaces a watcher the list already has at the same position", () => {
        // given
        const participants = [participant("user-viewer"), participant("user-other")];

        // when
        const next = appendOrReplaceParticipant(participants, participant("user-viewer", true));

        // then
        expect(next).toEqual([participant("user-viewer", true), participant("user-other")]);
    });

    it("returns a new array when it replaces", () => {
        // given
        const participants = [participant("user-viewer")];

        // when
        const next = appendOrReplaceParticipant(participants, participant("user-viewer", true));

        // then
        expect(next).not.toBe(participants);
    });
});

describe("upsertSession", () => {
    it("appends a session the list does not have", () => {
        // given
        const sessions = [makeSession()];

        // when
        const next = upsertSession(sessions, makeSession({ id: "session-2" }));

        // then
        expect(next).toEqual([makeSession(), makeSession({ id: "session-2" })]);
        expect(sessions).toEqual([makeSession()]);
    });

    it("replaces a session the list already has at the same position", () => {
        // given
        const sessions = [makeSession(), makeSession({ id: "session-2" })];

        // when
        const next = upsertSession(sessions, makeSession({ title: "Chiru rewatch, take two" }));

        // then
        expect(next).toEqual([makeSession({ title: "Chiru rewatch, take two" }), makeSession({ id: "session-2" })]);
    });

    it("takes the incoming roster wholesale rather than merging it", () => {
        // given
        const sessions = [makeSession({ participants: [participant("user-viewer"), participant("user-other")] })];

        // when
        const next = upsertSession(sessions, makeSession({ participants: [] }));

        // then
        expect(next[0].participants).toEqual([]);
    });
});

describe("mapSession", () => {
    it("applies the mapper to the matching session only", () => {
        // given
        const sessions = [makeSession(), makeSession({ id: "session-2" })];

        // when
        const next = mapSession(sessions, "session-2", s => ({ ...s, title: "renamed" }));

        // then
        expect(next).toEqual([makeSession(), makeSession({ id: "session-2", title: "renamed" })]);
    });

    it("returns the same array when the session id is unknown", () => {
        // given
        const sessions = [makeSession()];

        // when
        const next = mapSession(sessions, "session-unknown", s => ({ ...s, title: "renamed" }));

        // then
        expect(next).toBe(sessions);
    });

    it("does not call the mapper when the session id is unknown", () => {
        // given
        const sessions = [makeSession()];
        const fn = vi.fn((s: WatchPartySession) => s);

        // when
        mapSession(sessions, "session-unknown", fn);

        // then
        expect(fn).not.toHaveBeenCalled();
    });
});
