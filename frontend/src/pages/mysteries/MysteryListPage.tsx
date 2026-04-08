import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import type { Mystery, MysteryLeaderboardEntry, User } from "../../types/api";
import { getMysteryLeaderboard, listMysteries } from "../../api/endpoints";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RoleStyledName } from "../../components/RoleStyledName/RoleStyledName";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { relativeTime } from "../../utils/notifications";
import styles from "./MysteryPages.module.css";

function formatDuration(ms: number): string {
    const totalSeconds = Math.round(ms / 1000);
    const levels: [number, string][] = [
        [Math.floor(totalSeconds / 31536000), "years"],
        [Math.floor((totalSeconds % 31536000) / 86400), "days"],
        [Math.floor(((totalSeconds % 31536000) % 86400) / 3600), "hours"],
        [Math.floor((((totalSeconds % 31536000) % 86400) % 3600) / 60), "minutes"],
        [Math.floor((((totalSeconds % 31536000) % 86400) % 3600) % 60), "seconds"],
    ];
    let result = "";
    for (const [value, label] of levels) {
        if (value === 0) {
            continue;
        }
        result += ` ${value} ${value === 1 ? label.slice(0, -1) : label}`;
    }
    return result.trim() || "0 seconds";
}

function timerColour(createdAt: string, solved: boolean): string {
    if (solved) {
        return "#66bb6a";
    }
    const days = (Date.now() - new Date(createdAt).getTime()) / 86400000;
    if (days < 1) {
        return "#64b5f6";
    }
    if (days < 7) {
        return "#ffd54f";
    }
    if (days < 30) {
        return "#ffb74d";
    }
    return "#e57373";
}

function LiveTimer({ since, until }: { since: string; until?: string | null }) {
    const [elapsed, setElapsed] = useState(0);
    const end = until ? new Date(until).getTime() : null;
    const sinceMs = new Date(since).getTime();
    const isStopped = end !== null;

    useEffect(() => {
        function tick() {
            const target = isStopped && end !== null ? end : Date.now();
            setElapsed(Math.max(0, target - sinceMs));
        }
        tick();
        if (isStopped) {
            return;
        }
        const id = setInterval(tick, 1000);
        return () => clearInterval(id);
    }, [sinceMs, end, isStopped]);

    return <span>{formatDuration(elapsed)}</span>;
}

function LeaderboardAvatar({ user }: { user: User }) {
    const navigate = useNavigate();
    return (
        <span
            className={styles.leaderboardAvatar}
            onClick={e => {
                e.stopPropagation();
                navigate(`/user/${user.username}`);
            }}
        >
            {user.avatar_url ? (
                <img src={user.avatar_url} alt="" className={styles.leaderboardAvatarImg} />
            ) : (
                <span className={styles.leaderboardAvatarPlaceholder}>{user.display_name[0]}</span>
            )}
        </span>
    );
}

