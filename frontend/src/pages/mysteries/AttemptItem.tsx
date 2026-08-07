import { useCallback, useState } from "react";
import type { MysteryAttempt } from "../../types/api";
import {
    useCreateMysteryAttempt,
    useDeleteMysteryAttempt,
    useMarkMysterySolved,
    useVoteMysteryAttempt,
} from "../../api/mutations/mystery";
import { useAuth } from "../../hooks/useAuth";
import { useVote } from "../../hooks/useVote";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import { ReportButton } from "../../components/ReportButton/ReportButton";
import { siteUrl } from "../../utils/siteOrigin";
import styles from "./MysteryPages.module.css";

function flattenReplies(attempt: MysteryAttempt): { reply: MysteryAttempt; replyToName: string }[] {
    const result: { reply: MysteryAttempt; replyToName: string }[] = [];

    function walk(a: MysteryAttempt, parentName: string) {
        for (const reply of a.replies ?? []) {
            result.push({ reply, replyToName: parentName });
            walk(reply, reply.author.display_name);
        }
    }

    walk(attempt, attempt.author.display_name);
    return result;
}

function SingleAttempt({
    attempt,
    mysteryId,
    isAuthor,
    onRefresh,
    replyToName,
    mysterySolved,
    mysteryPaused,
    authorAlreadyWon,
}: {
    attempt: MysteryAttempt;
    mysteryId: string;
    isAuthor: boolean;
    onRefresh: () => void;
    replyToName?: string;
    mysterySolved: boolean;
    mysteryPaused: boolean;
    authorAlreadyWon: boolean;
}) {
    const { user } = useAuth();
    const [showReply, setShowReply] = useState(false);
    const [replyBody, setReplyBody] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const voteMutation = useVoteMysteryAttempt(mysteryId);
    const createReplyMutation = useCreateMysteryAttempt(mysteryId);
    const deleteAttemptMutation = useDeleteMysteryAttempt(mysteryId);
    const markSolvedMutation = useMarkMysterySolved(mysteryId);

    const voteFn = useCallback(
        async (value: number) => {
            await voteMutation.mutateAsync({ id: attempt.id, value });
        },
        [attempt.id, voteMutation],
    );

    const {
        score: voteScore,
        userVote,
        vote: handleVote,
    } = useVote(attempt.vote_score, attempt.user_vote ?? 0, voteFn);

    async function handleReply() {
        if (!replyBody.trim() || submitting) {
            return;
        }
        setSubmitting(true);
        try {
            await createReplyMutation.mutateAsync({ body: replyBody.trim(), parentId: attempt.id });
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
        await deleteAttemptMutation.mutateAsync(attempt.id);
        onRefresh();
    }

    async function handleSelectWinner() {
        if (!window.confirm(`Select this attempt by ${attempt.author.display_name} as the winner?`)) {
            return;
        }
        await markSolvedMutation.mutateAsync(attempt.id);
        onRefresh();
    }

    const isOwner = user?.id === attempt.author.id;
    const canDelete = isOwner || can(user, "delete_any_comment");

    return (
        <div
            id={`attempt-${attempt.id}`}
            className={`${styles.attempt}${attempt.is_winner ? ` ${styles.attemptWinner}` : ""}`}
        >
            <div className={styles.attemptHeader}>
                <ProfileLink user={attempt.author} size="small" />
                {replyToName && <span className={styles.replyTo}>@{replyToName}</span>}
                <RelativeTimestamp value={attempt.created_at} />
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
                        {(isAuthor || isOwner) && !mysterySolved && (!mysteryPaused || isAuthor) && (
                            <Button variant="ghost" size="small" onClick={() => setShowReply(!showReply)}>
                                Reply
                            </Button>
                        )}
                        {isAuthor && !mysterySolved && user?.id !== attempt.author.id && !authorAlreadyWon && (
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
                <Button
                    variant="ghost"
                    size="small"
                    onClick={() =>
                        navigator.clipboard.writeText(siteUrl(`/mystery/${mysteryId}#attempt-${attempt.id}`))
                    }
                >
                    Copy Link
                </Button>
                {user && !isOwner && (
                    <ReportButton targetType="mystery_attempt" targetId={attempt.id} contextId={mysteryId} />
                )}
            </div>
            {showReply && (!mysteryPaused || isAuthor) && (
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
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleReply}
                            disabled={!replyBody.trim() || submitting}
                        >
                            {submitting ? "..." : "Reply"}
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}

export function AttemptItem({
    attempt,
    mysteryId,
    isAuthor,
    onRefresh,
    mysterySolved,
    mysteryPaused,
    authorAlreadyWon,
}: {
    attempt: MysteryAttempt;
    mysteryId: string;
    isAuthor: boolean;
    onRefresh: () => void;
    mysterySolved: boolean;
    mysteryPaused: boolean;
    authorAlreadyWon: boolean;
}) {
    const allReplies = flattenReplies(attempt);
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div>
            <SingleAttempt
                attempt={attempt}
                mysteryId={mysteryId}
                isAuthor={isAuthor}
                onRefresh={onRefresh}
                mysterySolved={mysterySolved}
                mysteryPaused={mysteryPaused}
                authorAlreadyWon={authorAlreadyWon}
            />
            {allReplies.length > 0 && (
                <div className={styles.threadContainer}>
                    <button className={styles.collapseBtn} onClick={() => setCollapsed(!collapsed)}>
                        {collapsed
                            ? `Show ${allReplies.length} ${allReplies.length === 1 ? "reply" : "replies"}`
                            : `Hide ${allReplies.length} ${allReplies.length === 1 ? "reply" : "replies"}`}
                    </button>
                    {!collapsed && (
                        <div className={styles.thread}>
                            {allReplies.map(({ reply, replyToName }) => (
                                <SingleAttempt
                                    key={reply.id}
                                    attempt={reply}
                                    mysteryId={mysteryId}
                                    isAuthor={isAuthor}
                                    onRefresh={onRefresh}
                                    replyToName={replyToName}
                                    mysterySolved={mysterySolved}
                                    mysteryPaused={mysteryPaused}
                                    authorAlreadyWon={authorAlreadyWon}
                                />
                            ))}
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
