import { useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import type { ShipCharacter } from "../../types/api";
import { useShip } from "../../hooks/queries/ship";
import { useDeleteShip, useUpdateShip, useVoteShip } from "../../hooks/mutations/ship";
import { useAuth } from "../../hooks/useAuth";
import { useCommentHandlers } from "../../hooks/useCommentHandlers";
import { contentPermissions } from "../../domain/contentPermissions";
import { errorMessage } from "../../utils/errorMessage";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import { CharacterPicker } from "../../components/CharacterPicker/CharacterPicker";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import { renderRich } from "../../components/richText/richText";
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
    const { ship, loading, refresh } = useShip(id ?? "");
    usePageTitle(ship?.title ?? "Ship");
    const [voting, setVoting] = useState(false);
    const [lightboxOpen, setLightboxOpen] = useState(false);
    const [editing, setEditing] = useState(false);
    const [editTitle, setEditTitle] = useState("");
    const [editDesc, setEditDesc] = useState("");
    const [editChars, setEditChars] = useState<ShipCharacter[]>([]);
    const [saving, setSaving] = useState(false);
    const [editError, setEditError] = useState("");
    const [voteError, setVoteError] = useState("");
    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const voteShipMutation = useVoteShip(ship?.id ?? "");
    const deleteShipMutation = useDeleteShip();
    const updateShipMutation = useUpdateShip(ship?.id ?? "");
    const { createCommentFn, updateFn, deleteFn, likeFn, unlikeFn, uploadMediaFn } = useCommentHandlers(
        "ship",
        ship?.id ?? "",
        { enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"] },
    );

    const fetchShip = () => {
        refresh();
    };

    useScrollToHash(!loading && !!ship, highlightedComment ? `comment-${highlightedComment}` : null);

    async function handleVote(value: number) {
        if (!ship || voting) {
            return;
        }
        const current = ship.user_vote ?? 0;
        const newValue = current === value ? 0 : value;
        setVoting(true);
        setVoteError("");
        try {
            await voteShipMutation.mutateAsync(newValue);
        } catch (thrown) {
            setVoteError(errorMessage(thrown, "Failed to record your vote"));
        }
        setVoting(false);
    }

    async function handleDelete() {
        if (!ship || !window.confirm("Delete this ship? This cannot be undone.")) {
            return;
        }
        await deleteShipMutation.mutateAsync(ship.id);
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
            await updateShipMutation.mutateAsync({
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

    const { canEdit, canDelete } = contentPermissions(user, { family: "ship", authorId: ship.author.id });
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
                                                <span dir="auto">{c.character_name}</span>
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
                                <MentionTextArea value={editDesc} onChange={setEditDesc} rows={5} showColours />
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
                                    <h1 dir="auto" className={styles.detailTitle}>
                                        {ship.title}
                                    </h1>
                                    <div className={styles.detailMeta}>
                                        <ProfileLink user={ship.author} size="small" />
                                        <RelativeTimestamp value={ship.created_at} />
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

                            {ship.description && (
                                <div dir="auto" className={styles.detailDescription}>
                                    {renderRich(ship.description)}
                                </div>
                            )}
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
                    {voteError && <ErrorBanner message={voteError} />}
                </div>
            </div>

            <CommentsSection
                comments={ship.comments ?? []}
                targetId={ship.id}
                user={user}
                onChanged={fetchShip}
                blockedText="You cannot interact with this ship."
                viewerBlocked={ship.viewer_blocked}
                highlightedId={highlightedComment ?? undefined}
                linkPrefix="/ships"
                reportType="ship_comment"
                likeFn={likeFn}
                unlikeFn={unlikeFn}
                deleteFn={deleteFn}
                updateFn={updateFn}
                createCommentFn={createCommentFn}
                uploadMediaFn={uploadMediaFn}
            />

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
