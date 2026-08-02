import { useEffect, useState } from "react";
import { useIsSpeaking, useParticipants } from "@livekit/components-react";
import { RemoteParticipant } from "livekit-client";
import type { Participant } from "livekit-client";

import styles from "./Voice.module.css";

interface VoiceParticipantListProps {
    canModerate?: boolean;
    onForceMute?: (identity: string, muted: boolean) => void;
}

function applyVolume(participant: Participant, silenced: boolean) {
    if (participant instanceof RemoteParticipant) {
        participant.setVolume(silenced ? 0 : 1);
    }
}

export function VoiceParticipantList({ canModerate = false, onForceMute }: VoiceParticipantListProps) {
    const participants = useParticipants();
    const [mutedIds, setMutedIds] = useState<Set<string>>(new Set());
    const [forceMutedIds, setForceMutedIds] = useState<Set<string>>(new Set());
    const [deafened, setDeafened] = useState(false);

    useEffect(() => {
        for (const p of participants) {
            if (p.isLocal) {
                continue;
            }
            applyVolume(p, deafened || mutedIds.has(p.identity));
        }
    }, [participants, deafened, mutedIds]);

    const toggleForceMute = (identity: string) => {
        const next = new Set(forceMutedIds);
        const muted = !next.has(identity);
        if (muted) {
            next.add(identity);
        } else {
            next.delete(identity);
        }
        setForceMutedIds(next);
        onForceMute?.(identity, muted);
    };

    const toggleLocalMute = (participant: Participant) => {
        const next = new Set(mutedIds);
        if (next.has(participant.identity)) {
            next.delete(participant.identity);
        } else {
            next.add(participant.identity);
        }
        setMutedIds(next);
    };

    const toggleDeafen = () => {
        setDeafened(!deafened);
    };

    return (
        <div className={styles.participants}>
            <button
                type="button"
                className={`${styles.control} ${deafened ? styles.controlActive : ""}`}
                onClick={toggleDeafen}
                title={deafened ? "Unmute everyone for yourself" : "Mute everyone for yourself"}
            >
                {deafened ? "Unmute all" : "Mute all"}
            </button>
            {participants.map(p => (
                <VoiceParticipant
                    key={p.identity}
                    participant={p}
                    locallyMuted={deafened || mutedIds.has(p.identity)}
                    forceMuted={forceMutedIds.has(p.identity)}
                    canModerate={canModerate}
                    onToggleLocalMute={() => toggleLocalMute(p)}
                    onToggleForceMute={onForceMute ? () => toggleForceMute(p.identity) : undefined}
                />
            ))}
        </div>
    );
}

interface VoiceParticipantProps {
    participant: Participant;
    locallyMuted: boolean;
    forceMuted: boolean;
    canModerate: boolean;
    onToggleLocalMute: () => void;
    onToggleForceMute?: () => void;
}

function VoiceParticipant({
    participant,
    locallyMuted,
    forceMuted,
    canModerate,
    onToggleLocalMute,
    onToggleForceMute,
}: VoiceParticipantProps) {
    const isSpeaking = useIsSpeaking(participant);
    const name = participant.name || participant.identity;
    const isLocal = participant.isLocal;

    return (
        <span className={`${styles.participant} ${isSpeaking ? styles.speaking : ""}`} title={name}>
            <span className={styles.dot} />
            <span className={styles.name}>{name}</span>
            {!isLocal && (
                <button
                    type="button"
                    className={styles.miniBtn}
                    onClick={onToggleLocalMute}
                    title={locallyMuted ? "Muted just for you, click to hear them" : "Mute them just for you"}
                >
                    {locallyMuted ? "\u{1F507}" : "\u{1F50A}"}
                </button>
            )}
            {!isLocal && canModerate && onToggleForceMute && (
                <button
                    type="button"
                    className={`${styles.modMuteBtn} ${forceMuted ? styles.modMuteBtnActive : ""}`}
                    onClick={onToggleForceMute}
                    title={forceMuted ? "Unmute for everyone" : "Mute for everyone"}
                >
                    {forceMuted ? "Unmute" : "Mute"}
                </button>
            )}
        </span>
    );
}
