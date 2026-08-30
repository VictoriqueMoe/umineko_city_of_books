import { Link, useParams } from "react-router";
import type { PostComment, SecretComment } from "../../types/api";
import { useSecretRoom } from "../../hooks/useSecretRoom";
import { useCommentHandlers } from "../../hooks/useCommentHandlers";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RoleStyledName } from "../../components/RoleStyledName/RoleStyledName";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { Toast } from "../../components/Toast/Toast";
import styles from "./SecretDetailPage.module.css";

export function SecretDetailPage() {
    const { id = "" } = useParams<{ id: string }>();
    usePageTitle("Secret");
    const { user } = useAuth();
    const { detail, loading, refresh, solvedByName, dismissSolved } = useSecretRoom(id);

    const { createCommentFn, updateFn, deleteFn, likeFn, unlikeFn, uploadMediaFn } = useCommentHandlers("secret", id, {
        enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"],
    });

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
                    <Link to="/secrets">Secrets</Link> / <bdi>{detail.title}</bdi>
                </div>
                <h1 dir="auto" className={styles.title}>
                    {detail.title}
                </h1>
                <p dir="auto" className={styles.description}>
                    {detail.description}
                </p>

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
                <div dir="auto" className={styles.riddle}>
                    {detail.riddle}
                </div>
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

            <CommentsSection
                comments={(detail.comments ?? []).map(secretCommentToPostComment)}
                targetId={detail.id}
                user={user}
                onChanged={() => refresh()}
                title="Discussion"
                emptyText="No one has left a word yet."
                composerPosition="top"
                linkPrefix={`/secrets/${detail.id}`}
                reportType="secret_comment"
                likeFn={likeFn}
                unlikeFn={unlikeFn}
                deleteFn={deleteFn}
                updateFn={updateFn}
                createCommentFn={createCommentFn}
                uploadMediaFn={uploadMediaFn}
            />

            {solvedByName && (
                <Toast variant="arcane" duration={6000} onDismiss={dismissSolved}>
                    <bdi>{solvedByName}</bdi> spoke the witch&apos;s name.
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
