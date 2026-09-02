import { type ReactNode, useCallback, useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { useAuth } from "../../hooks/useAuth";
import type { User } from "../../types/api";
import { useGMLeaderboard, useMysteryLeaderboard, useMysteryList } from "../../hooks/queries/mystery";
import { timerColour } from "../../domain/mystery";
import { formatDuration, parseServerDate } from "../../utils/time";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RoleStyledName } from "../../components/RoleStyledName/RoleStyledName";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import { RulesBox } from "../../components/RulesBox/RulesBox";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import { PieceTrigger } from "../../components/easterEgg";
import { MysteryBadges } from "./MysteryBadges";
import { ellipsise } from "../../utils/text";
import styles from "./MysteryPages.module.css";

interface LeaderboardEntry {
    user: User;
    score: number;
}

function LiveTimer({
    since,
    until,
    pausedAt,
    pausedDurationSeconds,
}: {
    since: string;
    until?: string | null;
    pausedAt?: string | null;
    pausedDurationSeconds?: number;
}) {
    const [elapsed, setElapsed] = useState(0);
    const end = parseServerDate(until)?.getTime() ?? null;
    const sinceMs = parseServerDate(since)?.getTime() ?? 0;
    const pausedAtMs = parseServerDate(pausedAt)?.getTime() ?? null;
    const storedPausedMs = (pausedDurationSeconds ?? 0) * 1000;
    const isStopped = end !== null;

    useEffect(() => {
        function tick() {
            const target = isStopped && end !== null ? end : Date.now();
            const raw = target - sinceMs;
            const activePausedMs = pausedAtMs !== null ? Math.max(0, target - pausedAtMs) : 0;
            setElapsed(Math.max(0, raw - storedPausedMs - activePausedMs));
        }
        tick();
        if (isStopped || pausedAtMs !== null) {
            return;
        }
        const id = setInterval(tick, 1000);
        return () => clearInterval(id);
    }, [sinceMs, end, isStopped, pausedAtMs, storedPausedMs]);

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

function Leaderboard<TEntry extends LeaderboardEntry>({
    entries,
    info,
    emptyText,
    topLabel,
    topLabelTitle,
    topLabelClassName,
    expandedId,
    onToggle,
    renderBreakdown,
}: {
    entries: TEntry[];
    info: string;
    emptyText: string;
    topLabel: string;
    topLabelTitle: string;
    topLabelClassName: string;
    expandedId: string | null;
    onToggle: (id: string) => void;
    renderBreakdown: (entry: TEntry) => ReactNode;
}) {
    return (
        <>
            <p className={styles.leaderboardInfo}>{info}</p>
            {entries.length === 0 ? (
                <p className={styles.leaderboardEmpty}>{emptyText}</p>
            ) : (
                <ol className={styles.leaderboardList}>
                    {entries.map((entry, i) => {
                        const isExpanded = expandedId === entry.user.id;
                        return (
                            <li key={entry.user.id}>
                                <div
                                    className={`${styles.leaderboardItem}${isExpanded ? ` ${styles.leaderboardItemExpanded}` : ""}`}
                                    onClick={() => onToggle(entry.user.id)}
                                >
                                    <span className={styles.leaderboardRank}>#{i + 1}</span>
                                    <LeaderboardAvatar user={entry.user} />
                                    <span className={styles.leaderboardName}>
                                        <RoleStyledName name={entry.user.display_name} role={entry.user.role} />
                                    </span>
                                    <span className={styles.leaderboardScore}>{entry.score} pts</span>
                                </div>
                                {entry.score === entries[0].score && (
                                    <div className={styles.topDetectiveRow}>
                                        <span dir="auto" className={topLabelClassName} title={topLabelTitle}>
                                            {topLabel}
                                        </span>
                                    </div>
                                )}
                                {isExpanded && (
                                    <div className={styles.leaderboardBreakdown}>{renderBreakdown(entry)}</div>
                                )}
                            </li>
                        );
                    })}
                </ol>
            )}
        </>
    );
}

export function MysteryListPage() {
    usePageTitle("Mysteries");
    const { user } = useAuth();
    const siteInfo = useSiteInfo();
    const detectiveRole = siteInfo.vanity_roles?.find(v => v.id === "system_top_detective");
    const gmRole = siteInfo.vanity_roles?.find(v => v.id === "system_top_gm");
    const [searchParams, setSearchParams] = useSearchParams();
    const [offset, setOffset] = useState(0);
    const [sort, setSort] = useState(searchParams.get("sort") || "new");
    const [solved, setSolved] = useState(searchParams.get("solved") ?? "false");
    const [expandedId, setExpandedId] = useState<string | null>(null);
    const [gmExpandedId, setGMExpandedId] = useState<string | null>(null);
    const [leaderboardTab, setLeaderboardTab] = useState<"detectives" | "gm">("detectives");
    const limit = 20;
    const { mysteries, total, loading } = useMysteryList({ sort, solved: solved || undefined, limit, offset });
    const { entries: leaderboard } = useMysteryLeaderboard(10);
    const { entries: gmLeaderboard } = useGMLeaderboard(10);

    const toggleExpand = useCallback((id: string) => {
        setExpandedId(prev => (prev === id ? null : id));
    }, []);

    const toggleGMExpand = useCallback((id: string) => {
        setGMExpandedId(prev => (prev === id ? null : id));
    }, []);

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
                        {user && (
                            <Link to="/mystery/new">
                                <Button variant="primary" size="small">
                                    + New Mystery
                                </Button>
                            </Link>
                        )}
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
                                    <div dir="auto" className={styles.cardTitle}>
                                        {m.title}
                                    </div>
                                    <div className={styles.cardMeta}>
                                        <ProfileLink user={m.author} size="small" clickable={false} />
                                        <RelativeTimestamp value={m.created_at} />
                                    </div>
                                    <MysteryBadges mystery={m} />
                                    <div className={styles.cardStats}>
                                        <span>
                                            {m.clue_count} clue{m.clue_count !== 1 ? "s" : ""}
                                        </span>
                                        <span>
                                            {m.attempt_count} attempt{m.attempt_count !== 1 ? "s" : ""}
                                        </span>
                                    </div>
                                    <div className={styles.cardTimer}>
                                        {m.winner && (
                                            <span>
                                                Winner: <bdi>{m.winner.display_name}</bdi>
                                            </span>
                                        )}
                                        <span style={{ color: timerColour(m.created_at, m.solved) }}>
                                            {m.solved ? "Solved in " : m.paused ? "Paused at " : "Unsolved for "}
                                            <LiveTimer
                                                since={m.created_at}
                                                until={m.solved_at}
                                                pausedAt={m.paused_at}
                                                pausedDurationSeconds={m.paused_duration_seconds}
                                            />
                                        </span>
                                    </div>
                                    <p dir="auto" className={styles.cardPreview}>
                                        {ellipsise(m.body, 200)}
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
                        <div className={styles.leaderboardTabs}>
                            <button
                                className={`${styles.leaderboardTab}${leaderboardTab === "detectives" ? ` ${styles.leaderboardTabActive}` : ""}`}
                                onClick={() => setLeaderboardTab("detectives")}
                            >
                                Top Detectives <PieceTrigger pieceId="piece_04" />
                            </button>
                            <button
                                className={`${styles.leaderboardTab}${leaderboardTab === "gm" ? ` ${styles.leaderboardTabActive}` : ""}`}
                                onClick={() => setLeaderboardTab("gm")}
                            >
                                Top Game Masters
                            </button>
                        </div>

                        {leaderboardTab === "detectives" && (
                            <Leaderboard
                                entries={leaderboard}
                                info="Scores are based on difficulty: Easy 2 pts, Medium 4 pts, Hard 6 pts, Nightmare 8 pts. Click a detective to see their breakdown."
                                emptyText="No mysteries have been solved yet. Be the first to claim a winner's laurels."
                                topLabel={detectiveRole?.label ?? "True Detective"}
                                topLabelTitle="Ranked #1 in mysteries"
                                topLabelClassName={styles.topDetectiveBadge}
                                expandedId={expandedId}
                                onToggle={toggleExpand}
                                renderBreakdown={entry => (
                                    <>
                                        <span className={styles.breakdownTotal}>
                                            {entry.easy_solved +
                                                entry.medium_solved +
                                                entry.hard_solved +
                                                entry.nightmare_solved}{" "}
                                            solved
                                        </span>
                                        {entry.easy_solved > 0 && (
                                            <span className={styles.breakdownRow}>
                                                <span className={styles.breakdownLabel}>Easy</span>
                                                <span className={styles.breakdownCount}>{entry.easy_solved}</span>
                                            </span>
                                        )}
                                        {entry.medium_solved > 0 && (
                                            <span className={styles.breakdownRow}>
                                                <span className={styles.breakdownLabel}>Medium</span>
                                                <span className={styles.breakdownCount}>{entry.medium_solved}</span>
                                            </span>
                                        )}
                                        {entry.hard_solved > 0 && (
                                            <span className={styles.breakdownRow}>
                                                <span className={styles.breakdownLabel}>Hard</span>
                                                <span className={styles.breakdownCount}>{entry.hard_solved}</span>
                                            </span>
                                        )}
                                        {entry.nightmare_solved > 0 && (
                                            <span className={styles.breakdownRow}>
                                                <span className={styles.breakdownLabel}>Nightmare</span>
                                                <span className={styles.breakdownCount}>{entry.nightmare_solved}</span>
                                            </span>
                                        )}
                                        {entry.score_adjustment !== 0 && (
                                            <span className={styles.breakdownRow}>
                                                <span className={styles.breakdownLabel}>Adjusted score</span>
                                                <span className={styles.breakdownCount}>
                                                    {entry.score_adjustment > 0 ? "+" : ""}
                                                    {entry.score_adjustment}
                                                </span>
                                            </span>
                                        )}
                                    </>
                                )}
                            />
                        )}

                        {leaderboardTab === "gm" && (
                            <Leaderboard
                                entries={gmLeaderboard}
                                info="Scores are based on difficulty + player engagement: base points per solved mystery plus up to 5 bonus points for unique players. Click to see breakdown."
                                emptyText="No mysteries have been solved yet. Create a mystery and have it solved to appear here."
                                topLabel={gmRole?.label ?? "Game Master"}
                                topLabelTitle="Top ranked Game Master"
                                topLabelClassName={styles.topGMBadge}
                                expandedId={gmExpandedId}
                                onToggle={toggleGMExpand}
                                renderBreakdown={entry => (
                                    <>
                                        <span className={styles.breakdownTotal}>
                                            {entry.mystery_count} {entry.mystery_count === 1 ? "mystery" : "mysteries"}{" "}
                                            solved
                                        </span>
                                        <span className={styles.breakdownRow}>
                                            <span className={styles.breakdownLabel}>Total players</span>
                                            <span className={styles.breakdownCount}>{entry.player_count}</span>
                                        </span>
                                    </>
                                )}
                            />
                        )}
                    </div>
                </aside>
            </div>
        </div>
    );
}
