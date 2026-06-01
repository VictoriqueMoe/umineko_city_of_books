import { useCallback, useEffect, useRef, useState } from "react";
import { Room, RoomEvent } from "livekit-client";

import { getVoiceToken } from "../../../api/endpoints";
import { useNotifications } from "../../../hooks/useNotifications";
import { playVoiceJoinSound, playVoiceLeaveSound } from "../../../utils/sound";
import type { WSMessage } from "../../../types/api";

export type VoiceStatus = "idle" | "connecting" | "connected";

interface VoicePresenceData {
    room_id: string;
    participants: string[];
    count: number;
}

export function useVoiceChat(roomId: string, initialParticipants: string[] = []) {
    const { addWSListener } = useNotifications();
    const [status, setStatus] = useState<VoiceStatus>("idle");
    const [room, setRoom] = useState<Room | null>(null);
    const [wsPresence, setWsPresence] = useState<{ roomId: string; ids: string[] } | null>(null);
    const roomRef = useRef<Room | null>(null);

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type !== "voice_presence") {
                return;
            }

            const data = msg.data as VoicePresenceData;
            if (data.room_id !== roomId) {
                return;
            }

            setWsPresence({ roomId, ids: data.participants ?? [] });
        });
    }, [addWSListener, roomId]);

    const participantIds = wsPresence && wsPresence.roomId === roomId ? wsPresence.ids : initialParticipants;

    const leave = useCallback(() => {
        const current = roomRef.current;
        roomRef.current = null;
        setRoom(null);
        setStatus("idle");

        if (current) {
            playVoiceLeaveSound();
            current.disconnect().catch(() => {});
        }
    }, []);

    const join = useCallback(() => {
        if (roomRef.current) {
            return;
        }

        setStatus("connecting");

        const connect = async () => {
            const { token, url } = await getVoiceToken(roomId);

            const livekitRoom = new Room();
            roomRef.current = livekitRoom;

            livekitRoom.on(RoomEvent.Disconnected, () => {
                roomRef.current = null;
                setRoom(null);
                setStatus("idle");
            });

            livekitRoom.on(RoomEvent.ParticipantConnected, () => {
                playVoiceJoinSound();
            });

            livekitRoom.on(RoomEvent.ParticipantDisconnected, () => {
                playVoiceLeaveSound();
            });

            await livekitRoom.connect(url, token);

            setRoom(livekitRoom);
            setStatus("connected");
            playVoiceJoinSound();

            livekitRoom.localParticipant.setMicrophoneEnabled(true).catch(() => {});
        };

        connect().catch(() => {
            roomRef.current = null;
            setStatus("idle");
        });
    }, [roomId]);

    useEffect(() => {
        return () => {
            if (roomRef.current) {
                roomRef.current.disconnect().catch(() => {});
                roomRef.current = null;
            }
        };
    }, []);

    return { status, room, participantIds, presenceCount: participantIds.length, join, leave };
}
