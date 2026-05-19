import { useCallback, useEffect, useRef, useState } from "react";
import { useNotifications } from "../../../hooks/useNotifications";
import {
    endWatchParty as endWatchPartyApi,
    identifyWatchPartyParticipant as identifyWatchPartyParticipantApi,
    joinWatchParty as joinWatchPartyApi,
    kickWatchPartyParticipant as kickWatchPartyParticipantApi,
    leaveWatchParty as leaveWatchPartyApi,
    listWatchParties,
    listWatchPartyMessages,
    sendWatchPartyMessage as sendWatchPartyMessageApi,
    startWatchParty as startWatchPartyApi,
    transferWatchPartyControl as transferWatchPartyControlApi,
} from "../../../api/endpoints";
import type {
    WatchPartyControlChangedEvent,
    WatchPartyEndedEvent,
    WatchPartyKickedEvent,
    WatchPartyMessage,
    WatchPartyMessageEvent,
    WatchPartyParticipant,
    WatchPartyParticipantEvent,
    WatchPartyParticipantLeftEvent,
    WatchPartySession,
    WatchPartyStartedEvent,
    WSMessage,
} from "../../../types/api";
import { resolveOptimalRegion } from "./hyperbeamRegion";

export interface ActiveWatchPartySession {
    session: WatchPartySession;
    embedURL: string;
    messages: WatchPartyMessage[];
    hasControl: boolean;
}

interface UseWatchPartyResult {
    enabled: boolean;
    sessions: WatchPartySession[];
    activeSession: ActiveWatchPartySession | null;
    openSessionId: string | null;
    error: string | null;
    start: (opts: { title?: string; startURL?: string }) => Promise<WatchPartySession | null>;
    join: (sessionId: string) => Promise<void>;
    leave: () => Promise<void>;
    end: () => Promise<void>;
    transferControl: (userId: string) => Promise<void>;
    kick: (userId: string) => Promise<void>;
    identify: (identifier: string) => Promise<void>;
    sendMessage: (body: string) => Promise<void>;
    openExisting: (sessionId: string) => void;
    close: () => void;
    clearError: () => void;
}

interface RoomScopedState {
    roomId: string | null;
    sessions: WatchPartySession[];
    enabled: boolean;
    activeSessionId: string | null;
    embedURL: string;
    messages: WatchPartyMessage[];
}

const emptyState: RoomScopedState = {
    roomId: null,
    sessions: [],
    enabled: false,
    activeSessionId: null,
    embedURL: "",
    messages: [],
};

