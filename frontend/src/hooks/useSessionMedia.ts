import { useCallback, useEffect, useRef, useState } from "react";
import type { Room } from "livekit-client";

import { connectRoom, disconnectRoom } from "../api/livekit/connect";
import { setScreenShareEnabled, type ScreenShareMode } from "../api/livekit/screenShare";
import { reportClientError } from "../api/telemetry";
import { useWatchPartyVoiceToken } from "./mutations/watchParty";
import { errorMessage } from "../utils/errorMessage";
import { nonFatal } from "../utils/nonFatal";
import type { WatchPartyType } from "../types/api";

export type SessionMediaStatus = "idle" | "connecting" | "connected";

export const SHARE_SCREEN_FAILED = "Screen share could not start. Try again, or share without audio.";

function isPickerDismissal(thrown: unknown): boolean {
    if (!(thrown instanceof Error) || thrown.name !== "NotAllowedError") {
        return false;
    }

    return !thrown.message.includes("by system");
}

export type { ScreenShareMode };

interface UseSessionMediaArgs {
    roomId: string;
    sessionId: string;
    type: WatchPartyType;
    isStarter: boolean;
}

export function useSessionMedia({ roomId, sessionId, type, isStarter }: UseSessionMediaArgs) {
    const [room, setRoom] = useState<Room | null>(null);
    const [status, setStatus] = useState<SessionMediaStatus>("idle");
    const [inVoice, setInVoice] = useState(false);
    const [isSharing, setIsSharing] = useState(false);
    const [shareError, setShareError] = useState<string | null>(null);
    const roomRef = useRef<Room | null>(null);
    const connectingRef = useRef(false);
    const wantMicRef = useRef(false);
    const abortRef = useRef<AbortController | null>(null);

    const { mutateAsync: requestSessionToken } = useWatchPartyVoiceToken(roomId);

    const ensureConnected = useCallback(async (): Promise<Room | null> => {
        if (roomRef.current) {
            return roomRef.current;
        }
        if (connectingRef.current) {
            return null;
        }

        connectingRef.current = true;

        const controller = new AbortController();
        abortRef.current = controller;

        try {
            const { token, url } = await requestSessionToken(sessionId);

            const connected = await connectRoom({
                url,
                token,
                signal: controller.signal,
                on: {
                    onConnected: joined => {
                        setRoom(joined);
                        setStatus("connected");
                    },
                    onDisconnected: () => {
                        roomRef.current = null;
                        setRoom(null);
                        setStatus("idle");
                        setInVoice(false);
                        setIsSharing(false);
                    },
                    onLocalPublicationsChanged: joined => {
                        setIsSharing(joined.localParticipant.isScreenShareEnabled);
                    },
                    onMediaDevicesError: (_joined, error) => {
                        reportClientError(error, { source: "caught" });
                        setShareError(errorMessage(error, SHARE_SCREEN_FAILED));
                    },
                    onLocalPermissionsChanged: joined => {
                        if (!wantMicRef.current || joined.localParticipant.isMicrophoneEnabled) {
                            return;
                        }

                        joined.localParticipant
                            .setMicrophoneEnabled(true)
                            .then(() => setInVoice(true))
                            .catch(nonFatal);
                    },
                },
            });

            roomRef.current = connected;
            return connected;
        } catch (thrown: unknown) {
            reportClientError(thrown, { source: "caught" });

            roomRef.current = null;
            return null;
        } finally {
            connectingRef.current = false;
        }
    }, [requestSessionToken, sessionId]);

    useEffect(() => {
        if (type === "screenshare") {
            ensureConnected().catch(nonFatal);
        }

        return () => {
            abortRef.current?.abort();
            abortRef.current = null;
            disconnectRoom(roomRef.current);
            roomRef.current = null;
        };
    }, [type, ensureConnected]);

    const joinVoice = useCallback(async () => {
        wantMicRef.current = true;
        setStatus("connecting");

        const lkRoom = await ensureConnected();
        if (!lkRoom) {
            setStatus("idle");
            return;
        }

        setStatus("connected");

        try {
            await lkRoom.localParticipant.setMicrophoneEnabled(true);
            setInVoice(true);
        } catch {
            setInVoice(false);
        }
    }, [ensureConnected]);

    const leaveVoice = useCallback(async () => {
        wantMicRef.current = false;

        const lkRoom = roomRef.current;
        if (!lkRoom) {
            return;
        }

        await lkRoom.localParticipant.setMicrophoneEnabled(false);
        setInVoice(false);

        if (type !== "screenshare") {
            disconnectRoom(lkRoom);
        }
    }, [type]);

    const shareScreen = useCallback(
        async (on: boolean, mode: ScreenShareMode) => {
            if (!isStarter) {
                return;
            }

            setShareError(null);

            const lkRoom = await ensureConnected();
            if (!lkRoom) {
                setShareError(SHARE_SCREEN_FAILED);
                return;
            }

            try {
                await setScreenShareEnabled(lkRoom, on, mode);
            } catch (thrown: unknown) {
                if (!isPickerDismissal(thrown)) {
                    reportClientError(thrown, { source: "caught" });
                    setShareError(errorMessage(thrown, SHARE_SCREEN_FAILED));
                }
            }

            setIsSharing(lkRoom.localParticipant.isScreenShareEnabled);
        },
        [ensureConnected, isStarter],
    );

    const reload = useCallback(async () => {
        const existing = roomRef.current;
        if (existing) {
            roomRef.current = null;
            await existing.disconnect();
        }

        const lkRoom = await ensureConnected();
        if (lkRoom && wantMicRef.current) {
            lkRoom.localParticipant
                .setMicrophoneEnabled(true)
                .then(() => setInVoice(true))
                .catch(nonFatal);
        }
    }, [ensureConnected]);

    return { room, status, inVoice, isSharing, shareError, joinVoice, leaveVoice, shareScreen, reload };
}
