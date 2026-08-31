import { describe, expect, it } from "vitest";
import { makeWatchPartySession } from "../../test-utils/fixtures";
import type { User, WatchPartyParticipant, WatchPartySession } from "../../types/api";
import {
    WATCH_PARTY_KICKED_MESSAGE,
    watchPartyReducer,
    type WatchPartyEvent,
    type WatchPartyReducerContext,
    type WatchPartyState,
} from "./reducer";

const ROOM_ID = "room-1";
const OTHER_ROOM_ID = "room-2";
const VIEWER_ID = "user-viewer";
const OTHER_USER_ID = "user-other";
const SESSION_ID = "session-1";
const OTHER_SESSION_ID = "session-2";
const UNKNOWN_SESSION_ID = "session-unknown";
const EMBED_URL = "https://embed.example/session-1";

const defaultContext: WatchPartyReducerContext = { roomId: ROOM_ID, viewerUserId: VIEWER_ID };
const anonymousContext: WatchPartyReducerContext = { roomId: ROOM_ID, viewerUserId: null };

function makeUser(id: string): User {
    return { id, username: id, display_name: id };
}

function participant(id: string, hasControl = false): WatchPartyParticipant {
    return { user: makeUser(id), has_control: hasControl, joined_at: "2026-08-01T10:00:00Z" };
}

function makeSession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return makeWatchPartySession({
        id: SESSION_ID,
        room_id: ROOM_ID,
        started_by: VIEWER_ID,
        controller_id: VIEWER_ID,
        participants: [participant(VIEWER_ID)],
        ...overrides,
    });
}

function makeState(overrides: Partial<WatchPartyState> = {}): WatchPartyState {
    return {
        roomId: ROOM_ID,
        sessions: [],
        enabled: true,
        screenShareEnabled: true,
        activeSessionId: null,
        embedURL: "",
        ...overrides,
    };
}

function otherSession(): WatchPartySession {
    return makeSession({ id: OTHER_SESSION_ID, title: "Ep 5 rewatch", participants: [participant(OTHER_USER_ID)] });
}

interface TransitionCase {
    name: string;
    state: WatchPartyState;
    event: WatchPartyEvent;
    context?: WatchPartyReducerContext;
    expectedState: WatchPartyState;
    expectedError?: string;
    keepsIdentity: boolean;
}

