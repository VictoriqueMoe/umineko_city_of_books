import { useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { useArt } from "../../hooks/queries/art";
import { useDeleteArt, useLikeArt, useUnlikeArt, useUpdateArt } from "../../hooks/mutations/art";
import { useAuth } from "../../hooks/useAuth";
import { useCommentHandlers } from "../../hooks/useCommentHandlers";
import { useResetOnChange } from "../../hooks/useResetOnChange";
import { contentPermissions, isContentOwner, type ContentSubject } from "../../domain/contentPermissions";
import { renderRich } from "../../components/richText/richText";
import { parseServerDate } from "../../utils/time";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Button } from "../../components/Button/Button";
import { Modal } from "../../components/Modal/Modal";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import { TagInput } from "../../components/art/TagInput/TagInput";
import { SpoilerImage } from "../../components/SpoilerImage/SpoilerImage";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { ReportButton } from "../../components/ReportButton/ReportButton";
import { ShareButton } from "../../components/ShareButton/ShareButton";
import styles from "./ArtDetailPage.module.css";

export function ArtDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { art, loading, refresh } = useArt(id ?? "");
    usePageTitle(art?.title ?? "Art");
    const liked = art?.user_liked ?? false;
    const likeCount = art?.like_count ?? 0;
    const [editing, setEditing] = useState(false);
    const [editTitle, setEditTitle] = useState("");
    const [editDesc, setEditDesc] = useState("");
    const [editTags, setEditTags] = useState<string[]>([]);
    const [editSpoiler, setEditSpoiler] = useState(false);
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
    const [lightboxOpen, setLightboxOpen] = useState(false);

    useResetOnChange(id, () => {
        setEditing(false);
        setEditTitle("");
        setEditDesc("");
        setEditTags([]);
        setEditSpoiler(false);
        setDeleteConfirmOpen(false);
        setLightboxOpen(false);
    });

    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const likeArtMutation = useLikeArt();
    const unlikeArtMutation = useUnlikeArt();
    const deleteArtMutation = useDeleteArt();
    const updateArtMutation = useUpdateArt(id ?? "");
    const { createCommentFn, updateFn, deleteFn, likeFn, unlikeFn, uploadMediaFn } = useCommentHandlers(
        "art",
        id ?? "",
        { enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"] },
    );

    useScrollToHash(!loading && !!art, highlightedComment ? `comment-${highlightedComment}` : null);

    async function handleLike() {
        if (!id) {
            return;
        }

        const mutation = liked ? unlikeArtMutation : likeArtMutation;
        await mutation.mutateAsync(id).catch(() => undefined);
    }

    async function handleDelete() {
        if (!id) {
            return;
        }
        await deleteArtMutation.mutateAsync(id);
        navigate(-1);
    }

    function startEdit() {
        if (!art) {
            return;
        }
        setEditTitle(art.title);
        setEditDesc(art.description);
        setEditTags([...art.tags]);
        setEditSpoiler(art.is_spoiler);
        setEditing(true);
    }

    async function saveEdit() {
        if (!id || !editTitle.trim()) {
            return;
        }
        await updateArtMutation.mutateAsync({
            title: editTitle.trim(),
            description: editDesc.trim(),
            tags: editTags,
            is_spoiler: editSpoiler,
        });
        setEditing(false);
        refresh();
    }

    if (loading) {
        return <div className="loading">Loading art...</div>;
    }

    if (!art) {
        return <div className="empty-state">Art not found.</div>;
    }

    const subject: ContentSubject = { family: "art", authorId: art.author.id };
    const isAuthor = isContentOwner(user, subject);
    const { canEdit, canDelete } = contentPermissions(user, subject);

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate(-1)}>
                &larr; Back to Gallery
            </span>

            <SpoilerImage
                src={art.image_url}
                alt={art.title}
                isSpoiler={art.is_spoiler}
                className={styles.imageSection}
                imageClassName={styles.fullImage}
                onClick={() => setLightboxOpen(true)}
            />

            <div className={styles.detailCard}>
                {editing ? (
                    <div className={styles.editSection}>
                        <input
                            dir="auto"
                            className={styles.editTitle}
                            value={editTitle}
                            onChange={e => setEditTitle(e.target.value)}
                            placeholder="Title"
                        />
                        <MentionTextArea value={editDesc} onChange={setEditDesc} placeholder="Description" rows={3} />
                        <TagInput tags={editTags} onChange={setEditTags} />
                        <ToggleSwitch
                            enabled={editSpoiler}
                            onChange={setEditSpoiler}
                            label="Contains spoilers"
                            description="Image will be blurred until clicked"
                        />
                        <div className={styles.editActions}>
                            <Button variant="secondary" size="small" onClick={() => setEditing(false)}>
                                Cancel
                            </Button>
                            <Button variant="primary" size="small" onClick={saveEdit} disabled={!editTitle.trim()}>
                                Save
                            </Button>
                        </div>
                    </div>
                ) : (
                    <>
                        <h1 dir="auto" className={styles.title}>
                            {art.title}
                        </h1>
                        {art.description && (
                            <div dir="auto" className={styles.description}>
                                {renderRich(art.description)}
                            </div>
                        )}
                    </>
                )}

                <div className={styles.metaRow}>
                    <ProfileLink user={art.author} size="medium" />
                    <span className={styles.date}>
                        {parseServerDate(art.created_at)?.toLocaleDateString("en-GB", {
                            day: "numeric",
                            month: "short",
                            year: "numeric",
                        }) ?? ""}
                    </span>
                    {art.updated_at && <span className={styles.edited}>(edited)</span>}
                </div>

                <div className={styles.artistLinks}>
                    <span className={styles.artistLink} onClick={() => navigate(`/user/${art.author.username}`)}>
                        More by <bdi>{art.author.display_name}</bdi>
                    </span>
                    {art.gallery_id && (
                        <span className={styles.artistLink} onClick={() => navigate(`/gallery/view/${art.gallery_id}`)}>
                            View gallery
                        </span>
                    )}
                </div>

                {art.tags.length > 0 && (
                    <div className={styles.tags}>
                        {art.tags.map(tag => (
                            <span
                                key={tag}
                                className={styles.tag}
                                onClick={() => navigate(`/gallery?tag=${encodeURIComponent(tag)}`)}
                            >
                                #{tag}
                            </span>
                        ))}
                    </div>
                )}

                <div className={styles.actions}>
                    <button
                        className={`${styles.likeBtn}${liked ? ` ${styles.likeBtnActive}` : ""}`}
                        onClick={handleLike}
                        disabled={!user}
                    >
                        &#9829; {likeCount}
                    </button>
                    <span className={styles.viewCount}>&#128065; {art.view_count}</span>
                    <div className={styles.spacer} />
                    {canEdit && !editing && (
                        <Button variant="secondary" size="small" onClick={startEdit}>
                            Edit
                        </Button>
                    )}
                    {canDelete && (
                        <Button variant="danger" size="small" onClick={() => setDeleteConfirmOpen(true)}>
                            Delete
                        </Button>
                    )}
                    {user && !isAuthor && <ReportButton targetType="art" targetId={art.id} />}
                    <ShareButton contentId={art.id} contentType="art" contentTitle={art.title} />
                </div>
            </div>

            {art.liked_by && art.liked_by.length > 0 && (
                <div className={styles.likedBy}>
                    <h3 className={styles.sectionTitle}>Liked by ({art.liked_by.length})</h3>
                    <div className={styles.likedByList}>
                        {art.liked_by.map(u => (
                            <ProfileLink key={u.id} user={u} size="small" />
                        ))}
                    </div>
                </div>
            )}

            <CommentsSection
                comments={art.comments}
                targetId={art.id}
                user={user}
                onChanged={() => refresh()}
                viewerBlocked={art.viewer_blocked}
                highlightedId={highlightedComment ?? undefined}
                linkPrefix="/gallery/art"
                reportType="art_comment"
                likeFn={likeFn}
                unlikeFn={unlikeFn}
                deleteFn={deleteFn}
                updateFn={updateFn}
                createCommentFn={createCommentFn}
                uploadMediaFn={uploadMediaFn}
            />

            <Modal isOpen={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)} title="Delete Art">
                <div style={{ padding: "1.25rem" }}>
                    <p style={{ marginBottom: "1rem" }}>
                        Are you sure you want to delete this art? This cannot be undone.
                    </p>
                    <div className={styles.confirmActions}>
                        <Button variant="secondary" onClick={() => setDeleteConfirmOpen(false)}>
                            Cancel
                        </Button>
                        <Button variant="danger" onClick={handleDelete}>
                            Delete Art
                        </Button>
                    </div>
                </div>
            </Modal>

            {lightboxOpen && (
                <div className={styles.lightbox} onClick={() => setLightboxOpen(false)}>
                    <img src={art.image_url} alt={art.title} className={styles.lightboxImage} />
                </div>
            )}
        </div>
    );
}
