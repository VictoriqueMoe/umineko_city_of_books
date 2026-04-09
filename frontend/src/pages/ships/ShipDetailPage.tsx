import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { PostComment, ShipCharacter, ShipDetail } from "../../types/api";
import {
    createShipComment,
    deleteShip,
    deleteShipComment,
    getShip,
    likeShipComment,
    unlikeShipComment,
    updateShip,
    updateShipComment,
    uploadShipCommentMedia,
    voteShip,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { relativeTime } from "../../utils/notifications";
import { CharacterPicker } from "../../components/CharacterPicker/CharacterPicker";
import { CharacterPills } from "./ShipsListPage";
import { ShareButton } from "../../components/ShareButton/ShareButton";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import styles from "./ShipPages.module.css";

function characterPillClass(series: string): string {
    if (series === "umineko") {
        return `${styles.selectedCharacter} ${styles.characterPillUmineko}`;
    }
    if (series === "higurashi") {
        return `${styles.selectedCharacter} ${styles.characterPillHigurashi}`;
    }
    return `${styles.selectedCharacter} ${styles.characterPillOc}`;
}

export function ShipDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const [ship, setShip] = useState<ShipDetail | null>(null);
    usePageTitle(ship?.title ?? "Ship");
    const [loading, setLoading] = useState(true);
    const [voting, setVoting] = useState(false);
    const [lightboxOpen, setLightboxOpen] = useState(false);
    const [editing, setEditing] = useState(false);
    const [editTitle, setEditTitle] = useState("");
    const [editDesc, setEditDesc] = useState("");
    const [editChars, setEditChars] = useState<ShipCharacter[]>([]);
    const [saving, setSaving] = useState(false);
    const [editError, setEditError] = useState("");
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

    function startEdit() {
        if (!ship) {
            return;
        }
        setEditTitle(ship.title);
        setEditDesc(ship.description);
        setEditChars(ship.characters.map((c, i) => ({ ...c, sort_order: i })));
        setEditError("");
        setEditing(true);
    }

    function cancelEdit() {
        setEditing(false);
        setEditError("");
    }

    function addEditCharacter(character: ShipCharacter) {
        setEditChars(prev => [...prev, { ...character, sort_order: prev.length }]);
    }

    function removeEditCharacter(index: number) {
        setEditChars(prev => prev.filter((_, i) => i !== index).map((c, i) => ({ ...c, sort_order: i })));
    }

    async function saveEdit() {
        if (!ship) {
            return;
        }
        setEditError("");
        if (!editTitle.trim()) {
            setEditError("Title is required");
            return;
        }
        if (editChars.length < 2) {
            setEditError("A ship needs at least 2 characters");
            return;
        }
        setSaving(true);
        try {
            await updateShip(ship.id, {
                title: editTitle.trim(),
                description: editDesc.trim(),
                characters: editChars,
            });
            setEditing(false);
            fetchShip();
        } catch (e) {
            setEditError(e instanceof Error ? e.message : "Failed to update ship");
        } finally {
            setSaving(false);
        }
    }

    if (loading) {
        return <div className="loading">Loading ship...</div>;
    }

    if (!ship) {
        return <div className="empty-state">Ship not found.</div>;
    }

    const isAuthor = user?.id === ship.author.id;
    const canEdit = isAuthor || can(user?.role, "edit_any_post");
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
                    {editing ? (
                        <>
                            <div className={styles.formSection}>
                                <label className={styles.formLabel}>Ship Title</label>
                                <Input
                                    type="text"
                                    value={editTitle}
                                    onChange={e => setEditTitle(e.target.value)}
                                    fullWidth
                                />
                            </div>
                            <div className={styles.formSection}>
                                <label className={styles.formLabel}>Characters (at least 2)</label>
                                <CharacterPicker onAdd={addEditCharacter} existing={editChars} />
                                {editChars.length > 0 && (
                                    <div className={styles.selectedCharacters}>
                                        {editChars.map((c, i) => (
                                            <span
                                                key={`${c.series}-${c.character_id ?? c.character_name}-${i}`}
                                                className={characterPillClass(c.series)}
                                            >
                                                {c.character_name}
                                                <button
                                                    type="button"
                                                    className={styles.removeCharBtn}
                                                    onClick={() => removeEditCharacter(i)}
                                                    aria-label="Remove character"
                                                >
                                                    ×
                                                </button>
                                            </span>
                                        ))}
                                    </div>
                                )}
                            </div>
                            <div className={styles.formSection}>
                                <label className={styles.formLabel}>Why do you ship it?</label>
                                <textarea
                                    className={styles.formTextarea}
                                    value={editDesc}
                                    onChange={e => setEditDesc(e.target.value)}
                                    rows={5}
                                />
                            </div>
                            {editError && <ErrorBanner message={editError} />}
                            <div className={styles.formActions}>
                                <Button variant="ghost" onClick={cancelEdit} disabled={saving}>
                                    Cancel
                                </Button>
                                <Button
                                    variant="primary"
                                    onClick={saveEdit}
                                    disabled={saving || !editTitle.trim() || editChars.length < 2}
                                >
                                    {saving ? "Saving..." : "Save"}
                                </Button>
                            </div>
                        </>
                    ) : (
                        <>
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
                                <div style={{ display: "flex", gap: "0.5rem" }}>
                                    {canEdit && (
                                        <Button variant="secondary" size="small" onClick={startEdit}>
                                            Edit
                                        </Button>
                                    )}
                                    {canDelete && (
                                        <Button variant="danger" size="small" onClick={handleDelete}>
                                            Delete
                                        </Button>
                                    )}
                                </div>
                            </div>

                            {ship.description && <p className={styles.detailDescription}>{ship.description}</p>}
                        </>
                    )}

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
                        <ShareButton contentId={ship.id} contentType="ship" contentTitle={ship.title} />
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
                        highlightedId={highlightedComment ?? undefined}
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
