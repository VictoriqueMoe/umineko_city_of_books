import { useCallback, useEffect, useRef, useState } from "react";
import { useTheme } from "./useTheme";
import { isLightTheme } from "../domain/themes";
import { sendWatchPartyLeaveBeacon } from "../api/beacons/watchPartyLeave";
import { resolveOptimalRegion } from "../api/hyperbeam/region";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { reportClientError } from "../api/telemetry";
import { useWatchParties } from "./queries/watchParty";
import {
    useEndWatchParty,
    useIdentifyWatchPartyParticipant,
    useJoinWatchParty,
    useKickWatchPartyParticipant,
    useLeaveWatchParty,
    useStartWatchParty,
    useTransferWatchPartyControl,
} from "./mutations/watchParty";
import { watchPartyReducer, type WatchPartyState } from "../domain/watchParty/reducer";
import { upsertSession } from "../domain/watchParty/sessions";
import { errorMessage } from "../utils/errorMessage";
import type { WatchPartySession } from "../types/api";

export interface ActiveWatchPartySession {
    session: WatchPartySession;
    embedURL: string;
    hasControl: boolean;
}

interface UseWatchPartyResult {
    enabled: boolean;
    screenShareEnabled: boolean;
    loaded: boolean;
    sessions: WatchPartySession[];
    activeSession: ActiveWatchPartySession | null;
    openSessionId: string | null;
    error: string | null;
    refresh: () => Promise<void>;
    start: (opts: {
        title?: string;
        startURL?: string;
        type?: "hyperbeam" | "screenshare";
    }) => Promise<WatchPartySession | null>;
    join: (sessionId: string) => Promise<void>;
    leave: () => Promise<void>;
    end: () => Promise<void>;
    transferControl: (userId: string) => Promise<void>;
    kick: (userId: string) => Promise<void>;
    identify: (identifier: string) => Promise<void>;
    openExisting: (sessionId: string) => void;
    close: () => void;
    clearError: () => void;
}

interface RoomScopedError {
    roomId: string | null;
    message: string;
}

const emptyState: WatchPartyState = {
    roomId: null,
    sessions: [],
    enabled: false,
    screenShareEnabled: false,
    activeSessionId: null,
    embedURL: "",
};

const WATCH_PARTY_EVENTS = [
    REALTIME_EVENTS.WATCH_PARTY_STARTED,
    REALTIME_EVENTS.WATCH_PARTY_ENDED,
    REALTIME_EVENTS.WATCH_PARTY_PARTICIPANT_JOINED,
    REALTIME_EVENTS.WATCH_PARTY_PARTICIPANT_LEFT,
    REALTIME_EVENTS.WATCH_PARTY_CONTROL_CHANGED,
    REALTIME_EVENTS.WATCH_PARTY_KICKED,
] as const;

const HIDDEN_LEAVE_AFTER_MS = 10 * 60 * 1000;

