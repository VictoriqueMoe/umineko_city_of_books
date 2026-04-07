import { type MouseEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import type { MysteryAttempt, MysteryClue, MysteryDetail, PostComment } from "../../types/api";
import {
    addMysteryClue,
    createMysteryAttempt,
    createMysteryComment,
    deleteMystery,
    deleteMysteryComment,
    getMystery,
    likeMysteryComment,
    unlikeMysteryComment,
    updateMysteryComment,
    uploadMysteryCommentMedia,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { useThrottled } from "../../hooks/useThrottled";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/notifications";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { AttemptItem } from "./AttemptItem";
import styles from "./MysteryPages.module.css";

function ClueCopyBtn({ text }: { text: string }) {
    const [copied, setCopied] = useState(false);

    function handleCopy(e: MouseEvent) {
        e.stopPropagation();
        navigator.clipboard.writeText(text).then(() => {
            setCopied(true);
            setTimeout(() => setCopied(false), 1500);
        });
    }

    return (
        <button type="button" className={styles.clueCopy} onClick={handleCopy} title="Copy to clipboard">
            {copied ? (
                "\u2713"
            ) : (
                <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                >
                    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                    <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                </svg>
            )}
        </button>
    );
}

function PrivateClues({
    clues,
    playerId,
    mysteryId,
    isAuthor,
    onAdded,
}: {
    clues: MysteryClue[];
    playerId: string;
    mysteryId: string;
    isAuthor: boolean;
    onAdded: () => void;
}) {
    const [body, setBody] = useState("");
    const [adding, setAdding] = useState(false);
    const playerClues = clues.filter(c => c.player_id === playerId);

    async function handleAdd() {
        if (!body.trim() || adding) {
            return;
        }
        setAdding(true);
        try {
            await addMysteryClue(mysteryId, body.trim(), "red", playerId);
            setBody("");
            onAdded();
        } catch {
            // ignore
        } finally {
            setAdding(false);
        }
    }

    return (
        <div style={{ padding: "0 0.5rem", marginBottom: "0.5rem" }}>
            {playerClues.length > 0 && (
                <div className={styles.cluesSection} style={{ marginBottom: "0.5rem" }}>
                    {playerClues.map(clue => (
                        <div key={clue.id} className={styles.clue} style={{ fontSize: "0.85rem" }}>
                            {clue.body}
                            <ClueCopyBtn text={clue.body} />
                        </div>
                    ))}
                </div>
            )}
            {isAuthor && (
                <div style={{ display: "flex", gap: "0.4rem", alignItems: "center" }}>
                    <input
                        type="text"
                        value={body}
                        onChange={e => setBody(e.target.value)}
                        placeholder="Private red truth for this player..."
                        onKeyDown={e => {
                            if (e.key === "Enter") {
                                handleAdd();
                            }
                        }}
                        style={{
                            flex: 1,
                            background: "var(--bg-void)",
                            border: "1px solid rgba(229, 57, 53, 0.3)",
                            color: "#ef9a9a",
                            padding: "0.35rem 0.6rem",
                            borderRadius: "4px",
                            fontSize: "0.8rem",
                            fontFamily: "inherit",
                            fontStyle: "italic",
                        }}
                    />
                    <Button variant="danger" size="small" onClick={handleAdd} disabled={!body.trim() || adding}>
                        {adding ? "..." : "Add private Red Truth"}
                    </Button>
                </div>
            )}
        </div>
    );
}

function findWinningAttempt(attempts: MysteryAttempt[]): MysteryAttempt | null {
    for (const a of attempts) {
        if (a.is_winner) {
            return a;
        }
        if (a.replies && a.replies.length > 0) {
            const nested = findWinningAttempt(a.replies);
            if (nested) {
                return nested;
            }
        }
    }
    return null;
}

export function MysteryDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const [mystery, setMystery] = useState<MysteryDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const hash = location.hash;
    const highlightedAttempt = hash.startsWith("#attempt-") ? hash.replace("#attempt-", "") : null;
    const [attemptBody, setAttemptBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [collapsedPlayers, setCollapsedPlayers] = useState<Set<string>>(new Set());
    const [unreadPlayers, setUnreadPlayers] = useState<Set<string>>(new Set());
    const initialUnreadComputedFor = useRef<string | null>(null);
    const [newClueBody, setNewClueBody] = useState("");
    const [addingClue, setAddingClue] = useState(false);

    function togglePlayerCollapse(authorId: string) {
        setCollapsedPlayers(prev => {
            const next = new Set(prev);
            if (next.has(authorId)) {
                next.delete(authorId);
            } else {
                next.add(authorId);
            }
            return next;
        });
    }

    function markPlayerRead(authorId: string) {
        setUnreadPlayers(prev => {
            if (!prev.has(authorId)) {
                return prev;
            }
            const next = new Set(prev);
            next.delete(authorId);
            return next;
        });
    }

    function jumpToPlayer(authorId: string) {
        markPlayerRead(authorId);
        setCollapsedPlayers(prev => {
            if (!prev.has(authorId)) {
                return prev;
            }
            const next = new Set(prev);
            next.delete(authorId);
            return next;
        });
        requestAnimationFrame(() => {
            const el = document.getElementById(`player-group-${authorId}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "start" });
            }
        });
    }

    const winningAttempt = useMemo(
        () => (mystery?.solved ? findWinningAttempt(mystery.attempts ?? []) : null),
        [mystery],
    );

    const groupedAttempts = useMemo(() => {
        if (!mystery) {
            return [];
        }
        const groups = new Map<string, { author: MysteryAttempt["author"]; attempts: MysteryAttempt[] }>();
        for (const a of mystery.attempts ?? []) {
            const existing = groups.get(a.author.id);
            if (existing) {
                existing.attempts.push(a);
            } else {
                groups.set(a.author.id, { author: a.author, attempts: [a] });
            }
        }
        return Array.from(groups.values());
    }, [mystery]);

    const fetchMystery = useCallback(() => {
        if (!id) {
            return;
        }
        getMystery(id)
            .then(setMystery)
            .catch(() => setMystery(null))
            .finally(() => setLoading(false));
    }, [id]);

    const throttledFetchMystery = useThrottled(fetchMystery, 200);

    useEffect(() => {
        fetchMystery();
    }, [fetchMystery]);

    useEffect(() => {
        if (!id) {
            return;
        }
        return addWSListener(msg => {
            if (msg.type === "mystery_solved") {
                const data = msg.data as { mystery_id?: string; attempt_id?: string };
                if (data.mystery_id !== id) {
                    return;
                }
                throttledFetchMystery();
                if (data.attempt_id) {
                    requestAnimationFrame(() => {
                        const el = document.getElementById(`attempt-${data.attempt_id}`);
                        if (el) {
                            el.scrollIntoView({ behavior: "smooth", block: "center" });
                        }
                    });
                }
                return;
            }
            if (msg.type === "mystery_attempt_created") {
                const data = msg.data as { mystery_id?: string; author_id?: string };
                if (data.mystery_id !== id) {
                    return;
                }
                if (data.author_id && data.author_id !== user?.id) {
                    setUnreadPlayers(prev => {
                        const next = new Set(prev);
                        next.add(data.author_id as string);
                        return next;
                    });
                }
                throttledFetchMystery();
            }
            if (msg.type === "mystery_clue_added") {
                const data = msg.data as { mystery_id?: string };
                if (data.mystery_id === id) {
                    throttledFetchMystery();
                }
            }
        });
    }, [id, addWSListener, throttledFetchMystery, user?.id]);

    useEffect(() => {
        if (!mystery || !id) {
            return;
        }
        if (initialUnreadComputedFor.current === id) {
            return;
        }
        initialUnreadComputedFor.current = id;

        const isGM = user?.id === mystery.author.id || user?.role === "super_admin";
        if (!isGM || mystery.solved) {
            return;
        }
        const cursorRaw = localStorage.getItem(`mystery-read-cursor-${id}`);
        if (!cursorRaw) {
            localStorage.setItem(`mystery-read-cursor-${id}`, new Date().toISOString());
            return;
        }
        const cursor = new Date(cursorRaw).getTime();
        const unread = new Set<string>();
        for (const a of mystery.attempts ?? []) {
            const created = new Date(a.created_at).getTime();
            if (created > cursor && a.author.id !== user?.id) {
                unread.add(a.author.id);
            }
        }
        if (unread.size > 0) {
            setUnreadPlayers(unread);
        }
    }, [mystery, id, user?.id, user?.role]);

    useEffect(() => {
        if (!id) {
            return;
        }
        return () => {
            localStorage.setItem(`mystery-read-cursor-${id}`, new Date().toISOString());
        };
    }, [id]);

    useEffect(() => {
        if (!mystery || loading || !highlightedAttempt) {
            return;
        }
        requestAnimationFrame(() => {
            const el = document.getElementById(`attempt-${highlightedAttempt}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
            }
        });
    }, [mystery, loading, highlightedAttempt]);

    if (loading) {
        return <div className="loading">Investigating the mystery...</div>;
    }

    if (!mystery) {
        return <div className="empty-state">Mystery not found.</div>;
    }

    const isAuthor = user?.id === mystery.author.id;
    const canDelete = isAuthor || can(user?.role, "delete_any_theory");
    const canSeeAsGameMaster = isAuthor || user?.role === "super_admin";

    async function handleSubmitAttempt() {
        if (!attemptBody.trim() || submitting || !id) {
            return;
        }
        setSubmitting(true);
        try {
            await createMysteryAttempt(id, attemptBody.trim());
            setAttemptBody("");
            fetchMystery();
        } catch {
            // ignore
        } finally {
            setSubmitting(false);
        }
    }

    async function handleAddClue() {
        if (!newClueBody.trim() || addingClue || !id) {
            return;
        }
        setAddingClue(true);
        try {
            await addMysteryClue(id, newClueBody.trim(), "red");
            setNewClueBody("");
            fetchMystery();
        } catch {
            // ignore
        } finally {
            setAddingClue(false);
        }
    }

    async function handleDelete() {
        if (!window.confirm("Delete this mystery? This cannot be undone.")) {
            return;
        }
        await deleteMystery(mystery!.id);
        navigate("/mysteries");
    }

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/mysteries")}>
                &larr; All Mysteries
            </span>

            {mystery.solved && mystery.winner && (
                <div className={styles.solvedBanner}>Mystery solved! Winner: {mystery.winner.display_name}</div>
            )}

            <div className={styles.detail}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                    <div>
                        <h1 className={styles.detailTitle}>{mystery.title}</h1>
                        <div className={styles.detailMeta}>
                            <ProfileLink user={mystery.author} size="small" />
                            <span>{relativeTime(mystery.created_at)}</span>
                        </div>
                        <div className={styles.cardBadges}>
                            <span className={`${styles.badge} ${styles.badgeDifficulty}`}>{mystery.difficulty}</span>
                            <span
                                className={`${styles.badge} ${mystery.solved ? styles.badgeSolved : styles.badgeOpen}`}
                            >
                                {mystery.solved ? "Solved" : "Open"}
                            </span>
                            <span className={`${styles.badge} ${styles.badgePieces}`}>
                                {mystery.player_count} piece{mystery.player_count !== 1 ? "s" : ""} attempting
                            </span>
                        </div>
                    </div>
                    {canDelete && (
                        <Button variant="danger" size="small" onClick={handleDelete}>
                            Delete
                        </Button>
                    )}
                </div>

                <div className={styles.detailBody}>{mystery.body}</div>

                {mystery.clues.filter(c => !c.player_id).length > 0 && (
                    <div className={styles.cluesSection}>
                        <h3 className={styles.cluesTitle}>Red Truths</h3>
                        {mystery.clues
                            .filter(c => !c.player_id)
                            .map(clue => (
                                <div
                                    key={clue.id}
                                    className={`${styles.clue}${clue.truth_type === "purple" ? ` ${styles.cluePurple}` : ""}`}
                                >
                                    {clue.body}
                                    <ClueCopyBtn text={clue.body} />
                                </div>
                            ))}
                    </div>
                )}

                {isAuthor && (
                    <div className={styles.composer}>
                        <textarea
                            className={styles.composerTextarea}
                            placeholder="Add a new red truth clue..."
                            value={newClueBody}
                            onChange={e => setNewClueBody(e.target.value)}
                            rows={2}
                        />
                        <div className={styles.composerActions}>
                            <Button
                                variant="danger"
                                size="small"
                                onClick={handleAddClue}
                                disabled={!newClueBody.trim() || addingClue}
                            >
                                {addingClue ? "..." : "Add global Red Truth"}
                            </Button>
                        </div>
                    </div>
                )}
            </div>

            <div className={styles.attemptsSection}>
                <h3 className={styles.attemptsTitle}>Blue Truth Attempts ({mystery.attempts.length})</h3>

                {canSeeAsGameMaster && !mystery.solved && groupedAttempts.length > 0 && (
                    <div className={styles.playerPills}>
                        {groupedAttempts.map(group => {
                            const isUnread = unreadPlayers.has(group.author.id);
                            return (
                                <button
                                    key={group.author.id}
                                    type="button"
                                    className={`${styles.playerPill}${isUnread ? ` ${styles.playerPillUnread}` : ""}`}
                                    onClick={() => jumpToPlayer(group.author.id)}
                                    title={`Jump to ${group.author.display_name}'s attempts`}
                                >
                                    {group.author.avatar_url ? (
                                        <img className={styles.playerPillAvatar} src={group.author.avatar_url} alt="" />
                                    ) : (
                                        <span className={styles.playerPillAvatarPlaceholder}>
                                            {group.author.display_name[0]}
                                        </span>
                                    )}
                                    <span className={styles.playerPillName}>{group.author.display_name}</span>
                                    {isUnread && <span className={styles.playerPillDot} aria-label="unread" />}
                                </button>
                            );
                        })}
                    </div>
                )}

                {winningAttempt && (
                    <div className={styles.pinnedWinner}>
                        <div className={styles.pinnedWinnerHeader}>
                            <span className={styles.pinnedWinnerLabel}>Winning Attempt</span>
                            <a
                                className={styles.pinnedWinnerJump}
                                href={`#attempt-${winningAttempt.id}`}
                                onClick={e => {
                                    e.preventDefault();
                                    const el = document.getElementById(`attempt-${winningAttempt.id}`);
                                    if (el) {
                                        el.scrollIntoView({ behavior: "smooth", block: "center" });
                                        window.history.replaceState(null, "", `#attempt-${winningAttempt.id}`);
                                    }
                                }}
                            >
                                Jump to original &rarr;
                            </a>
                        </div>
                        <div className={styles.pinnedWinnerMeta}>
                            <ProfileLink user={winningAttempt.author} size="small" />
                            <span>{relativeTime(winningAttempt.created_at)}</span>
                        </div>
                        <div className={styles.pinnedWinnerBody}>{winningAttempt.body}</div>
                    </div>
                )}

                {canSeeAsGameMaster || mystery.solved ? (
                    groupedAttempts.map(group => {
                        const collapsed = collapsedPlayers.has(group.author.id);
                        return (
                            <div
                                key={group.author.id}
                                id={`player-group-${group.author.id}`}
                                className={styles.playerGroup}
                            >
                                <button
                                    type="button"
                                    className={styles.playerGroupHeader}
                                    onClick={() => togglePlayerCollapse(group.author.id)}
                                    aria-expanded={!collapsed}
                                >
                                    <span className={styles.playerGroupChevron}>{collapsed ? "\u25B6" : "\u25BC"}</span>
                                    <ProfileLink user={group.author} size="small" />
                                    <span className={styles.playerGroupCount}>
                                        {group.attempts.length} attempt
                                        {group.attempts.length !== 1 ? "s" : ""}
                                    </span>
                                </button>
                                {!collapsed && (
                                    <>
                                        <PrivateClues
                                            clues={mystery.clues}
                                            playerId={group.author.id}
                                            mysteryId={mystery.id}
                                            isAuthor={isAuthor}
                                            onAdded={fetchMystery}
                                        />
                                        {group.attempts.map(a => (
                                            <AttemptItem
                                                key={a.id}
                                                attempt={a}
                                                mysteryId={mystery.id}
                                                isAuthor={isAuthor}
                                                onRefresh={fetchMystery}
                                                mysterySolved={mystery.solved}
                                            />
                                        ))}
                                    </>
                                )}
                            </div>
                        );
                    })
                ) : (
                    <>
                        {user && mystery.clues.filter(c => c.player_id === user.id).length > 0 && (
                            <div className={styles.cluesSection}>
                                <h3 className={styles.cluesTitle} style={{ fontSize: "0.85rem" }}>
                                    Private Red Truths (to you)
                                </h3>
                                {mystery.clues
                                    .filter(c => c.player_id === user.id)
                                    .map(clue => (
                                        <div key={clue.id} className={styles.clue}>
                                            {clue.body}
                                            <ClueCopyBtn text={clue.body} />
                                        </div>
                                    ))}
                            </div>
                        )}
                        {mystery.attempts.map(a => (
                            <AttemptItem
                                key={a.id}
                                attempt={a}
                                mysteryId={mystery.id}
                                isAuthor={isAuthor}
                                onRefresh={fetchMystery}
                                mysterySolved={mystery.solved}
                            />
                        ))}
                    </>
                )}

                {mystery.attempts.length === 0 && (
                    <div className="empty-state">
                        {!canSeeAsGameMaster && mystery.player_count > 0
                            ? `There ${mystery.player_count === 1 ? "is" : "are"} ${mystery.player_count} piece${mystery.player_count !== 1 ? "s" : ""} playing this mystery. Join the game board and declare your own blue truth!`
                            : canSeeAsGameMaster
                              ? "No attempts yet. Waiting for pieces to make their move."
                              : "No attempts yet. Be the first to declare your blue truth!"}
                    </div>
                )}

                {user && !isAuthor && !mystery.solved && (
                    <div className={styles.composer}>
                        <textarea
                            className={styles.composerTextarea}
                            placeholder="Declare your blue truth..."
                            value={attemptBody}
                            onChange={e => setAttemptBody(e.target.value)}
                            rows={3}
                        />
                        <div className={styles.composerActions}>
                            <Button
                                variant="primary"
                                onClick={handleSubmitAttempt}
                                disabled={!attemptBody.trim() || submitting}
                            >
                                {submitting ? "..." : "Submit Blue Truth"}
                            </Button>
                        </div>
                    </div>
                )}

                {!user && (
                    <div className="empty-state">
                        <Button variant="primary" onClick={() => navigate("/login")}>
                            Sign in to attempt
                        </Button>
                    </div>
                )}
            </div>

            {mystery.solved && (
                <div className={styles.discussionSection}>
                    <h3 className={styles.attemptsTitle}>
                        Post-Game Discussion {mystery.comments.length > 0 && `(${mystery.comments.length})`}
                    </h3>
                    {mystery.comments.map(c => (
                        <CommentItem
                            key={c.id}
                            comment={c as unknown as PostComment}
                            postId={mystery.id}
                            onDelete={fetchMystery}
                            highlighted={false}
                            linkPrefix="/mystery"
                            reportType="mystery_comment"
                            likeFn={likeMysteryComment}
                            unlikeFn={unlikeMysteryComment}
                            deleteFn={deleteMysteryComment}
                            updateFn={updateMysteryComment}
                            createCommentFn={createMysteryComment}
                            uploadMediaFn={uploadMysteryCommentMedia}
                        />
                    ))}
                    {mystery.comments.length === 0 && (
                        <p className="empty-state">The mystery is solved. Share your thoughts on the game!</p>
                    )}
                    {user && id && (
                        <CommentComposer
                            postId={id}
                            onCreated={fetchMystery}
                            createCommentFn={createMysteryComment}
                            uploadMediaFn={uploadMysteryCommentMedia}
                        />
                    )}
                </div>
            )}
        </div>
    );
}
