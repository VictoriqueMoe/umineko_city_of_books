import { useCallback, useEffect, useRef, useState } from "react";
import { Room, RoomEvent } from "livekit-client";

import { getWatchPartyVoiceToken } from "../../../api/endpoints";
import type { WatchPartyType } from "../../../types/api";

export type SessionMediaStatus = "idle" | "connecting" | "connected";

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
    const roomRef = useRef<Room | null>(null);
    const connectingRef = useRef(false);
    const wantMicRef = useRef(false);

    const buildRoom = useCallback(() => {
        const lkRoom = new Room();
        roomRef.current = lkRoom;

        lkRoom.on(RoomEvent.Connected, () => {
            setRoom(lkRoom);
            setStatus("connected");
        });

        lkRoom.on(RoomEvent.Disconnected, () => {
            roomRef.current = null;
            setRoom(null);
            setStatus("idle");
            setInVoice(false);
            setIsSharing(false);
        });

        lkRoom.on(RoomEvent.ParticipantPermissionsChanged, (_prev, participant) => {
            if (participant !== lkRoom.localParticipant || !wantMicRef.current) {
                return;
            }
            if (lkRoom.localParticipant.isMicrophoneEnabled) {
                return;
            }
            lkRoom.localParticipant
                .setMicrophoneEnabled(true)
                .then(() => setInVoice(true))
                .catch(() => {});
        });

        return lkRoom;
    }, []);

    const ensureConnected = useCallback(async (): Promise<Room | null> => {
        if (roomRef.current) {
            return roomRef.current;
        }
        if (connectingRef.current) {
            return null;
        }

        connectingRef.current = true;

        try {
            const { token, url } = await getWatchPartyVoiceToken(roomId, sessionId);

            const lkRoom = buildRoom();
            await lkRoom.connect(url, token);
            return lkRoom;
        } catch {
            roomRef.current = null;
            return null;
        } finally {
            connectingRef.current = false;
        }
    }, [roomId, sessionId, buildRoom]);

    useEffect(() => {
        if (type === "screenshare") {
            ensureConnected().catch(() => {});
        }

        return () => {
            if (roomRef.current) {
                roomRef.current.disconnect().catch(() => {});
                roomRef.current = null;
            }
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
            lkRoom.disconnect().catch(() => {});
        }
    }, [type]);

    const shareScreen = useCallback(
        async (on: boolean) => {
            if (!isStarter) {
                return;
            }

            const lkRoom = await ensureConnected();
            if (!lkRoom) {
                return;
            }

            await lkRoom.localParticipant.setScreenShareEnabled(on, { audio: true });
            setIsSharing(on);
        },
        [ensureConnected, isStarter],
    );

    return { room, status, inVoice, isSharing, joinVoice, leaveVoice, shareScreen };
}
