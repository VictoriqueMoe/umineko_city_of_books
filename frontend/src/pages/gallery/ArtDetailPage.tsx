import { useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useQueryClient } from "@tanstack/react-query";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import type { ArtDetail } from "../../types/api";
import { useArt } from "../../api/queries/art";
import { queryKeys } from "../../api/queryKeys";
import {
    useCreateArtComment,
    useDeleteArt,
    useDeleteArtComment,
    useLikeArt,
    useLikeArtComment,
    useUnlikeArt,
    useUnlikeArtComment,
    useUpdateArt,
    useUpdateArtComment,
    useUploadArtCommentMedia,
} from "../../api/mutations/art";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import { renderRich } from "../../utils/richText";
import { parseServerDate } from "../../utils/time";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Button } from "../../components/Button/Button";
import { Modal } from "../../components/Modal/Modal";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
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
    const qc = useQueryClient();
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

    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const likeArtMutation = useLikeArt();
    const unlikeArtMutation = useUnlikeArt();
    const deleteArtMutation = useDeleteArt();
    const updateArtMutation = useUpdateArt(id ?? "");
    const createCommentMutation = useCreateArtComment(id ?? "");
    const updateCommentMutation = useUpdateArtComment(id ?? "");
    const deleteCommentMutation = useDeleteArtComment(id ?? "");
    const likeCommentMutation = useLikeArtComment(id ?? "");
    const unlikeCommentMutation = useUnlikeArtComment(id ?? "");
    const uploadMediaMutation = useUploadArtCommentMedia(id ?? "");

    useScrollToHash(!loading && !!art, highlightedComment ? `comment-${highlightedComment}` : null);

    function applyLikeOverlay(delta: number, nextLiked: boolean) {
        if (!id) {
            return;
        }
        qc.setQueryData<ArtDetail>(queryKeys.art.detail(id), prev =>
            prev ? { ...prev, user_liked: nextLiked, like_count: prev.like_count + delta } : prev,
        );
    }

    async function handleLike() {
        if (!id) {
            return;
        }
        if (liked) {
            applyLikeOverlay(-1, false);
            await unlikeArtMutation.mutateAsync(id).catch(() => applyLikeOverlay(1, true));
        } else {
            applyLikeOverlay(1, true);
            await likeArtMutation.mutateAsync(id).catch(() => applyLikeOverlay(-1, false));
        }
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

    const isAuthor = user && user.id === art.author.id;
    const canEdit = isAuthor || can(user?.role, "edit_any_post");
    const canDelete = isAuthor || can(user?.role, "delete_any_post");

    const likeCommentFn = (commentId: string) => likeCommentMutation.mutateAsync(commentId);
    const unlikeCommentFn = (commentId: string) => unlikeCommentMutation.mutateAsync(commentId);
    const deleteCommentFn = (commentId: string) => deleteCommentMutation.mutateAsync(commentId);
    const updateCommentFn = (commentId: string, body: string) =>
        updateCommentMutation.mutateAsync({ commentId, body }).then(() => undefined);
    const createCommentFn = (_postId: string, body: string, parentId?: string) =>
        createCommentMutation.mutateAsync({ body, parentId });
    const uploadMediaFn = (commentId: string, file: File) => uploadMediaMutation.mutateAsync({ commentId, file });

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
                            <Button variant="primary" size="small" onClick={saveEdit}>
                                Save
                            </Button>
                        </div>
                    </div>
                ) : (
                    <>
                        <h1 className={styles.title}>{art.title}</h1>
                        {art.description && <div className={styles.description}>{renderRich(art.description)}</div>}
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
                        More by {art.author.display_name}
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

            <div className={styles.comments}>
                <h3 className={styles.sectionTitle}>
                    Comments {art.comments.length > 0 && `(${art.comments.length})`}
                </h3>
                {art.comments.map(c => (
                    <CommentItem
                        key={c.id}
                        comment={c}
                        postId={art.id}
                        onDelete={() => refresh()}
                        highlightedId={highlightedComment ?? undefined}
                        linkPrefix="/gallery/art"
                        reportType="art_comment"
                        likeFn={likeCommentFn}
                        unlikeFn={unlikeCommentFn}
                        deleteFn={deleteCommentFn}
                        updateFn={updateCommentFn}
                        createCommentFn={createCommentFn}
                        uploadMediaFn={uploadMediaFn}
                        viewerBlocked={art.viewer_blocked}
                    />
                ))}
                {art.comments.length === 0 && <p className={styles.noComments}>No comments yet.</p>}
                {art.viewer_blocked && <p className={styles.blockedNotice}>You cannot interact with this post.</p>}
                {user && id && !art.viewer_blocked && (
                    <CommentComposer
                        postId={id}
                        onCreated={() => refresh()}
                        createCommentFn={createCommentFn}
                        uploadMediaFn={uploadMediaFn}
                    />
                )}
            </div>

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
