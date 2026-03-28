import {useCallback, useState} from "react";
import {useNavigate, useParams} from "react-router";
import {useTheory} from "../hooks/useTheory";
import {useVote} from "../hooks/useVote";
import {useAuth} from "../hooks/useAuth";
import {deleteTheory, voteTheory} from "../api/endpoints";
import {Modal} from "../components/common/Modal";
import {ProfileLink} from "../components/common/ProfileLink";
import {VoteButton} from "../components/theory/VoteButton";
import {EvidenceList} from "../components/theory/EvidenceList";
import {ResponseList} from "../components/theory/ResponseCard";
import {ResponseEditor} from "../components/theory/ResponseEditor";

export function TheoryPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const theoryId = parseInt(id ?? "0", 10);
    const { theory, loading, refresh } = useTheory(theoryId);

    const voteFn = useCallback(
        async (value: number) => {
            await voteTheory(theoryId, value);
        },
        [theoryId],
    );

    const { score, userVote, vote } = useVote(theory?.vote_score ?? 0, theory?.user_vote ?? 0, voteFn);
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

    const isAuthor = user && theory && user.id === theory.author.id;

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

    const withLove = theory.responses?.filter(r => r.side === "with_love") ?? [];
    const withoutLove = theory.responses?.filter(r => r.side === "without_love") ?? [];

    return (
        <div className="theory-detail-page">
            <button className="back-btn" onClick={() => navigate(-1)}>
                &larr; Back
            </button>

            <div className="theory-detail-preamble">
                <ProfileLink user={theory.author} size="large" showName={false} />
                {theory.author.display_name} declares in blue:
            </div>

            <div className="theory-detail-card">
                <div className="theory-detail-header">
                    <VoteButton score={score} userVote={userVote} onVote={vote} />
                    <div className="theory-detail-info">
                        <h2 className="theory-detail-title">{theory.title}</h2>
                        {theory.episode > 0 && <span className="truth-episode">Episode {theory.episode}</span>}
                    </div>
                    {isAuthor && (
                        <div className="theory-author-actions">
                            <button className="nav-btn" onClick={() => navigate(`/theory/${theoryId}/edit`)}>
                                Edit
                            </button>
                            <button className="delete-btn" onClick={() => setDeleteConfirmOpen(true)}>
                                Delete
                            </button>
                        </div>
                    )}
                </div>

                <div className="theory-detail-body blue-truth">{theory.body}</div>

                <EvidenceList evidence={theory.evidence ?? []} />
            </div>

            <div className="debate-section">
                <div className="debate-column">
                    <h3 className="debate-header with-love">With love, it can be seen ({withLove.length})</h3>
                    {withLove.length > 0 ? (
                        <ResponseList responses={withLove} theoryId={theoryId} onDeleted={refresh} />
                    ) : (
                        <div className="empty-state">No supporters yet.</div>
                    )}
                </div>

                <div className="debate-column">
                    <h3 className="debate-header without-love">
                        Without love, it cannot be seen ({withoutLove.length})
                    </h3>
                    {withoutLove.length > 0 ? (
                        <ResponseList responses={withoutLove} theoryId={theoryId} onDeleted={refresh} />
                    ) : (
                        <div className="empty-state">No deniers yet.</div>
                    )}
                </div>
            </div>

            {user && <ResponseEditor theoryId={theoryId} onCreated={refresh} />}

            {!user && (
                <div className="empty-state">
                    <button className="nav-btn" onClick={() => navigate("/login")}>
                        Sign in to respond
                    </button>
                </div>
            )}

            <Modal isOpen={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)} title="Delete Theory">
                <div style={{ padding: "1.25rem" }}>
                    <p style={{ marginBottom: "1rem" }}>
                        Are you sure you want to delete this theory? This cannot be undone.
                    </p>
                    <div className="editor-actions">
                        <button className="nav-btn" onClick={() => setDeleteConfirmOpen(false)}>
                            Cancel
                        </button>
                        <button className="delete-btn" onClick={handleDelete} style={{ fontSize: "0.9rem" }}>
                            Delete Theory
                        </button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}
