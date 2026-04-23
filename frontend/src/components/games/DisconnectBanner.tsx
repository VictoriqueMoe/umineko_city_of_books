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
            {offlinePlayer.display_name} disconnected - forfeits in {forfeitRemaining}s
        </div>
    );
}
