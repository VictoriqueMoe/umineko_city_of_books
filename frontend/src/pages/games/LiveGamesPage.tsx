import { useEffect, useState } from "react";
import { Link } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import * as api from "../../api/endpoints";
import type { GameRoom } from "../../types/api";
import styles from "./GamesPages.module.css";

export function LiveGamesPage() {
    usePageTitle("Live Games");
    const [rooms, setRooms] = useState<GameRoom[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    useEffect(() => {
        let cancelled = false;
        api.listLiveGameRooms()
            .then(resp => {
                if (cancelled) {
                    return;
                }
                setRooms(resp.rooms ?? []);
                setLoading(false);
            })
            .catch(err => {
                if (cancelled) {
                    return;
                }
                setError(err instanceof Error ? err.message : "Failed to load live games");
                setLoading(false);
            });
        return () => {
            cancelled = true;
        };
    }, []);

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>Live Games</h2>
            <p>Matches currently in progress. Click to spectate and chat with other viewers.</p>
            {error && <div className={styles.error}>{error}</div>}
            {loading ? (
                <p className={styles.empty}>Loading...</p>
            ) : rooms.length === 0 ? (
                <p className={styles.empty}>No games in progress right now.</p>
            ) : (
                <div className={styles.gameList}>
                    {rooms.map(r => {
                        const white = r.players.find(p => p.slot === 0);
                        const black = r.players.find(p => p.slot === 1);
                        return (
                            <Link key={r.id} to={`/games/${r.game_type}/${r.id}`} className={styles.gameRow}>
                                <div className={styles.gameRowContent}>
                                    <span className={styles.opponentLine}>
                                        {white?.display_name ?? "?"} vs {black?.display_name ?? "?"}
                                    </span>
                                    <span className={styles.subline}>
                                        {r.game_type} — {r.watcher_count} watching
                                    </span>
                                </div>
                                <span className={`${styles.statusBadge} ${styles.statusActive}`}>live</span>
                            </Link>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
