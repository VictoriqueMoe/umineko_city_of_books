import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import type { PostComment, ShipDetail } from "../../types/api";
import {
    createShipComment,
    deleteShip,
    deleteShipComment,
    getShip,
    likeShipComment,
    unlikeShipComment,
    updateShipComment,
    uploadShipCommentMedia,
    voteShip,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { relativeTime } from "../../utils/notifications";
import { CharacterPills } from "./ShipsListPage";
import styles from "./ShipPages.module.css";

export function ShipDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const [ship, setShip] = useState<ShipDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const [voting, setVoting] = useState(false);
    const [lightboxOpen, setLightboxOpen] = useState(false);
    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const fetchShip = useCallback(() => {
        if (!id) {
            return;
        }
        getShip(id)
            .then(setShip)
            .catch(() => setShip(null))
            .finally(() => setLoading(false));
    }, [id]);

    useEffect(() => {
        fetchShip();
    }, [fetchShip]);

    useEffect(() => {
        if (!ship || loading || !highlightedComment) {
            return;
        }
        requestAnimationFrame(() => {
            const el = document.getElementById(`comment-${highlightedComment}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
            }
        });
    }, [ship, loading, highlightedComment]);

    async function handleVote(value: number) {
        if (!ship || voting) {
            return;
        }
        const current = ship.user_vote ?? 0;
        const newValue = current === value ? 0 : value;
        setVoting(true);
        try {
            await voteShip(ship.id, newValue);
            setShip({
                ...ship,
                vote_score: ship.vote_score - current + newValue,
                user_vote: newValue,
                is_crackship: ship.vote_score - current + newValue <= -3,
            });
        } catch {
            // ignore
        } finally {
            setVoting(false);
        }
    }

    async function handleDelete() {
        if (!ship || !window.confirm("Delete this ship? This cannot be undone.")) {
            return;
        }
        await deleteShip(ship.id);
        navigate("/ships");
    }

    if (loading) {
        return <div className="loading">Loading ship...</div>;
    }

    if (!ship) {
        return <div className="empty-state">Ship not found.</div>;
    }

    const isAuthor = user?.id === ship.author.id;
    const canDelete = isAuthor || can(user?.role, "delete_any_post");
    const userVote = ship.user_vote ?? 0;

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/ships")}>
                &larr; All Ships
            </span>

            <div className={styles.detailHeader}>
                {(ship.image_url || ship.thumbnail_url) && (
                    <img
                        className={styles.detailImage}
                        src={ship.image_url || ship.thumbnail_url}
                        alt={ship.title}
                        onClick={() => setLightboxOpen(true)}
                        style={{ cursor: "zoom-in" }}
                    />
                )}
                <div className={styles.detailBody}>
                    <div
                        style={{
                            display: "flex",
                            justifyContent: "space-between",
                            alignItems: "flex-start",
                            gap: "1rem",
                        }}
                    >
                        <div style={{ flex: 1 }}>
                            <h1 className={styles.detailTitle}>{ship.title}</h1>
                            <div className={styles.detailMeta}>
                                <ProfileLink user={ship.author} size="small" />
                                <span>{relativeTime(ship.created_at)}</span>
                                {ship.is_crackship && <span className={styles.crackshipBadge}>Crackship</span>}
                            </div>
                            <CharacterPills characters={ship.characters} />
                        </div>
                        {canDelete && (
                            <Button variant="danger" size="small" onClick={handleDelete}>
                                Delete
                            </Button>
                        )}
                    </div>

                    {ship.description && <p className={styles.detailDescription}>{ship.description}</p>}

                    <div className={styles.voteRow}>
                        <Button variant="ghost" size="small" onClick={() => handleVote(1)} disabled={!user || voting}>
                            {userVote === 1 ? "\u25B2" : "\u25B3"}
                        </Button>
                        <span className={styles.voteScore}>
                            {ship.vote_score > 0 ? "+" : ""}
                            {ship.vote_score}
                        </span>
                        <Button variant="ghost" size="small" onClick={() => handleVote(-1)} disabled={!user || voting}>
                            {userVote === -1 ? "\u25BC" : "\u25BD"}
                        </Button>
                    </div>
                </div>
            </div>

            <div className={styles.commentsSection}>
                <h3 className={styles.commentsTitle}>
                    Comments {ship.comments.length > 0 && `(${ship.comments.length})`}
                </h3>
                {ship.comments.map(c => (
                    <CommentItem
                        key={c.id}
                        comment={c as unknown as PostComment}
                        postId={ship.id}
                        onDelete={fetchShip}
                        highlighted={c.id === highlightedComment}
                        linkPrefix="/ships"
                        reportType="ship_comment"
                        likeFn={likeShipComment}
                        unlikeFn={unlikeShipComment}
                        deleteFn={deleteShipComment}
                        updateFn={updateShipComment}
                        createCommentFn={createShipComment}
                        uploadMediaFn={uploadShipCommentMedia}
                        viewerBlocked={ship.viewer_blocked}
                    />
                ))}
                {ship.comments.length === 0 && <p className="empty-state">No comments yet.</p>}
                {ship.viewer_blocked && <p className="empty-state">You cannot interact with this ship.</p>}
                {user && id && !ship.viewer_blocked && (
                    <CommentComposer
                        postId={id}
                        onCreated={fetchShip}
                        createCommentFn={createShipComment}
                        uploadMediaFn={uploadShipCommentMedia}
                    />
                )}
            </div>

            {lightboxOpen && (ship.image_url || ship.thumbnail_url) && (
                <Lightbox
                    src={ship.image_url || ship.thumbnail_url || ""}
                    alt={ship.title}
                    onClose={() => setLightboxOpen(false)}
                />
            )}
        </div>
    );
}