export function useWatchParty(roomId: string | null, viewerUserId: string | null): UseWatchPartyResult {
    const { theme } = useTheme();
    const [data, setData] = useState<WatchPartyState>(emptyState);
    const [raised, setRaised] = useState<RoomScopedError | null>(null);
    const [dismissedLoadError, setDismissedLoadError] = useState("");
    const dataRef = useRef<WatchPartyState>(emptyState);
    const activeIdRef = useRef<string | null>(null);
    const roomIdRef = useRef<string | null>(null);

    const {
        sessions: listedSessions,
        enabled: listedEnabled,
        screenShareEnabled: listedScreenShare,
        loaded: listLoaded,
        error: listError,
        refresh,
    } = useWatchParties(roomId);

    const { mutateAsync: startParty } = useStartWatchParty(roomId);
    const { mutateAsync: joinParty } = useJoinWatchParty(roomId);
    const { mutateAsync: leaveParty } = useLeaveWatchParty(roomId);
    const { mutateAsync: endParty } = useEndWatchParty(roomId);
    const { mutateAsync: transferParty } = useTransferWatchPartyControl(roomId);
    const { mutateAsync: kickParty } = useKickWatchPartyParticipant(roomId);
    const { mutateAsync: identifyParty } = useIdentifyWatchPartyParticipant(roomId);

    const writeState = useCallback((next: WatchPartyState) => {
        dataRef.current = next;
        setData(next);
    }, []);

    const raiseError = useCallback(
        (message: string) => {
            setRaised({ roomId, message });
        },
        [roomId],
    );

    const clearError = useCallback(() => {
        setRaised(null);
        setDismissedLoadError(listError);
    }, [listError]);

    const loadError = listError && listError !== dismissedLoadError ? listError : "";
    const error = (raised && raised.roomId === roomId ? raised.message : null) ?? (loadError || null);

    const stateMatches = data.roomId === roomId;
    const sessions = stateMatches ? data.sessions : [];
    const enabled = stateMatches ? data.enabled : false;
    const screenShareEnabled = stateMatches ? data.screenShareEnabled : false;
    const activeSessionId = stateMatches ? data.activeSessionId : null;

    useEffect(() => {
        activeIdRef.current = activeSessionId;
    }, [activeSessionId]);

    useEffect(() => {
        roomIdRef.current = roomId;
    }, [roomId]);

    useEffect(() => {
        if (!roomId || !listLoaded) {
            return;
        }

        const previous = dataRef.current;
        const sameRoom = previous.roomId === roomId;

        writeState({
            roomId,
            sessions: listedSessions,
            enabled: listedEnabled,
            screenShareEnabled: listedScreenShare,
            activeSessionId: sameRoom ? previous.activeSessionId : null,
            embedURL: sameRoom ? previous.embedURL : "",
        });
    }, [roomId, listLoaded, listedSessions, listedEnabled, listedScreenShare, writeState]);

    useEffect(() => {
        if (!roomId || !activeSessionId) {
            return;
        }
        const sid = activeSessionId;
        const rid = roomId;

        const handleBeforeUnload = () => {
            sendWatchPartyLeaveBeacon(rid, sid);
        };

        let hiddenTimer: ReturnType<typeof setTimeout> | null = null;
        const handleVisibility = () => {
            if (document.visibilityState === "hidden") {
                if (hiddenTimer) {
                    return;
                }
                hiddenTimer = setTimeout(() => {
                    hiddenTimer = null;
                    if (activeIdRef.current === sid && roomIdRef.current === rid) {
                        sendWatchPartyLeaveBeacon(rid, sid);
                    }
                }, HIDDEN_LEAVE_AFTER_MS);
                return;
            }
            if (hiddenTimer) {
                clearTimeout(hiddenTimer);
                hiddenTimer = null;
            }
        };

        window.addEventListener("beforeunload", handleBeforeUnload);
        document.addEventListener("visibilitychange", handleVisibility);
        return () => {
            window.removeEventListener("beforeunload", handleBeforeUnload);
            document.removeEventListener("visibilitychange", handleVisibility);
            if (hiddenTimer) {
                clearTimeout(hiddenTimer);
            }
        };
    }, [roomId, activeSessionId]);

    useRealtimeEvent(WATCH_PARTY_EVENTS, event => {
        if (!roomId) {
            return;
        }

        const previous = dataRef.current;
        const reduction = watchPartyReducer(previous, event, { roomId, viewerUserId });

        if (reduction.error) {
            raiseError(reduction.error);
        }

        if (reduction.state !== previous) {
            writeState(reduction.state);
        }
    });

    const start = useCallback(
        async (opts: { title?: string; startURL?: string; type?: "hyperbeam" | "screenshare" }) => {
            if (!roomId) {
                return null;
            }

            clearError();

            try {
                const partyType = opts.type ?? "hyperbeam";

                let region = "";
                if (partyType !== "screenshare") {
                    region = (await resolveOptimalRegion()) ?? "";
                }

                const resp = await startParty({
                    start_url: opts.startURL,
                    region: region || undefined,
                    title: opts.title,
                    type: partyType,
                    light: isLightTheme(theme) || undefined,
                });

                const previous = dataRef.current;
                const sameRoom = previous.roomId === roomId;

                writeState({
                    roomId,
                    sessions: upsertSession(sameRoom ? previous.sessions : [], resp.session),
                    enabled: true,
                    screenShareEnabled: sameRoom ? previous.screenShareEnabled : true,
                    activeSessionId: resp.session.id,
                    embedURL: resp.embed_url,
                });

                return resp.session;
            } catch (thrown: unknown) {
                raiseError(errorMessage(thrown, "Failed to start watch party"));
                throw thrown;
            }
        },
        [roomId, theme, startParty, writeState, raiseError, clearError],
    );

    const join = useCallback(
        async (sessionId: string) => {
            if (!roomId) {
                return;
            }

            clearError();

            try {
                const resp = await joinParty(sessionId);

                const previous = dataRef.current;
                const sameRoom = previous.roomId === roomId;

                writeState({
                    roomId,
                    sessions: upsertSession(sameRoom ? previous.sessions : [], resp.session),
                    enabled: true,
                    screenShareEnabled: sameRoom ? previous.screenShareEnabled : true,
                    activeSessionId: resp.session.id,
                    embedURL: resp.embed_url,
                });
            } catch (thrown: unknown) {
                raiseError(errorMessage(thrown, "Failed to join watch party"));
                throw thrown;
            }
        },
        [roomId, joinParty, writeState, raiseError, clearError],
    );

    const leave = useCallback(async () => {
        if (!roomId || !activeSessionId) {
            return;
        }

        clearError();

        try {
            await leaveParty(activeSessionId);
        } catch (thrown: unknown) {
            raiseError(errorMessage(thrown, "Failed to leave watch party"));
            throw thrown;
        } finally {
            const previous = dataRef.current;
            if (previous.roomId === roomId) {
                writeState({ ...previous, activeSessionId: null, embedURL: "" });
            }
        }
    }, [roomId, activeSessionId, leaveParty, writeState, raiseError, clearError]);

    const end = useCallback(async () => {
        if (!roomId || !activeSessionId) {
            return;
        }

        clearError();

        try {
            await endParty(activeSessionId);
        } catch (thrown: unknown) {
            raiseError(errorMessage(thrown, "Failed to end watch party"));
            throw thrown;
        }
    }, [roomId, activeSessionId, endParty, raiseError, clearError]);

    const transferControl = useCallback(
        async (userId: string) => {
            if (!roomId || !activeSessionId) {
                return;
            }

            clearError();

            try {
                await transferParty({ sessionId: activeSessionId, userId });
            } catch (thrown: unknown) {
                raiseError(errorMessage(thrown, "Failed to transfer control"));
                throw thrown;
            }
        },
        [roomId, activeSessionId, transferParty, raiseError, clearError],
    );

    const kick = useCallback(
        async (userId: string) => {
            if (!roomId || !activeSessionId) {
                return;
            }

            clearError();

            try {
                await kickParty({ sessionId: activeSessionId, userId });
            } catch (thrown: unknown) {
                raiseError(errorMessage(thrown, "Failed to kick participant"));
                throw thrown;
            }
        },
        [roomId, activeSessionId, kickParty, raiseError, clearError],
    );

    const identify = useCallback(
        async (identifier: string) => {
            if (!roomId || !activeSessionId || !identifier) {
                return;
            }

            try {
                await identifyParty({ sessionId: activeSessionId, identifier });
            } catch (thrown: unknown) {
                reportClientError(thrown, { source: "recoverable" });
            }
        },
        [roomId, activeSessionId, identifyParty],
    );

    const openExisting = useCallback(
        (sessionId: string) => {
            const previous = dataRef.current;
            if (previous.roomId !== roomId || previous.activeSessionId === sessionId) {
                return;
            }

            writeState({ ...previous, activeSessionId: sessionId });
        },
        [roomId, writeState],
    );

    const close = useCallback(() => {
        const previous = dataRef.current;
        if (previous.roomId !== roomId) {
            return;
        }

        writeState({ ...previous, activeSessionId: null, embedURL: "" });
    }, [roomId, writeState]);

    const activeSession = resolveActiveSession(sessions, activeSessionId, data.embedURL, viewerUserId);

    return {
        enabled,
        screenShareEnabled,
        loaded: !!roomId && stateMatches,
        sessions,
        activeSession,
        openSessionId: activeSessionId,
        error,
        refresh,
        start,
        join,
        leave,
        end,
        transferControl,
        kick,
        identify,
        openExisting,
        close,
        clearError,
    };
}

function resolveActiveSession(
    sessions: WatchPartySession[],
    activeSessionId: string | null,
    embedURL: string,
    viewerUserId: string | null,
): ActiveWatchPartySession | null {
    if (!activeSessionId) {
        return null;
    }

    const session = sessions.find(s => s.id === activeSessionId);
    if (!session) {
        return null;
    }

    const viewerParticipant = viewerUserId ? session.participants.find(p => p.user.id === viewerUserId) : undefined;

    return { session, embedURL, hasControl: viewerParticipant?.has_control ?? false };
}