export function MysteryListPage() {
    const [searchParams, setSearchParams] = useSearchParams();
    const [mysteries, setMysteries] = useState<Mystery[]>([]);
    const [total, setTotal] = useState(0);
    const [offset, setOffset] = useState(0);
    const [sort, setSort] = useState(searchParams.get("sort") || "new");
    const [solved, setSolved] = useState(searchParams.get("solved") ?? "false");
    const [loading, setLoading] = useState(true);
    const [leaderboard, setLeaderboard] = useState<MysteryLeaderboardEntry[]>([]);
    const [expandedId, setExpandedId] = useState<string | null>(null);
    const limit = 20;

    const toggleExpand = useCallback((id: string) => {
        setExpandedId(prev => (prev === id ? null : id));
    }, []);

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
                                const v = e.target.value;
                                setSort(v);
                                setOffset(0);
                                setSearchParams(
                                    prev => {
                                        prev.set("sort", v);
                                        return prev;
                                    },
                                    { replace: true },
                                );
                            }}
                        >
                            <option value="new">Newest</option>
                            <option value="old">Oldest</option>
                        </Select>
                        <Select
                            value={solved}
                            onChange={e => {
                                const v = e.target.value;
                                setSolved(v);
                                setOffset(0);
                                setSearchParams(
                                    prev => {
                                        if (v) {
                                            prev.set("solved", v);
                                        } else {
                                            prev.delete("solved");
                                        }
                                        return prev;
                                    },
                                    { replace: true },
                                );
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
                                <Link
                                    key={m.id}
                                    to={`/mystery/${m.id}`}
                                    className={`${styles.card}${m.solved ? ` ${styles.cardSolved}` : ""}`}
                                >
                                    <div className={styles.cardTitle}>{m.title}</div>
                                    <div className={styles.cardMeta}>
                                        <ProfileLink user={m.author} size="small" clickable={false} />
                                        <span>{relativeTime(m.created_at)}</span>
                                    </div>
                                    <div className={styles.cardBadges}>
                                        <span
                                            className={`${styles.badge} ${m.solved ? styles.badgeSolved : styles.badgeOpen}`}
                                        >
                                            {m.solved ? "Solved" : "Open"}
                                        </span>
                                        <span className={`${styles.badge} ${styles.badgeDifficulty}`}>
                                            {m.difficulty}
                                        </span>
                                    </div>
                                    <div className={styles.cardStats}>
                                        <span>
                                            {m.clue_count} clue{m.clue_count !== 1 ? "s" : ""}
                                        </span>
                                        <span>
                                            {m.attempt_count} attempt{m.attempt_count !== 1 ? "s" : ""}
                                        </span>
                                    </div>
                                    <div className={styles.cardTimer}>
                                        {m.winner && <span>Winner: {m.winner.display_name}</span>}
                                        <span style={{ color: timerColour(m.created_at, m.solved) }}>
                                            {m.solved ? "Solved in " : "Unsolved for "}
                                            <LiveTimer since={m.created_at} until={m.solved_at} />
                                        </span>
                                    </div>
                                    <p className={styles.cardPreview}>
                                        {m.body.length > 200 ? m.body.slice(0, 200) + "..." : m.body}
                                    </p>
                                </Link>
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
                        <p className={styles.leaderboardInfo}>
                            Scores are based on difficulty: Easy 2 pts, Medium 4 pts, Hard 6 pts, Nightmare 8 pts. Click
                            a detective to see their breakdown.
                        </p>
                        {leaderboard.length === 0 ? (
                            <p className={styles.leaderboardEmpty}>
                                No mysteries have been solved yet. Be the first to claim a winner's laurels.
                            </p>
                        ) : (
                            <ol className={styles.leaderboardList}>
                                {leaderboard.map((entry, i) => {
                                    const isExpanded = expandedId === entry.user.id;
                                    const total =
                                        entry.easy_solved +
                                        entry.medium_solved +
                                        entry.hard_solved +
                                        entry.nightmare_solved;
                                    return (
                                        <li key={entry.user.id}>
                                            <div
                                                className={`${styles.leaderboardItem}${isExpanded ? ` ${styles.leaderboardItemExpanded}` : ""}`}
                                                onClick={() => toggleExpand(entry.user.id)}
                                            >
                                                <span className={styles.leaderboardRank}>#{i + 1}</span>
                                                <LeaderboardAvatar user={entry.user} />
                                                <span className={styles.leaderboardName}>
                                                    <RoleStyledName
                                                        name={entry.user.display_name}
                                                        role={entry.user.role}
                                                    />
                                                </span>
                                                <span className={styles.leaderboardScore}>{entry.score} pts</span>
                                            </div>
                                            {leaderboard.length > 0 && entry.score === leaderboard[0].score && (
                                                <div className={styles.topDetectiveRow}>
                                                    <span
                                                        className={styles.topDetectiveBadge}
                                                        title="Ranked #1 in mysteries"
                                                    >
                                                        True Detective
                                                    </span>
                                                </div>
                                            )}
                                            {isExpanded && (
                                                <div className={styles.leaderboardBreakdown}>
                                                    <span className={styles.breakdownTotal}>{total} solved</span>
                                                    {entry.easy_solved > 0 && (
                                                        <span className={styles.breakdownRow}>
                                                            <span className={styles.breakdownLabel}>Easy</span>
                                                            <span className={styles.breakdownCount}>
                                                                {entry.easy_solved}
                                                            </span>
                                                        </span>
                                                    )}
                                                    {entry.medium_solved > 0 && (
                                                        <span className={styles.breakdownRow}>
                                                            <span className={styles.breakdownLabel}>Medium</span>
                                                            <span className={styles.breakdownCount}>
                                                                {entry.medium_solved}
                                                            </span>
                                                        </span>
                                                    )}
                                                    {entry.hard_solved > 0 && (
                                                        <span className={styles.breakdownRow}>
                                                            <span className={styles.breakdownLabel}>Hard</span>
                                                            <span className={styles.breakdownCount}>
                                                                {entry.hard_solved}
                                                            </span>
                                                        </span>
                                                    )}
                                                    {entry.nightmare_solved > 0 && (
                                                        <span className={styles.breakdownRow}>
                                                            <span className={styles.breakdownLabel}>Nightmare</span>
                                                            <span className={styles.breakdownCount}>
                                                                {entry.nightmare_solved}
                                                            </span>
                                                        </span>
                                                    )}
                                                    {entry.score_adjustment !== 0 && (
                                                        <span className={styles.breakdownRow}>
                                                            <span className={styles.breakdownLabel}>
                                                                Adjusted score
                                                            </span>
                                                            <span className={styles.breakdownCount}>
                                                                {entry.score_adjustment > 0 ? "+" : ""}
                                                                {entry.score_adjustment}
                                                            </span>
                                                        </span>
                                                    )}
                                                </div>
                                            )}
                                        </li>
                                    );
                                })}
                            </ol>
                        )}
                    </div>
                </aside>
            </div>
        </div>
    );
}