const cases: TransitionCase[] = [
    {
        name: "watch_party_started in another room leaves the state untouched",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_started",
            data: { session: makeSession({ id: "session-elsewhere", room_id: OTHER_ROOM_ID }) },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_started before the room has loaded leaves the state untouched",
        state: makeState({ roomId: null }),
        event: { type: "watch_party_started", data: { session: makeSession() } },
        expectedState: makeState({ roomId: null }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_started while the state still holds another room leaves it untouched",
        state: makeState({ roomId: OTHER_ROOM_ID }),
        event: { type: "watch_party_started", data: { session: makeSession() } },
        expectedState: makeState({ roomId: OTHER_ROOM_ID }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_started adds a session the list has never seen",
        state: makeState({ sessions: [] }),
        event: { type: "watch_party_started", data: { session: makeSession() } },
        expectedState: makeState({ sessions: [makeSession()] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_started replaces a known session in place",
        state: makeState({ sessions: [makeSession(), otherSession()] }),
        event: { type: "watch_party_started", data: { session: makeSession({ title: "Chiru rewatch, take two" }) } },
        expectedState: makeState({
            sessions: [makeSession({ title: "Chiru rewatch, take two" }), otherSession()],
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_ended in another room leaves the state untouched",
        state: makeState({ sessions: [makeSession()], activeSessionId: SESSION_ID, embedURL: EMBED_URL }),
        event: {
            type: "watch_party_ended",
            data: { session_id: SESSION_ID, room_id: OTHER_ROOM_ID, reason: "owner_left" },
        },
        expectedState: makeState({ sessions: [makeSession()], activeSessionId: SESSION_ID, embedURL: EMBED_URL }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_ended while the state still holds another room leaves it untouched",
        state: makeState({ roomId: null }),
        event: { type: "watch_party_ended", data: { session_id: SESSION_ID, room_id: ROOM_ID, reason: "owner_left" } },
        expectedState: makeState({ roomId: null }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_ended drops the session and closes the embed when it was the open one",
        state: makeState({
            sessions: [makeSession(), otherSession()],
            activeSessionId: SESSION_ID,
            embedURL: EMBED_URL,
        }),
        event: { type: "watch_party_ended", data: { session_id: SESSION_ID, room_id: ROOM_ID, reason: "owner_left" } },
        expectedState: makeState({ sessions: [otherSession()] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_ended drops the session and leaves a different open one alone",
        state: makeState({
            sessions: [makeSession(), otherSession()],
            activeSessionId: OTHER_SESSION_ID,
            embedURL: EMBED_URL,
        }),
        event: { type: "watch_party_ended", data: { session_id: SESSION_ID, room_id: ROOM_ID, reason: "expired" } },
        expectedState: makeState({
            sessions: [otherSession()],
            activeSessionId: OTHER_SESSION_ID,
            embedURL: EMBED_URL,
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_ended for a session the list never held still rebuilds the state",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_ended",
            data: { session_id: UNKNOWN_SESSION_ID, room_id: ROOM_ID, reason: "expired" },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_joined while the state still holds another room leaves it untouched",
        state: makeState({ roomId: null }),
        event: {
            type: "watch_party_participant_joined",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, participant: participant(OTHER_USER_ID) },
        },
        expectedState: makeState({ roomId: null }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_participant_joined appends a watcher the session did not have",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_participant_joined",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, participant: participant(OTHER_USER_ID) },
        },
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID), participant(OTHER_USER_ID)] })],
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_joined replaces a watcher the session already had",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_participant_joined",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, participant: participant(VIEWER_ID, true) },
        },
        expectedState: makeState({ sessions: [makeSession({ participants: [participant(VIEWER_ID, true)] })] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_joined keys on the session id and ignores the room in the payload",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_participant_joined",
            data: { session_id: SESSION_ID, room_id: OTHER_ROOM_ID, participant: participant(OTHER_USER_ID) },
        },
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID), participant(OTHER_USER_ID)] })],
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_joined for a session the list never held still rebuilds the state",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_participant_joined",
            data: { session_id: UNKNOWN_SESSION_ID, room_id: ROOM_ID, participant: participant(OTHER_USER_ID) },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_left while the state still holds another room leaves it untouched",
        state: makeState({ roomId: null }),
        event: {
            type: "watch_party_participant_left",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: OTHER_USER_ID },
        },
        expectedState: makeState({ roomId: null }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_participant_left removes another watcher and keeps the party open",
        state: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID), participant(OTHER_USER_ID)] })],
            activeSessionId: SESSION_ID,
            embedURL: EMBED_URL,
        }),
        event: {
            type: "watch_party_participant_left",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: OTHER_USER_ID },
        },
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID)] })],
            activeSessionId: SESSION_ID,
            embedURL: EMBED_URL,
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_left closes the party when the viewer is the one who left",
        state: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID), participant(OTHER_USER_ID)] })],
            activeSessionId: SESSION_ID,
            embedURL: EMBED_URL,
        }),
        event: {
            type: "watch_party_participant_left",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: VIEWER_ID },
        },
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(OTHER_USER_ID)] })],
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_left by the viewer leaves a different open session alone",
        state: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID), participant(OTHER_USER_ID)] })],
            activeSessionId: OTHER_SESSION_ID,
            embedURL: EMBED_URL,
        }),
        event: {
            type: "watch_party_participant_left",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: VIEWER_ID },
        },
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(OTHER_USER_ID)] })],
            activeSessionId: OTHER_SESSION_ID,
            embedURL: EMBED_URL,
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_left never closes the party for an anonymous viewer",
        state: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID), participant(OTHER_USER_ID)] })],
            activeSessionId: SESSION_ID,
            embedURL: EMBED_URL,
        }),
        event: {
            type: "watch_party_participant_left",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: VIEWER_ID },
        },
        context: anonymousContext,
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(OTHER_USER_ID)] })],
            activeSessionId: SESSION_ID,
            embedURL: EMBED_URL,
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_participant_left for a session the list never held still rebuilds the state",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_participant_left",
            data: { session_id: UNKNOWN_SESSION_ID, room_id: ROOM_ID, user_id: OTHER_USER_ID },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_control_changed while the state still holds another room leaves it untouched",
        state: makeState({ roomId: null }),
        event: {
            type: "watch_party_control_changed",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: OTHER_USER_ID, has_control: true },
        },
        expectedState: makeState({ roomId: null }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_control_changed grants control to the named watcher and touches nobody else",
        state: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID, true), participant(OTHER_USER_ID)] })],
        }),
        event: {
            type: "watch_party_control_changed",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: OTHER_USER_ID, has_control: true },
        },
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID, true), participant(OTHER_USER_ID, true)] })],
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_control_changed takes control away from the named watcher",
        state: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID, true), participant(OTHER_USER_ID)] })],
        }),
        event: {
            type: "watch_party_control_changed",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: VIEWER_ID, has_control: false },
        },
        expectedState: makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID), participant(OTHER_USER_ID)] })],
        }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_control_changed for a watcher outside the session leaves the roster alone",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_control_changed",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, user_id: OTHER_USER_ID, has_control: true },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_control_changed for a session the list never held still rebuilds the state",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_control_changed",
            data: { session_id: UNKNOWN_SESSION_ID, room_id: ROOM_ID, user_id: VIEWER_ID, has_control: true },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        keepsIdentity: false,
    },
    {
        name: "watch_party_kicked in another room says nothing",
        state: makeState({ sessions: [makeSession()], activeSessionId: SESSION_ID, embedURL: EMBED_URL }),
        event: {
            type: "watch_party_kicked",
            data: { session_id: SESSION_ID, room_id: OTHER_ROOM_ID, actor_id: OTHER_USER_ID },
        },
        expectedState: makeState({ sessions: [makeSession()], activeSessionId: SESSION_ID, embedURL: EMBED_URL }),
        keepsIdentity: true,
    },
    {
        name: "watch_party_kicked reports the removal before the room state has loaded",
        state: makeState({ roomId: null }),
        event: {
            type: "watch_party_kicked",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, actor_id: OTHER_USER_ID },
        },
        expectedState: makeState({ roomId: null }),
        expectedError: WATCH_PARTY_KICKED_MESSAGE,
        keepsIdentity: true,
    },
    {
        name: "watch_party_kicked from a session that is not the open one only reports it",
        state: makeState({
            sessions: [makeSession(), otherSession()],
            activeSessionId: OTHER_SESSION_ID,
            embedURL: EMBED_URL,
        }),
        event: {
            type: "watch_party_kicked",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, actor_id: OTHER_USER_ID },
        },
        expectedState: makeState({
            sessions: [makeSession(), otherSession()],
            activeSessionId: OTHER_SESSION_ID,
            embedURL: EMBED_URL,
        }),
        expectedError: WATCH_PARTY_KICKED_MESSAGE,
        keepsIdentity: true,
    },
    {
        name: "watch_party_kicked with nothing open only reports it",
        state: makeState({ sessions: [makeSession()] }),
        event: {
            type: "watch_party_kicked",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, actor_id: OTHER_USER_ID, reason: "spam" },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        expectedError: WATCH_PARTY_KICKED_MESSAGE,
        keepsIdentity: true,
    },
    {
        name: "watch_party_kicked from the open session closes the embed and keeps the session listed",
        state: makeState({ sessions: [makeSession()], activeSessionId: SESSION_ID, embedURL: EMBED_URL }),
        event: {
            type: "watch_party_kicked",
            data: { session_id: SESSION_ID, room_id: ROOM_ID, actor_id: OTHER_USER_ID, reason: "spam" },
        },
        expectedState: makeState({ sessions: [makeSession()] }),
        expectedError: WATCH_PARTY_KICKED_MESSAGE,
        keepsIdentity: false,
    },
];

describe("watchPartyReducer", () => {
    it.each(cases)("$name", ({ state, event, context, expectedState, expectedError, keepsIdentity }) => {
        // given the state, the event and the context from the table row

        // when
        const result = watchPartyReducer(state, event, context ?? defaultContext);

        // then
        expect(result.state).toEqual(expectedState);
        expect(result.error).toBe(expectedError);
        expect(result.state === state).toBe(keepsIdentity);
    });

    it("never mutates the state it is handed", () => {
        // given
        const state = makeState({
            sessions: [makeSession({ participants: [participant(VIEWER_ID, true), participant(OTHER_USER_ID)] })],
            activeSessionId: SESSION_ID,
            embedURL: EMBED_URL,
        });
        const before = structuredClone(state);

        // when
        for (const row of cases) {
            watchPartyReducer(state, row.event, row.context ?? defaultContext);
        }

        // then
        expect(state).toEqual(before);
    });
});
