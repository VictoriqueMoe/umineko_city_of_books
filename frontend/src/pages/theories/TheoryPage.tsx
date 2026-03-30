import { useCallback, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useTheory } from "../../hooks/useTheory";
import { useVote } from "../../hooks/useVote";
import { useAuth } from "../../hooks/useAuth";
import { deleteTheory, voteTheory } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Modal } from "../../components/Modal/Modal";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { VoteButton } from "../../components/theory/VoteButton/VoteButton";
import { EvidenceList } from "../../components/theory/EvidenceList/EvidenceList";
import { ResponseList } from "../../components/theory/ResponseCard/ResponseCard";
import { ResponseEditor } from "../../components/theory/ResponseEditor/ResponseEditor";
import { CredibilityBadge } from "../../components/theory/CredibilityBadge/CredibilityBadge";
import { can } from "../../utils/permissions";
import styles from "./TheoryPage.module.css";

export function TheoryPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const theoryId = id ?? "";
    const { theory, loading, refresh } = useTheory(theoryId);
    const [spoilerDismissed, setSpoilerDismissed] = useState(false);

    const voteFn = useCallback(
        async (value: number) => {
            await voteTheory(theoryId, value);
        },
        [theoryId],
    );

    const { score, userVote, vote } = useVote(theory?.vote_score ?? 0, theory?.user_vote ?? 0, voteFn);
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

    const isAuthor = user && theory && user.id === theory.author.id;
    const canDelete = isAuthor || can(user?.role, "delete_any_theory");

    async function handleDelete() {
        await deleteTheory(theoryId);
        navigate("/");
    }

    if (loading) {
        return <div className="loading">Consulting the game board...</div>;
    }

    if (!theory) {
        return <div className="empty-state">Theory not found.</div>;
    }

    const isSpoiler =
        !spoilerDismissed &&
        (user?.episode_progress ?? 0) > 0 &&
        theory.episode > 0 &&
        theory.episode >= (user?.episode_progress ?? 0);

    if (isSpoiler) {
        return (
            <div className={styles.page}>
                <Button variant="secondary" className={styles.backBtn} onClick={() => navigate(-1)}>
                    &larr; Back
                </Button>
                <div className={styles.spoilerWarning}>
                    <h2>Spoiler Warning</h2>
                    <p>
                        This theory references Episode {theory.episode}, which is beyond your current reading progress.
                    </p>
                    <Button variant="primary" onClick={() => setSpoilerDismissed(true)}>
                        Continue anyway
                    </Button>
                </div>
            </div>
        );
    }

    const withLove = theory.responses?.filter(r => r.side === "with_love") ?? [];
    const withoutLove = theory.responses?.filter(r => r.side === "without_love") ?? [];

    return (
        <div className={styles.page}>
            <Button variant="secondary" className={styles.backBtn} onClick={() => navigate(-1)}>
                &larr; Back
            </Button>

            <div className={styles.preamble}>
                <ProfileLink user={theory.author} size="large" showName={false} />
                {theory.author.display_name} declares in blue:
            </div>

            <div className={styles.detailCard}>
                <div className={styles.detailHeader}>
                    <VoteButton score={score} userVote={userVote} onVote={vote} />
                    <div className={styles.detailInfo}>
                        <h2 className={styles.detailTitle}>{theory.title}</h2>
                        <div className={styles.detailMeta}>
                            {theory.episode > 0 && <span className={styles.episode}>Episode {theory.episode}</span>}
                            <CredibilityBadge score={theory.credibility_score} />
                        </div>
                    </div>
                    {(isAuthor || canDelete) && (
                        <div className={styles.authorActions}>
                            {isAuthor && (
                                <Button
                                    variant="secondary"
                                    size="small"
                                    onClick={() => navigate(`/theory/${theoryId}/edit`)}
                                >
                                    Edit
                                </Button>
                            )}
                            {canDelete && (
                                <Button variant="danger" size="small" onClick={() => setDeleteConfirmOpen(true)}>
                                    Delete
                                </Button>
                            )}
                        </div>
                    )}
                </div>

                <div className={styles.body}>{theory.body}</div>

                <EvidenceList evidence={theory.evidence ?? []} />
            </div>

            <div className={styles.debateSection}>
                <div>
                    <h3 className={`${styles.debateHeader} ${styles.debateHeaderWithLove}`}>
                        With love, it can be seen ({withLove.length})
                    </h3>
                    {withLove.length > 0 ? (
                        <ResponseList responses={withLove} theoryId={theoryId} onDeleted={refresh} />
                    ) : (
                        <div className="empty-state">No supporters yet.</div>
                    )}
                </div>

                <div>
                    <h3 className={`${styles.debateHeader} ${styles.debateHeaderWithoutLove}`}>
                        Without love, it cannot be seen ({withoutLove.length})
                    </h3>
                    {withoutLove.length > 0 ? (
                        <ResponseList responses={withoutLove} theoryId={theoryId} onDeleted={refresh} />
                    ) : (
                        <div className="empty-state">No deniers yet.</div>
                    )}
                </div>
            </div>

            {user && !isAuthor && <ResponseEditor theoryId={theoryId} onCreated={refresh} />}

            {!user && (
                <div className="empty-state">
                    <Button variant="primary" onClick={() => navigate("/login")}>
                        Sign in to respond
                    </Button>
                </div>
            )}

            <Modal isOpen={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)} title="Delete Theory">
                <div style={{ padding: "1.25rem" }}>
                    <p style={{ marginBottom: "1rem" }}>
                        Are you sure you want to delete this theory? This cannot be undone.
                    </p>
                    <div className={styles.confirmActions}>
                        <Button variant="secondary" onClick={() => setDeleteConfirmOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="danger" onClick={handleDelete}>
                            Delete Theory
                        </Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}
