import { RoomAudioRenderer, RoomContext, useLocalParticipant } from "@livekit/components-react";
import type { Room } from "livekit-client";

import { VoiceParticipantList } from "./VoiceParticipants";
import styles from "./Voice.module.css";

interface VoiceBarProps {
    room: Room;
    onLeave: () => void;
    canModerate?: boolean;
    onForceMute?: (identity: string, muted: boolean) => void;
}

export function VoiceBar({ room, onLeave, canModerate = false, onForceMute }: VoiceBarProps) {
    return (
        <RoomContext.Provider value={room}>
            <RoomAudioRenderer />
            <VoiceBarInner onLeave={onLeave} canModerate={canModerate} onForceMute={onForceMute} />
        </RoomContext.Provider>
    );
}

function VoiceBarInner({
    onLeave,
    canModerate,
    onForceMute,
}: {
    onLeave: () => void;
    canModerate: boolean;
    onForceMute?: (identity: string, muted: boolean) => void;
}) {
    const { localParticipant, isMicrophoneEnabled } = useLocalParticipant();

    const toggleMute = () => {
        localParticipant.setMicrophoneEnabled(!isMicrophoneEnabled).catch(() => {});
    };

    return (
        <div className={styles.bar}>
            <span className={styles.icon}>{"\u{1F50A}"}</span>

            <VoiceParticipantList canModerate={canModerate} onForceMute={onForceMute} />

            <button type="button" className={styles.control} onClick={toggleMute}>
                {isMicrophoneEnabled ? "Mute" : "Unmute"}
            </button>
            <button type="button" className={`${styles.control} ${styles.leave}`} onClick={onLeave}>
                Leave
            </button>
        </div>
    );
}
