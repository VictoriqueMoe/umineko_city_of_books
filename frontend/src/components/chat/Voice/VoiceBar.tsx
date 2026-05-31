import {
    RoomAudioRenderer,
    RoomContext,
    useIsSpeaking,
    useLocalParticipant,
    useParticipants,
} from "@livekit/components-react";
import type { Participant, Room } from "livekit-client";

import styles from "./Voice.module.css";

interface VoiceBarProps {
    room: Room;
    onLeave: () => void;
}

export function VoiceBar({ room, onLeave }: VoiceBarProps) {
    return (
        <RoomContext.Provider value={room}>
            <RoomAudioRenderer />
            <VoiceBarInner onLeave={onLeave} />
        </RoomContext.Provider>
    );
}

function VoiceBarInner({ onLeave }: { onLeave: () => void }) {
    const participants = useParticipants();
    const { localParticipant, isMicrophoneEnabled } = useLocalParticipant();

    const toggleMute = () => {
        localParticipant.setMicrophoneEnabled(!isMicrophoneEnabled).catch(() => {});
    };

    return (
        <div className={styles.bar}>
            <span className={styles.icon}>{"\u{1F50A}"}</span>

            <div className={styles.participants}>
                {participants.map(p => (
                    <VoiceParticipant key={p.identity} participant={p} />
                ))}
            </div>

            <button type="button" className={styles.control} onClick={toggleMute}>
                {isMicrophoneEnabled ? "Mute" : "Unmute"}
            </button>
            <button type="button" className={`${styles.control} ${styles.leave}`} onClick={onLeave}>
                Leave
            </button>
        </div>
    );
}

function VoiceParticipant({ participant }: { participant: Participant }) {
    const isSpeaking = useIsSpeaking(participant);
    const name = participant.name || participant.identity;

    return (
        <span className={`${styles.participant} ${isSpeaking ? styles.speaking : ""}`} title={name}>
            <span className={styles.dot} />
            {name}
        </span>
    );
}