export function useWatchParty(roomId: string | null, viewerUserId: string | null): UseWatchPartyResult {
    const { addWSListener } = useNotifications();
    const [data, setData] = useState<RoomScopedState>(emptyState);
    const [error, setError] = useState<string | null>(null);
    const activeIdRef = useRef<string | null>(null);

    const stateMatches = data.roomId === roomId;
    const sessions = stateMatches ? data.sessions : [];
    const enabled = stateMatches ? data.enabled : false;
    const activeSessionId = stateMatches ? data.activeSessionId : null;

    useEffect(() => {
        activeIdRef.current = activeSessionId;
    }, [activeSessionId]);

    useEffect(() => {
        if (!roomId) {
            return;
        }
        let cancelled = false;
        listWatchParties(roomId)
            .then(resp => {
                if (cancelled) {
                    return;
                }
                setError(null);
                setData({
                    roomId,
                    sessions: resp.sessions,
                    enabled: resp.enabled,
                    activeSessionId: null,
                    embedURL: "",
                    messages: [],
                });
            })
            .catch((err: unknown) => {
                if (cancelled) {
                    return;
                }
                setError(messageFromError(err, "Failed to load watch parties"));
            });
        return () => {
            cancelled = true;
        };
    }, [roomId]);

    useEffect(() => {
        if (!roomId) {
            return;
        }
        const unsubscribe = addWSListener((msg: WSMessage) => {
            if (!msg.type.startsWith("watch_party_")) {
                return;
            }
            if (msg.type === "watch_party_started") {
                const payload = msg.data as WatchPartyStartedEvent;
                if (payload.session.room_id !== roomId) {
                    return;
                }
                setData(prev => {
                    if (prev.roomId !== roomId) {
                        return prev;
                    }
                    return { ...prev, sessions: upsertSession(prev.sessions, payload.session) };
                });
                return;
            }
            if (msg.type === "watch_party_ended") {
                const payload = msg.data as WatchPartyEndedEvent;
                if (payload.room_id !== roomId) {
                    return;
                }
                setData(prev => {
                    if (prev.roomId !== roomId) {
                        return prev;
                    }
                    const sessions = prev.sessions.filter(s => s.id !== payload.session_id);
                    const wasActive = prev.activeSessionId === payload.session_id;
                    return {
                        ...prev,
                        sessions,
                        activeSessionId: wasActive ? null : prev.activeSessionId,
                        embedURL: wasActive ? "" : prev.embedURL,
                        messages: wasActive ? [] : prev.messages,
                    };
                });
                return;
            }
            if (msg.type === "watch_party_participant_joined") {
                const payload = msg.data as WatchPartyParticipantEvent;
                setData(prev => {
                    if (prev.roomId !== roomId) {
                        return prev;
                    }
                    return {
                        ...prev,
                        sessions: mapSession(prev.sessions, payload.session_id, s => ({
                            ...s,
                            participants: appendOrReplaceParticipant(s.participants, payload.participant),
                        })),
                    };
                });
                return;
            }
            if (msg.type === "watch_party_participant_left") {
                const payload = msg.data as WatchPartyParticipantLeftEvent;
                setData(prev => {
                    if (prev.roomId !== roomId) {
                        return prev;
                    }
                    const sessions = mapSession(prev.sessions, payload.session_id, s => ({
                        ...s,
                        participants: s.participants.filter(p => p.user.id !== payload.user_id),
                    }));
                    let activeSessionId = prev.activeSessionId;
                    let embedURL = prev.embedURL;
                    if (
                        viewerUserId &&
                        payload.user_id === viewerUserId &&
                        prev.activeSessionId === payload.session_id
                    ) {
                        activeSessionId = null;
                        embedURL = "";
                    }
                    return { ...prev, sessions, activeSessionId, embedURL };
                });
                return;
            }
            if (msg.type === "watch_party_control_changed") {
                const payload = msg.data as WatchPartyControlChangedEvent;
                setData(prev => {
                    if (prev.roomId !== roomId) {
                        return prev;
                    }
                    return {
                        ...prev,
                        sessions: mapSession(prev.sessions, payload.session_id, s => ({
                            ...s,
                            participants: s.participants.map(p =>
                                p.user.id === payload.user_id ? { ...p, has_control: payload.has_control } : p,
                            ),
                        })),
                    };
                });
                return;
            }
            if (msg.type === "watch_party_message") {
                const payload = msg.data as WatchPartyMessageEvent;
                if (activeIdRef.current !== payload.session_id) {
                    return;
                }
                setData(prev => {
                    if (prev.activeSessionId !== payload.session_id) {
                        return prev;
                    }
                    if (prev.messages.some(m => m.id === payload.message.id)) {
                        return prev;
                    }
                    return { ...prev, messages: [...prev.messages, payload.message] };
                });
                return;
            }
            if (msg.type === "watch_party_kicked") {
                const payload = msg.data as WatchPartyKickedEvent;
                if (payload.room_id !== roomId) {
                    return;
                }
                setError("You were removed from the watch party.");
                setData(prev => {
                    if (prev.roomId !== roomId) {
                        return prev;
                    }
                    if (prev.activeSessionId !== payload.session_id) {
                        return prev;
                    }
                    return { ...prev, activeSessionId: null, embedURL: "", messages: [] };
                });
                return;
            }
        });
        return unsubscribe;
    }, [addWSListener, roomId, viewerUserId]);

    const start = useCallback(
        async (opts: { title?: string; startURL?: string }) => {
            if (!roomId) {
                return null;
            }
            setError(null);
            try {
                const region = await resolveOptimalRegion();
                const resp = await startWatchPartyApi(roomId, {
                    start_url: opts.startURL,
                    region: region || undefined,
                    title: opts.title,
                });
                let messages: WatchPartyMessage[] = [];
                try {
                    const msgsResp = await listWatchPartyMessages(roomId, resp.session.id);
                    messages = msgsResp.messages;
                } catch {
                    messages = [];
                }
                setData(prev => ({
                    roomId,
                    sessions: upsertSession(prev.roomId === roomId ? prev.sessions : [], resp.session),
                    enabled: true,
                    activeSessionId: resp.session.id,
                    embedURL: resp.embed_url,
                    messages,
                }));
                return resp.session;
            } catch (err: unknown) {
                setError(messageFromError(err, "Failed to start watch party"));
                throw err;
            }
        },
        [roomId],
    );

    const join = useCallback(
        async (sessionId: string) => {
            if (!roomId) {
                return;
            }
            setError(null);
            try {
                const resp = await joinWatchPartyApi(roomId, sessionId);
                let messages: WatchPartyMessage[] = [];
                try {
                    const msgsResp = await listWatchPartyMessages(roomId, sessionId);
                    messages = msgsResp.messages;
                } catch {
                    messages = [];
                }
                setData(prev => ({
                    roomId,
                    sessions: upsertSession(prev.roomId === roomId ? prev.sessions : [], resp.session),
                    enabled: true,
                    activeSessionId: resp.session.id,
                    embedURL: resp.embed_url,
                    messages,
                }));
            } catch (err: unknown) {
                setError(messageFromError(err, "Failed to join watch party"));
                throw err;
            }
        },
        [roomId],
    );

    const leave = useCallback(async () => {
        if (!roomId || !activeSessionId) {
            return;
        }
        const sid = activeSessionId;
        setError(null);
        try {
            await leaveWatchPartyApi(roomId, sid);
        } catch (err: unknown) {
            setError(messageFromError(err, "Failed to leave watch party"));
            throw err;
        } finally {
            setData(prev => {
                if (prev.roomId !== roomId) {
                    return prev;
                }
                return { ...prev, activeSessionId: null, embedURL: "", messages: [] };
            });
        }
    }, [roomId, activeSessionId]);

    const end = useCallback(async () => {
        if (!roomId || !activeSessionId) {
            return;
        }
        setError(null);
        try {
            await endWatchPartyApi(roomId, activeSessionId);
        } catch (err: unknown) {
            setError(messageFromError(err, "Failed to end watch party"));
            throw err;
        }
    }, [roomId, activeSessionId]);

    const transferControl = useCallback(
        async (userId: string) => {
            if (!roomId || !activeSessionId) {
                return;
            }
            setError(null);
            try {
                await transferWatchPartyControlApi(roomId, activeSessionId, userId);
            } catch (err: unknown) {
                setError(messageFromError(err, "Failed to transfer control"));
                throw err;
            }
        },
        [roomId, activeSessionId],
    );

    const kick = useCallback(
        async (userId: string) => {
            if (!roomId || !activeSessionId) {
                return;
            }
            setError(null);
            try {
                await kickWatchPartyParticipantApi(roomId, activeSessionId, userId);
            } catch (err: unknown) {
                setError(messageFromError(err, "Failed to kick participant"));
                throw err;
            }
        },
        [roomId, activeSessionId],
    );

    const identify = useCallback(
        async (identifier: string) => {
            if (!roomId || !activeSessionId || !identifier) {
                return;
            }
            try {
                await identifyWatchPartyParticipantApi(roomId, activeSessionId, identifier);
            } catch (err: unknown) {
                console.warn("watch party identify failed", err);
            }
        },
        [roomId, activeSessionId],
    );

    const sendMessage = useCallback(
        async (body: string) => {
            if (!roomId || !activeSessionId) {
                return;
            }
            setError(null);
            try {
                await sendWatchPartyMessageApi(roomId, activeSessionId, body);
            } catch (err: unknown) {
                setError(messageFromError(err, "Failed to send message"));
                throw err;
            }
        },
        [roomId, activeSessionId],
    );

    const openExisting = useCallback(
        (sessionId: string) => {
            setData(prev => {
                if (prev.roomId !== roomId) {
                    return prev;
                }
                if (prev.activeSessionId === sessionId) {
                    return prev;
                }
                return { ...prev, activeSessionId: sessionId };
            });
        },
        [roomId],
    );

    const close = useCallback(() => {
        setData(prev => {
            if (prev.roomId !== roomId) {
                return prev;
            }
            return { ...prev, activeSessionId: null, embedURL: "", messages: [] };
        });
    }, [roomId]);

    const clearError = useCallback(() => setError(null), []);

    const activeSession: ActiveWatchPartySession | null = (() => {
        if (!activeSessionId) {
            return null;
        }
        const session = sessions.find(s => s.id === activeSessionId);
        if (!session) {
            return null;
        }
        const viewerParticipant = viewerUserId ? session.participants.find(p => p.user.id === viewerUserId) : undefined;
        const hasControl = viewerParticipant?.has_control ?? false;
        return {
            session,
            embedURL: data.embedURL,
            messages: data.messages,
            hasControl,
        };
    })();

    return {
        enabled,
        sessions,
        activeSession,
        openSessionId: activeSessionId,
        error,
        start,
        join,
        leave,
        end,
        transferControl,
        kick,
        identify,
        sendMessage,
        openExisting,
        close,
        clearError,
    };
}

function appendOrReplaceParticipant(
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

function upsertSession(sessions: WatchPartySession[], incoming: WatchPartySession): WatchPartySession[] {
    const idx = sessions.findIndex(s => s.id === incoming.id);
    if (idx === -1) {
        return [...sessions, incoming];
    }
    const next = [...sessions];
    next[idx] = { ...incoming, participants: incoming.participants };
    return next;
}

function mapSession(
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

function messageFromError(err: unknown, fallback: string): string {
    if (err instanceof Error && err.message) {
        return err.message;
    }
    return fallback;
}
