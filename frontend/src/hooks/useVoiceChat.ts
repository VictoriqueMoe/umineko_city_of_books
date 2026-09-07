import { useCallback, useEffect, useRef, useState } from "react";
import type { Room } from "livekit-client";

import { connectRoom, disconnectRoom } from "../api/livekit/connect";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { reportClientError } from "../api/telemetry";
import { useVoiceToken } from "./mutations/voice";
import { playVoiceJoinSound, playVoiceLeaveSound } from "../platform/sound";
import { errorMessage } from "../utils/errorMessage";
import { nonFatal } from "../utils/nonFatal";

export type VoiceStatus = "idle" | "connecting" | "connected";

export const VOICE_JOIN_FAILED = "Could not join the voice call.";

export interface VoiceChatState {
    status: VoiceStatus;
    room: Room | null;
    participantIds: string[];
    presenceCount: number;
    error: string | null;
    join: () => void;
    leave: () => void;
    clearError: () => void;
}

export function useVoiceChat(roomId: string, initialParticipants: string[] = []): VoiceChatState {
    const [status, setStatus] = useState<VoiceStatus>("idle");
    const [room, setRoom] = useState<Room | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [wsPresence, setWsPresence] = useState<{ roomId: string; ids: string[] } | null>(null);
    const roomRef = useRef<Room | null>(null);
    const joiningRef = useRef(false);
    const abortRef = useRef<AbortController | null>(null);

    const { mutateAsync: requestVoiceToken } = useVoiceToken();

    useRealtimeEvent(REALTIME_EVENTS.VOICE_PRESENCE, event => {
        if (event.data.room_id !== roomId) {
            return;
        }

        setWsPresence({ roomId, ids: event.data.participants ?? [] });
    });

    const participantIds = wsPresence && wsPresence.roomId === roomId ? wsPresence.ids : initialParticipants;

    const leave = useCallback(() => {
        const current = roomRef.current;
        roomRef.current = null;
        setRoom(null);
        setStatus("idle");

        if (current) {
            playVoiceLeaveSound();
            disconnectRoom(current);
        }
    }, []);

    const join = useCallback(() => {
        if (roomRef.current || joiningRef.current) {
            return;
        }

        joiningRef.current = true;
        setError(null);
        setStatus("connecting");

        const controller = new AbortController();
        abortRef.current = controller;

        const connect = async () => {
            const { token, url } = await requestVoiceToken(roomId);

            const livekitRoom = await connectRoom({
                url,
                token,
                signal: controller.signal,
                on: {
                    onDisconnected: () => {
                        roomRef.current = null;
                        setRoom(null);
                        setStatus("idle");
                    },
                    onParticipantConnected: () => playVoiceJoinSound(),
                    onParticipantDisconnected: () => playVoiceLeaveSound(),
                },
            });

            if (!livekitRoom) {
                return;
            }

            roomRef.current = livekitRoom;
            setRoom(livekitRoom);
            setStatus("connected");
            playVoiceJoinSound();

            livekitRoom.localParticipant.setMicrophoneEnabled(true).catch(nonFatal);
        };

        connect()
            .catch((thrown: unknown) => {
                reportClientError(thrown, { source: "caught" });

                roomRef.current = null;
                setStatus("idle");
                setError(errorMessage(thrown, VOICE_JOIN_FAILED));
            })
            .finally(() => {
                joiningRef.current = false;
            });
    }, [requestVoiceToken, roomId]);

    const clearError = useCallback(() => setError(null), []);

    useEffect(() => {
        return () => {
            abortRef.current?.abort();
            abortRef.current = null;
            joiningRef.current = false;
            disconnectRoom(roomRef.current);
            roomRef.current = null;
            setRoom(null);
            setStatus("idle");
            setError(null);
            setWsPresence(null);
        };
    }, [roomId]);

    return {
        status,
        room,
        participantIds,
        presenceCount: participantIds.length,
        error,
        join,
        leave,
        clearError,
    };
}
