import type { GameRoomPlayer } from "../../types/api";
import styles from "./DisconnectBanner.module.css";

interface DisconnectBannerProps {
    offlinePlayer: GameRoomPlayer | undefined;
    forfeitRemaining: number | null;
}

export function DisconnectBanner({ offlinePlayer, forfeitRemaining }: DisconnectBannerProps) {
    if (!offlinePlayer || forfeitRemaining === null) {
        return null;
    }
    return (
        <div className={styles.disconnectBanner}>
            <bdi>{offlinePlayer.display_name}</bdi> disconnected - forfeits in {forfeitRemaining}s
        </div>
    );
}
