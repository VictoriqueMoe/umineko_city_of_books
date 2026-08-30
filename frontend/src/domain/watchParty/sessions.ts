import type { WatchPartyParticipant, WatchPartySession } from "../../types/api";

export function appendOrReplaceParticipant(
    participants: WatchPartyParticipant[],
    incoming: WatchPartyParticipant,
): WatchPartyParticipant[] {
    const idx = participants.findIndex(p => p.user.id === incoming.user.id);
    if (idx === -1) {
        return [...participants, incoming];
    }
    const next = [...participants];
    next[idx] = incoming;
    return next;
}

export function upsertSession(sessions: WatchPartySession[], incoming: WatchPartySession): WatchPartySession[] {
    const idx = sessions.findIndex(s => s.id === incoming.id);
    if (idx === -1) {
        return [...sessions, incoming];
    }
    const next = [...sessions];
    next[idx] = { ...incoming, participants: incoming.participants };
    return next;
}

export function mapSession(
    sessions: WatchPartySession[],
    sessionId: string,
    fn: (s: WatchPartySession) => WatchPartySession,
): WatchPartySession[] {
    const idx = sessions.findIndex(s => s.id === sessionId);
    if (idx === -1) {
        return sessions;
    }
    const next = [...sessions];
    next[idx] = fn(sessions[idx]);
    return next;
}
