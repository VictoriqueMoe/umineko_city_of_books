import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router";
import type {
    PostComment,
    SecretComment,
    SecretDetailResponse,
    SecretLeaderboardEntry,
    SecretProgressEvent,
    SecretSolvedEvent,
} from "../../types/api";
import {
    createSecretComment,
    deleteSecretComment,
    getSecret,
    likeSecretComment,
    unlikeSecretComment,
    updateSecretComment,
    uploadSecretCommentMedia,
} from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RoleStyledName } from "../../components/RoleStyledName/RoleStyledName";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { Toast } from "../../components/Toast/Toast";
import styles from "./SecretDetailPage.module.css";

function sortLeaderboard(rows: SecretLeaderboardEntry[]): SecretLeaderboardEntry[] {
    return [...rows].sort((a, b) => {
        if (a.solved !== b.solved) {
            return a.solved ? -1 : 1;
        }
        if (b.pieces_collected !== a.pieces_collected) {
            return b.pieces_collected - a.pieces_collected;
        }
        return a.user.display_name.localeCompare(b.user.display_name);
    });
}

export function SecretDetailPage() {
    const { id = "" } = useParams<{ id: string }>();
    usePageTitle("Secret");
    const { user } = useAuth();
    const { addWSListener, sendWSMessage, wsEpoch } = useNotifications();
    const [detail, setDetail] = useState<SecretDetailResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [toast, setToast] = useState<string | null>(null);

    const fetchDetail = useCallback(() => {
        if (!id) {
            return;
        }
        getSecret(id)
            .then(d => setDetail({ ...d, leaderboard: sortLeaderboard(d.leaderboard) }))
            .catch(() => setDetail(null))
            .finally(() => setLoading(false));
    }, [id]);

    useEffect(() => {
        fetchDetail();
    }, [fetchDetail]);

    const refresh = fetchDetail;

    useEffect(() => {
        if (!id || wsEpoch === 0) {
            return;
        }
        sendWSMessage({ type: "secret_join", data: { secret_id: id } });
        fetchDetail();
        return () => {
            sendWSMessage({ type: "secret_leave", data: { secret_id: id } });
        };
    }, [id, sendWSMessage, wsEpoch, fetchDetail]);

    useEffect(() => {
        if (!id) {
            return;
        }
        return addWSListener(msg => {
            if (msg.type === "secret_progress") {
                const data = msg.data as SecretProgressEvent;
                if (data.secret_id !== id) {
                    return;
                }
                setDetail(prev => {
                    if (!prev) {
                        return prev;
                    }
                    const next = [...prev.leaderboard];
                    const existingIdx = next.findIndex(e => e.user.id === data.user.id);
                    if (existingIdx >= 0) {
                        next[existingIdx] = {
                            ...next[existingIdx],
                            pieces_collected: data.pieces_collected,
                        };
                    } else {
                        next.push({ user: data.user, pieces_collected: data.pieces_collected, solved: false });
                    }
                    return { ...prev, leaderboard: sortLeaderboard(next) };
                });
            } else if (msg.type === "secret_solved") {
                const data = msg.data as SecretSolvedEvent;
                if (data.secret_id !== id) {
                    return;
                }
                setToast(`${data.solver.display_name} spoke the witch's name.`);
                setDetail(prev => {
                    if (!prev) {
                        return prev;
                    }
                    const next = prev.leaderboard.map(e => (e.user.id === data.solver.id ? { ...e, solved: true } : e));
                    return {
                        ...prev,
                        solved: true,
                        solver: data.solver,
                        solved_at: data.solved_at,
                        leaderboard: sortLeaderboard(next),
                    };
                });
            }
        });
    }, [id, addWSListener]);

    if (loading) {
        return <div className="loading">Consulting the game board...</div>;
    }

    if (!detail) {
        return (
            <div className={styles.page}>
                <div className={styles.empty}>Secret not found.</div>
                <p className={styles.breadcrumb}>
                    <Link to="/secrets">Back to secrets</Link>
                </p>
            </div>
        );
    }

    const leaderboard = detail.leaderboard;

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <div className={styles.breadcrumb}>
                    <Link to="/secrets">Secrets</Link> / {detail.title}
                </div>
                <h1 className={styles.title}>{detail.title}</h1>
                <p className={styles.description}>{detail.description}</p>

                <div className={`${styles.statusBar} ${detail.solved ? styles.statusSolved : styles.statusOpen}`}>
                    {detail.solved && detail.solver ? (
                        <span>
                            <strong>Solved</strong> by{" "}
                            <RoleStyledName name={detail.solver.display_name} role={detail.solver.role} />
                        </span>
                    ) : (
                        <span>Open. No one has spoken the answer yet.</span>
                    )}
                    {user && detail.viewer_progress > 0 && (
                        <span className={styles.progressChip}>
                            You: {detail.viewer_progress} / {detail.total_pieces}
                        </span>
                    )}
                </div>
            </div>

            <section className={styles.section}>
                <h2 className={styles.sectionTitle}>The Riddle</h2>
                <div className={styles.riddle}>{detail.riddle}</div>
            </section>

            <section className={styles.section}>
                <h2 className={styles.sectionTitle}>
                    Progress ({leaderboard.length} hunter{leaderboard.length === 1 ? "" : "s"})
                </h2>
                {leaderboard.length === 0 ? (
                    <div className={styles.empty}>No one has picked up a piece yet. Be the first.</div>
                ) : (
                    <div className={styles.leaderboard}>
                        {leaderboard.map((entry, idx) => (
                            <div
                                key={entry.user.id}
                                className={`${styles.lbRow} ${entry.solved ? styles.lbRowSolved : ""}`}
                            >
                                <span className={styles.lbRank}>#{idx + 1}</span>
                                <span className={styles.lbUser}>
                                    <ProfileLink user={entry.user} size="small" />
                                </span>
                                <span className={styles.lbPieces}>
                                    {entry.pieces_collected} / {detail.total_pieces}
                                </span>
                                {entry.solved && <span className={styles.lbTrophy}>{"\u2605"}</span>}
                            </div>
                        ))}
                    </div>
                )}
            </section>

            <section className={styles.section}>
                <h2 className={styles.sectionTitle}>Discussion</h2>
                {user && (
                    <CommentComposer
                        postId={detail.id}
                        onCreated={refresh}
                        createCommentFn={createSecretComment}
                        uploadMediaFn={uploadSecretCommentMedia}
                    />
                )}
                <div className={styles.comments}>
                    {detail.comments.length === 0 ? (
                        <div className={styles.empty}>No one has left a word yet.</div>
                    ) : (
                        detail.comments.map(c => (
                            <CommentItem
                                key={c.id}
                                comment={secretCommentToPostComment(c)}
                                postId={detail.id}
                                onDelete={refresh}
                                linkPrefix={`/secrets/${detail.id}`}
                                reportType="secret_comment"
                                likeFn={likeSecretComment}
                                unlikeFn={unlikeSecretComment}
                                deleteFn={deleteSecretComment}
                                updateFn={updateSecretComment}
                                createCommentFn={createSecretComment}
                                uploadMediaFn={uploadSecretCommentMedia}
                            />
                        ))
                    )}
                </div>
            </section>

            {toast && (
                <Toast variant="arcane" duration={6000} onDismiss={() => setToast(null)}>
                    {toast}
                </Toast>
            )}
        </div>
    );
}

function secretCommentToPostComment(c: SecretComment): PostComment {
    return {
        ...c,
        replies: c.replies?.map(secretCommentToPostComment),
    } as PostComment;
}
