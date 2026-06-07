import { useEffect } from "react";
import { Room, RoomEvent } from "livekit-client";

export function useAudioPlaybackGuard(room: Room | null) {
    useEffect(() => {
        if (!room) {
            return;
        }

        const recoverBlockedPlayback = () => {
            if (!room.canPlaybackAudio) {
                room.startAudio().catch(() => {});
            }
        };

        const resumePausedTrack = (e: Event) => {
            const el = e.target;
            if (!(el instanceof HTMLAudioElement)) {
                return;
            }

            const stream = el.srcObject;
            if (!(stream instanceof MediaStream)) {
                return;
            }

            const hasLiveAudio = stream.getAudioTracks().some(t => t.readyState === "live");
            if (!hasLiveAudio) {
                return;
            }

            el.play().catch(() => {});
        };

        room.on(RoomEvent.AudioPlaybackStatusChanged, recoverBlockedPlayback);
        document.addEventListener("pause", resumePausedTrack, true);

        return () => {
            room.off(RoomEvent.AudioPlaybackStatusChanged, recoverBlockedPlayback);
            document.removeEventListener("pause", resumePausedTrack, true);
        };
    }, [room]);
}
