import { useEffect, useState } from "react";
import { Link } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import * as api from "../../api/endpoints";
import type { GameRoom } from "../../types/api";
import { Button } from "../../components/Button/Button";
import styles from "./GamesPages.module.css";

const PAGE_SIZE = 20;

export function PastGamesPage() {
    usePageTitle("Past Games");
    const [rooms, setRooms] = useState<GameRoom[]>([]);
    const [total, setTotal] = useState(0);
    const [offset, setOffset] = useState(0);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    useEffect(() => {
        let cancelled = false;
        api.listFinishedGameRooms(undefined, PAGE_SIZE, offset)
            .then(resp => {
                if (cancelled) {
                    return;
                }
                setRooms(resp.rooms ?? []);
                setTotal(resp.total ?? 0);
                setLoading(false);
            })
            .catch(err => {
                if (cancelled) {
                    return;
                }
                setError(err instanceof Error ? err.message : "Failed to load past games");
                setLoading(false);
            });
        return () => {
            cancelled = true;
        };
    }, [offset]);

    function goPrev() {
        setLoading(true);
        setOffset(Math.max(0, offset - PAGE_SIZE));
    }

    function goNext() {
        setLoading(true);
        setOffset(offset + PAGE_SIZE);
    }

    const hasPrev = offset > 0;
    const hasNext = offset + PAGE_SIZE < total;

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>Past Games</h2>
            <p>Every finished match, newest first. Click into any game to see the final board, move list, and stats.</p>
            {error && <div className={styles.error}>{error}</div>}
            {loading ? (
                <p className={styles.empty}>Loading...</p>
            ) : rooms.length === 0 ? (
                <p className={styles.empty}>No finished games yet.</p>
            ) : (
                <>
                    <div className={styles.gameList}>
                        {rooms.map(r => {
                            const white = r.players.find(p => p.slot === 0);
                            const black = r.players.find(p => p.slot === 1);
                            let outcome: string;
                            if (!r.winner_user_id) {
                                outcome = "Draw";
                            } else if (r.winner_user_id === white?.user_id) {
                                outcome = `${white?.display_name ?? "White"} won`;
                            } else {
                                outcome = `${black?.display_name ?? "Black"} won`;
                            }
                            const when = r.finished_at ?? r.updated_at;
                            return (
                                <Link key={r.id} to={`/games/${r.game_type}/${r.id}`} className={styles.gameRow}>
                                    <div className={styles.gameRowContent}>
                                        <span className={styles.opponentLine}>
                                            {white?.display_name ?? "?"} vs {black?.display_name ?? "?"}
                                        </span>
                                        <span className={styles.subline}>
                                            {r.game_type}, {outcome}, {new Date(when).toLocaleString()}
                                        </span>
                                    </div>
                                    <span className={`${styles.statusBadge} ${styles.statusFinished}`}>{r.status}</span>
                                </Link>
                            );
                        })}
                    </div>
                    <div className={styles.pager}>
                        <Button disabled={!hasPrev} onClick={goPrev}>
                            Previous
                        </Button>
                        <span className={styles.pagerInfo}>
                            {offset + 1}-{Math.min(offset + PAGE_SIZE, total)} of {total}
                        </span>
                        <Button disabled={!hasNext} onClick={goNext}>
                            Next
                        </Button>
                    </div>
                </>
            )}
        </div>
    );
}
