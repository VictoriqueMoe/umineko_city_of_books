import type {
    WatchPartyControlChangedEvent,
    WatchPartyEndedEvent,
    WatchPartyKickedEvent,
    WatchPartyParticipantEvent,
    WatchPartyParticipantLeftEvent,
    WatchPartySession,
    WatchPartyStartedEvent,
} from "../../types/api";
import { appendOrReplaceParticipant, mapSession, upsertSession } from "./sessions";

export const WATCH_PARTY_KICKED_MESSAGE = "You were removed from the watch party.";

export interface WatchPartyState {
    roomId: string | null;
    sessions: WatchPartySession[];
    enabled: boolean;
    screenShareEnabled: boolean;
    activeSessionId: string | null;
    embedURL: string;
}

export interface WatchPartyReducerContext {
    roomId: string;
    viewerUserId: string | null;
}

export type WatchPartyEvent =
    | { type: "watch_party_started"; data: WatchPartyStartedEvent }
    | { type: "watch_party_ended"; data: WatchPartyEndedEvent }
    | { type: "watch_party_participant_joined"; data: WatchPartyParticipantEvent }
    | { type: "watch_party_participant_left"; data: WatchPartyParticipantLeftEvent }
    | { type: "watch_party_control_changed"; data: WatchPartyControlChangedEvent }
    | { type: "watch_party_kicked"; data: WatchPartyKickedEvent };

export interface WatchPartyReduction {
    state: WatchPartyState;
    error?: string;
}

export function watchPartyReducer(
    state: WatchPartyState,
    event: WatchPartyEvent,
    context: WatchPartyReducerContext,
): WatchPartyReduction {
    const { roomId, viewerUserId } = context;

    switch (event.type) {
        case "watch_party_started": {
            const data = event.data;
            if (data.session.room_id !== roomId) {
                return { state };
            }
            if (state.roomId !== roomId) {
                return { state };
            }

            return { state: { ...state, sessions: upsertSession(state.sessions, data.session) } };
        }

        case "watch_party_ended": {
            const data = event.data;
            if (data.room_id !== roomId) {
                return { state };
            }
            if (state.roomId !== roomId) {
                return { state };
            }

            const sessions = state.sessions.filter(s => s.id !== data.session_id);
            const wasActive = state.activeSessionId === data.session_id;

            return {
                state: {
                    ...state,
                    sessions,
                    activeSessionId: wasActive ? null : state.activeSessionId,
                    embedURL: wasActive ? "" : state.embedURL,
                },
            };
        }

        case "watch_party_participant_joined": {
            if (state.roomId !== roomId) {
                return { state };
            }

            const data = event.data;

            return {
                state: {
                    ...state,
                    sessions: mapSession(state.sessions, data.session_id, s => ({
                        ...s,
                        participants: appendOrReplaceParticipant(s.participants, data.participant),
                    })),
                },
            };
        }

        case "watch_party_participant_left": {
            if (state.roomId !== roomId) {
                return { state };
            }

            const data = event.data;
            const sessions = mapSession(state.sessions, data.session_id, s => ({
                ...s,
                participants: s.participants.filter(p => p.user.id !== data.user_id),
            }));

            let activeSessionId = state.activeSessionId;
            let embedURL = state.embedURL;
            if (viewerUserId && data.user_id === viewerUserId && state.activeSessionId === data.session_id) {
                activeSessionId = null;
                embedURL = "";
            }

            return { state: { ...state, sessions, activeSessionId, embedURL } };
        }

        case "watch_party_control_changed": {
            if (state.roomId !== roomId) {
                return { state };
            }

            const data = event.data;

            return {
                state: {
                    ...state,
                    sessions: mapSession(state.sessions, data.session_id, s => ({
                        ...s,
                        participants: s.participants.map(p =>
                            p.user.id === data.user_id ? { ...p, has_control: data.has_control } : p,
                        ),
                    })),
                },
            };
        }

        case "watch_party_kicked": {
            const data = event.data;
            if (data.room_id !== roomId) {
                return { state };
            }

            const error = WATCH_PARTY_KICKED_MESSAGE;
            if (state.roomId !== roomId) {
                return { state, error };
            }
            if (state.activeSessionId !== data.session_id) {
                return { state, error };
            }

            return { state: { ...state, activeSessionId: null, embedURL: "" }, error };
        }
    }
}
