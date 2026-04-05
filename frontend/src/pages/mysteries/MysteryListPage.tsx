import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import type { Mystery, MysteryLeaderboardEntry } from "../../types/api";
import { getMysteryLeaderboard, listMysteries } from "../../api/endpoints";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { relativeTime } from "../../utils/notifications";
import styles from "./MysteryPages.module.css";

export function MysteryListPage() {
    const navigate = useNavigate();
    const [mysteries, setMysteries] = useState<Mystery[]>([]);
    const [total, setTotal] = useState(0);
    const [offset, setOffset] = useState(0);
    const [sort, setSort] = useState("new");
    const [solved, setSolved] = useState("");
    const [loading, setLoading] = useState(true);
    const [leaderboard, setLeaderboard] = useState<MysteryLeaderboardEntry[]>([]);
    const limit = 20;

    useEffect(() => {
        getMysteryLeaderboard(10)
            .then(res => setLeaderboard(res.entries ?? []))
            .catch(() => setLeaderboard([]));
    }, []);

    useEffect(() => {
        let cancelled = false;
        listMysteries({ sort, solved: solved || undefined, limit, offset })
            .then(data => {
                if (!cancelled) {
                    setMysteries(data.mysteries ?? []);
                    setTotal(data.total);
                    setLoading(false);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setMysteries([]);
                    setLoading(false);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [sort, solved, offset]);

    return (
        <div className={styles.page}>
            <h1 className={styles.heading}>Mysteries</h1>

            <div className={styles.layout}>
                <div className={styles.main}>
                    <InfoPanel title="Welcome, Piece">
                        <p>
                            A Game Master has laid out a mystery for you to solve. Read the scenario, study the red
                            truths carefully, they are absolute and cannot be denied. Then declare your blue truth: your
                            theory on the solution. The Game Master will respond, either dismantling your theory or
                            acknowledging your deduction. The first piece to solve the mystery is declared the winner.
                        </p>
                    </InfoPanel>

                    <RulesBox page="mysteries" />

                    <div className={styles.controls}>
                        <Select
                            value={sort}
                            onChange={e => {
                                setSort(e.target.value);
                                setOffset(0);
                            }}
                        >
                            <option value="new">Newest</option>
                            <option value="old">Oldest</option>
                        </Select>
                        <Select
                            value={solved}
                            onChange={e => {
                                setSolved(e.target.value);
                                setOffset(0);
                            }}
                        >
                            <option value="">All</option>
                            <option value="false">Unsolved</option>
                            <option value="true">Solved</option>
                        </Select>
                    </div>

                    {loading && <div className="loading">Loading mysteries...</div>}

                    {!loading && mysteries.length === 0 && (
                        <div className="empty-state">
                            No mysteries yet. Be the first game master to challenge the board.
                        </div>
                    )}

                    {!loading && (
                        <div className={styles.list}>
                            {mysteries.map(m => (
                                <div
                                    key={m.id}
                                    className={`${styles.card}${m.solved ? ` ${styles.cardSolved}` : ""}`}
                                    onClick={() => navigate(`/mystery/${m.id}`)}
                                >
                                    <div className={styles.cardHeader}>
                                        <span className={styles.cardTitle}>{m.title}</span>
                                        <span
                                            className={`${styles.badge} ${m.solved ? styles.badgeSolved : styles.badgeOpen}`}
                                        >
                                            {m.solved ? "Solved" : "Open"}
                                        </span>
                                        <span className={`${styles.badge} ${styles.badgeDifficulty}`}>
                                            {m.difficulty}
                                        </span>
                                    </div>
                                    <div className={styles.cardMeta}>
                                        <ProfileLink user={m.author} size="small" />
                                        <span>{relativeTime(m.created_at)}</span>
                                    </div>
                                    <div className={styles.cardStats}>
                                        <span>
                                            {m.clue_count} clue{m.clue_count !== 1 ? "s" : ""}
                                        </span>
                                        <span>
                                            {m.attempt_count} attempt{m.attempt_count !== 1 ? "s" : ""}
                                        </span>
                                        {m.winner && <span>Winner: {m.winner.display_name}</span>}
                                    </div>
                                    <p className={styles.cardPreview}>
                                        {m.body.length > 200 ? m.body.slice(0, 200) + "..." : m.body}
                                    </p>
                                </div>
                            ))}
                        </div>
                    )}

                    <Pagination
                        offset={offset}
                        limit={limit}
                        total={total}
                        hasNext={offset + limit < total}
                        hasPrev={offset > 0}
                        onNext={() => setOffset(offset + limit)}
                        onPrev={() => setOffset(Math.max(0, offset - limit))}
                    />
                </div>

                <aside className={styles.sidebar}>
                    <div className={styles.leaderboard}>
                        <h3 className={styles.leaderboardTitle}>Top Detectives</h3>
                        {leaderboard.length === 0 ? (
                            <p className={styles.leaderboardEmpty}>
                                No mysteries have been solved yet. Be the first to claim a winner's laurels.
                            </p>
                        ) : (
                            <ol className={styles.leaderboardList}>
                                {leaderboard.map((entry, i) => (
                                    <li key={entry.user.id} className={styles.leaderboardItem}>
                                        <span className={styles.leaderboardRank}>#{i + 1}</span>
                                        <ProfileLink user={entry.user} size="small" />
                                        <span className={styles.leaderboardScore}>{entry.solved_count} solved</span>
                                    </li>
                                ))}
                            </ol>
                        )}
                    </div>
                </aside>
            </div>
        </div>
    );
}
