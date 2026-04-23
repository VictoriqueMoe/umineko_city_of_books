import type { GameRoom } from "../../types/api";
import { formatDuration } from "./gameRoomHelpers";
import styles from "./GamePlayerBar.module.css";

interface GamePlayerBarProps {
    room: GameRoom;
    slot0Label: string;
    slot1Label: string;
    liveDurationSeconds: number;
}

export function GamePlayerBar({ room, slot0Label, slot1Label, liveDurationSeconds }: GamePlayerBarProps) {
    const slot0 = room.players.find(p => p.slot === 0);
    const slot1 = room.players.find(p => p.slot === 1);
    const isActive = room.status === "active";

    return (
        <div className={styles.status}>
            <div className={styles.statusLeft}>
                <span className={`${styles.playerDot} ${slot1?.connected ? styles.playerDotOn : ""}`} />
                <span className={styles.playerName}>{slot1?.display_name ?? slot1Label}</span>
                <span className={styles.colourLabel}>({slot1Label})</span>
                <span
                    className={`${styles.turnMarker} ${
                        room.turn_user_id === slot1?.user_id && isActive ? styles.turnMarkerActive : ""
                    }`}
                >
                    {room.turn_user_id === slot1?.user_id && isActive ? "to move" : ""}
                </span>
            </div>
            <div className={styles.statusCenter}>
                <span className={styles.watcherCount} title="Spectators watching">
                    👁 {room.watcher_count}
                </span>
                <span className={styles.watcherCount} title="Game duration">
                    ⏱ {formatDuration(liveDurationSeconds)}
                </span>
            </div>
            <div className={styles.statusRight}>
                <span
                    className={`${styles.turnMarker} ${
                        room.turn_user_id === slot0?.user_id && isActive ? styles.turnMarkerActive : ""
                    }`}
                >
                    {room.turn_user_id === slot0?.user_id && isActive ? "to move" : ""}
                </span>
                <span className={styles.colourLabel}>({slot0Label})</span>
                <span className={styles.playerName}>{slot0?.display_name ?? slot0Label}</span>
                <span className={`${styles.playerDot} ${slot0?.connected ? styles.playerDotOn : ""}`} />
            </div>
        </div>
    );
}
