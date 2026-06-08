import { useCallback, useEffect, useRef, useState } from "react";
import { AudioPresets, Room, RoomEvent } from "livekit-client";

import { getWatchPartyVoiceToken } from "../../../api/endpoints";
import type { WatchPartyType } from "../../../types/api";

export type SessionMediaStatus = "idle" | "connecting" | "connected";

export type ScreenShareMode = "gaming" | "screenshare";

interface ScreenSharePreset {
    contentHint: "motion" | "detail";
    resolution: { width: number; height: number; frameRate: number };
    videoCodec: "vp9";
    degradationPreference: "maintain-framerate" | "maintain-resolution";
    maxBitrate: number;
}

const SCREEN_SHARE_PRESETS: Record<ScreenShareMode, ScreenSharePreset> = {
    gaming: {
        contentHint: "motion",
        resolution: { width: 1920, height: 1080, frameRate: 60 },
        videoCodec: "vp9",
        degradationPreference: "maintain-framerate",
        maxBitrate: 6_000_000,
    },
    screenshare: {
        contentHint: "detail",
        resolution: { width: 1920, height: 1080, frameRate: 15 },
        videoCodec: "vp9",
        degradationPreference: "maintain-resolution",
        maxBitrate: 2_500_000,
    },
};

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
        } catch (err) {
            console.error("[watchparty] LiveKit connect failed", err);
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
        async (on: boolean, mode: ScreenShareMode) => {
            if (!isStarter) {
                return;
            }

            const lkRoom = await ensureConnected();
            if (!lkRoom) {
                return;
            }

            const preset = SCREEN_SHARE_PRESETS[mode];

            await lkRoom.localParticipant.setScreenShareEnabled(
                on,
                {
                    audio: {
                        echoCancellation: false,
                        noiseSuppression: false,
                        autoGainControl: false,
                    },
                    contentHint: preset.contentHint,
                    resolution: preset.resolution,
                },
                {
                    audioPreset: AudioPresets.musicHighQualityStereo,
                    dtx: false,
                    red: false,
                    forceStereo: true,
                    videoCodec: preset.videoCodec,
                    degradationPreference: preset.degradationPreference,
                    screenShareEncoding: {
                        maxBitrate: preset.maxBitrate,
                        maxFramerate: preset.resolution.frameRate,
                    },
                },
            );
            setIsSharing(on);
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
                .catch(() => {});
        }
    }, [ensureConnected]);

    return { room, status, inVoice, isSharing, joinVoice, leaveVoice, shareScreen, reload };
}
