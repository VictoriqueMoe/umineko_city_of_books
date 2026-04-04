import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import type { MysteryAttempt, MysteryDetail } from "../../types/api";
import {
    addMysteryClue,
    createMysteryAttempt,
    deleteMystery,
    deleteMysteryAttempt,
    getMystery,
    markMysterySolved,
    voteMysteryAttempt,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/notifications";
import styles from "./MysteryPages.module.css";

function AttemptItem({
    attempt,
    mysteryId,
    isAuthor,
    onRefresh,
    depth,
}: {
    attempt: MysteryAttempt;
    mysteryId: string;
    isAuthor: boolean;
    onRefresh: () => void;
    depth: number;
}) {
    const { user } = useAuth();
    const [showReply, setShowReply] = useState(false);
    const [replyBody, setReplyBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [voteScore, setVoteScore] = useState(attempt.vote_score);
    const [userVote, setUserVote] = useState(attempt.user_vote ?? 0);

    async function handleVote(value: number) {
        const newValue = userVote === value ? 0 : value;
        const oldScore = voteScore;
        const oldVote = userVote;
        setVoteScore(voteScore - oldVote + newValue);
        setUserVote(newValue);
        try {
            await voteMysteryAttempt(attempt.id, newValue);
        } catch {
            setVoteScore(oldScore);
            setUserVote(oldVote);
        }
    }

    async function handleReply() {
        if (!replyBody.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        try {
            await createMysteryAttempt(mysteryId, replyBody.trim(), attempt.id);
            setReplyBody("");
            setShowReply(false);
            onRefresh();
        } catch {
            // ignore
        } finally {
            setSubmitting(false);
        }
    }

    async function handleDelete() {
        if (!window.confirm("Delete this attempt?")) {
            return;
        }
        await deleteMysteryAttempt(attempt.id);
        onRefresh();
    }

    async function handleSelectWinner() {
        if (!window.confirm(`Select ${attempt.author.display_name} as the winner?`)) {
            return;
        }
        await markMysterySolved(mysteryId, attempt.author.id);
        onRefresh();
    }

    const isOwner = user?.id === attempt.author.id;
    const canDelete = isOwner || can(user?.role, "delete_any_comment");

    return (
        <div>
            <div className={`${styles.attempt}${attempt.is_winner ? ` ${styles.attemptWinner}` : ""}`}>
                <div className={styles.attemptHeader}>
                    <ProfileLink user={attempt.author} size="small" />
                    <span>{relativeTime(attempt.created_at)}</span>
                    {attempt.is_winner && <span className={styles.winnerBadge}>Winner</span>}
                </div>
                <div className={styles.attemptBody}>{attempt.body}</div>
                <div className={styles.attemptActions}>
                    {user && (
                        <>
                            <Button variant="ghost" size="small" onClick={() => handleVote(1)}>
                                {userVote === 1 ? "\u25B2" : "\u25B3"} {voteScore > 0 ? voteScore : ""}
                            </Button>
                            <Button variant="ghost" size="small" onClick={() => handleVote(-1)}>
                                {userVote === -1 ? "\u25BC" : "\u25BD"}
                            </Button>
                            {depth < 3 && (
                                <Button variant="ghost" size="small" onClick={() => setShowReply(!showReply)}>
                                    Reply
                                </Button>
                            )}
                            {isAuthor && !attempt.is_winner && (
                                <Button variant="ghost" size="small" onClick={handleSelectWinner}>
                                    Select Winner
                                </Button>
                            )}
                        </>
                    )}
                    {canDelete && (
                        <Button variant="ghost" size="small" onClick={handleDelete}>
                            Delete
                        </Button>
                    )}
                </div>
                {showReply && (
                    <div className={styles.composer}>
                        <textarea
                            className={styles.composerTextarea}
                            placeholder="Reply..."
                            value={replyBody}
                            onChange={e => setReplyBody(e.target.value)}
                            rows={2}
                        />
                        <div className={styles.composerActions}>
                            <Button variant="ghost" size="small" onClick={() => setShowReply(false)}>
                                Cancel
                            </Button>
                            <Button variant="primary" size="small" onClick={handleReply} disabled={!replyBody.trim() || submitting}>
                                {submitting ? "..." : "Reply"}
                            </Button>
                        </div>
                    </div>
                )}
            </div>
            {attempt.replies && attempt.replies.length > 0 && (
                <div className={styles.replies}>
                    {attempt.replies.map(r => (
                        <AttemptItem
                            key={r.id}
                            attempt={r}
                            mysteryId={mysteryId}
                            isAuthor={isAuthor}
                            onRefresh={onRefresh}
                            depth={depth + 1}
                        />
                    ))}
                </div>
            )}
        </div>
    );
}

export function MysteryDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const [mystery, setMystery] = useState<MysteryDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const [spoilerRevealed, setSpoilerRevealed] = useState(false);
    const [attemptBody, setAttemptBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [newClueBody, setNewClueBody] = useState("");
    const [addingClue, setAddingClue] = useState(false);

    const fetchMystery = useCallback(() => {
        if (!id) {
            return;
        }
        getMystery(id)
            .then(setMystery)
            .catch(() => setMystery(null))
            .finally(() => setLoading(false));
    }, [id]);

    useEffect(() => {
        fetchMystery();
    }, [fetchMystery]);

    if (loading) {
        return <div className="loading">Investigating the mystery...</div>;
    }

    if (!mystery) {
        return <div className="empty-state">Mystery not found.</div>;
    }

    const isAuthor = user?.id === mystery.author.id;
    const canDelete = isAuthor || can(user?.role, "delete_any_theory");

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
                <div className={styles.solvedBanner}>
                    Mystery solved! Winner: {mystery.winner.display_name}
                </div>
            )}

            <div className={styles.detail}>
                <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
                    <div>
                        <h1 className={styles.detailTitle}>{mystery.title}</h1>
                        <div className={styles.detailMeta}>
                            <ProfileLink user={mystery.author} size="small" />
                            <span>{relativeTime(mystery.created_at)}</span>
                            <span className={`${styles.badge} ${styles.badgeDifficulty}`}>{mystery.difficulty}</span>
                            <span className={`${styles.badge} ${mystery.solved ? styles.badgeSolved : styles.badgeOpen}`}>
                                {mystery.solved ? "Solved" : "Open"}
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

                {mystery.clues.length > 0 && (
                    <div className={styles.cluesSection}>
                        <h3 className={styles.cluesTitle}>Red Truths</h3>
                        {mystery.clues.map(clue => (
                            <div
                                key={clue.id}
                                className={`${styles.clue}${clue.truth_type === "purple" ? ` ${styles.cluePurple}` : ""}`}
                            >
                                {clue.body}
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
                            <Button variant="primary" size="small" onClick={handleAddClue} disabled={!newClueBody.trim() || addingClue}>
                                {addingClue ? "..." : "Add Red Truth"}
                            </Button>
                        </div>
                    </div>
                )}
            </div>

            {!spoilerRevealed && !isAuthor ? (
                <div className={styles.spoilerGate}>
                    <h3 className={styles.spoilerGateTitle}>Want to try solving this mystery?</h3>
                    <p className={styles.spoilerGateText}>
                        The attempts below may contain spoilers. Read the mystery and clues above first, then reveal when ready.
                    </p>
                    <Button variant="primary" onClick={() => setSpoilerRevealed(true)}>
                        Reveal Attempts ({mystery.attempts.length})
                    </Button>
                </div>
            ) : (
                <div className={styles.attemptsSection}>
                    <h3 className={styles.attemptsTitle}>
                        Blue Truth Attempts ({mystery.attempts.length})
                    </h3>

                    {mystery.attempts.map(a => (
                        <AttemptItem
                            key={a.id}
                            attempt={a}
                            mysteryId={mystery.id}
                            isAuthor={isAuthor}
                            onRefresh={fetchMystery}
                            depth={0}
                        />
                    ))}

                    {mystery.attempts.length === 0 && (
                        <div className="empty-state">No attempts yet. Be the first to declare your blue truth!</div>
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
            )}
        </div>
    );
}
