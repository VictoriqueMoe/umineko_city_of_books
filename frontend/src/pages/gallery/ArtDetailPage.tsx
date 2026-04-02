import { useCallback, useEffect, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import type { ArtDetail } from "../../types/api";
import {
    createArtComment,
    deleteArt as apiDeleteArt,
    deleteArtComment,
    getArt,
    likeArt,
    likeArtComment,
    unlikeArt,
    unlikeArtComment,
    updateArt as apiUpdateArt,
    uploadArtCommentMedia,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import { linkify } from "../../utils/linkify";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Button } from "../../components/Button/Button";
import { Modal } from "../../components/Modal/Modal";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { MentionTextArea } from "../../components/MentionTextArea/MentionTextArea";
import { TagInput } from "../../components/art/TagInput/TagInput";
import styles from "./ArtDetailPage.module.css";

export function ArtDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const [art, setArt] = useState<ArtDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const [liked, setLiked] = useState(false);
    const [likeCount, setLikeCount] = useState(0);
    const [editing, setEditing] = useState(false);
    const [editTitle, setEditTitle] = useState("");
    const [editDesc, setEditDesc] = useState("");
    const [editTags, setEditTags] = useState<string[]>([]);
    const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
    const [lightboxOpen, setLightboxOpen] = useState(false);

    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;

    const fetchArt = useCallback(() => {
        if (!id) {
            return;
        }
        getArt(id)
            .then(data => {
                setArt(data);
                setLiked(data.user_liked);
                setLikeCount(data.like_count);
            })
            .catch(() => setArt(null))
            .finally(() => setLoading(false));
    }, [id]);

    useEffect(() => {
        fetchArt();
    }, [fetchArt]);

    useEffect(() => {
        if (!art || loading || !highlightedComment) {
            return;
        }
        requestAnimationFrame(() => {
            const el = document.getElementById(`comment-${highlightedComment}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
            }
        });
    }, [art, loading, highlightedComment]);

    async function handleLike() {
        if (!id) {
            return;
        }
        if (liked) {
            setLiked(false);
            setLikeCount(c => c - 1);
            await unlikeArt(id).catch(() => {
                setLiked(true);
                setLikeCount(c => c + 1);
            });
        } else {
            setLiked(true);
            setLikeCount(c => c + 1);
            await likeArt(id).catch(() => {
                setLiked(false);
                setLikeCount(c => c - 1);
            });
        }
    }

    async function handleDelete() {
        if (!id) {
            return;
        }
        await apiDeleteArt(id);
        navigate(-1);
    }

    function startEdit() {
        if (!art) {
            return;
        }
        setEditTitle(art.title);
        setEditDesc(art.description);
        setEditTags([...art.tags]);
        setEditing(true);
    }

    async function saveEdit() {
        if (!id || !editTitle.trim()) {
            return;
        }
        await apiUpdateArt(id, {
            title: editTitle.trim(),
            description: editDesc.trim(),
            tags: editTags,
        });
        setEditing(false);
        fetchArt();
    }

    if (loading) {
        return <div className="loading">Loading art...</div>;
    }

    if (!art) {
        return <div className="empty-state">Art not found.</div>;
    }

    const isAuthor = user && user.id === art.author.id;
    const canDelete = isAuthor || can(user?.role, "delete_any_post");

    function flattenComments(comments: ArtDetail["comments"]): ArtDetail["comments"] {
        const result: ArtDetail["comments"] = [];
        for (const c of comments) {
            result.push(c);
            if (c.replies) {
                result.push(...flattenComments(c.replies));
            }
        }
        return result;
    }

    const allComments = flattenComments(art.comments);

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate(-1)}>
                &larr; Back to Gallery
            </span>

            <div className={styles.imageSection}>
                <img
                    src={art.image_url}
                    alt={art.title}
                    className={styles.fullImage}
                    onClick={() => setLightboxOpen(true)}
                />
            </div>

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
                        {art.description && <div className={styles.description}>{linkify(art.description)}</div>}
                    </>
                )}

                <div className={styles.metaRow}>
                    <ProfileLink user={art.author} size="medium" />
                    <span className={styles.date}>
                        {new Date(art.created_at).toLocaleDateString("en-GB", {
                            day: "numeric",
                            month: "short",
                            year: "numeric",
                        })}
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
                    {isAuthor && !editing && (
                        <Button variant="secondary" size="small" onClick={startEdit}>
                            Edit
                        </Button>
                    )}
                    {canDelete && (
                        <Button variant="danger" size="small" onClick={() => setDeleteConfirmOpen(true)}>
                            Delete
                        </Button>
                    )}
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
                <h3 className={styles.sectionTitle}>Comments {allComments.length > 0 && `(${allComments.length})`}</h3>
                {allComments.map(c => (
                    <div
                        key={c.id}
                        id={`comment-${c.id}`}
                        className={`${styles.comment}${c.id === highlightedComment ? ` ${styles.commentHighlighted}` : ""}`}
                    >
                        <div className={styles.commentHeader}>
                            <ProfileLink user={c.author} size="small" />
                            <span className={styles.commentDate}>
                                {new Date(c.created_at).toLocaleDateString("en-GB")}
                            </span>
                            {c.updated_at && <span className={styles.edited}>(edited)</span>}
                        </div>
                        <div className={styles.commentBody}>{linkify(c.body)}</div>
                        <div className={styles.commentActions}>
                            <button
                                className={`${styles.commentLikeBtn}${c.user_liked ? ` ${styles.likeBtnActive}` : ""}`}
                                onClick={async () => {
                                    if (c.user_liked) {
                                        await unlikeArtComment(c.id);
                                    } else {
                                        await likeArtComment(c.id);
                                    }
                                    fetchArt();
                                }}
                                disabled={!user}
                            >
                                &#9829; {c.like_count}
                            </button>
                            {user && (
                                <button
                                    className={styles.replyBtn}
                                    onClick={() => {
                                        const composer = document.getElementById("art-comment-composer");
                                        if (composer) {
                                            composer.scrollIntoView({ behavior: "smooth" });
                                        }
                                    }}
                                >
                                    Reply
                                </button>
                            )}
                            {user && user.id === c.author.id && (
                                <button
                                    className={styles.deleteBtn}
                                    onClick={async () => {
                                        if (window.confirm("Delete this comment?")) {
                                            await deleteArtComment(c.id);
                                            fetchArt();
                                        }
                                    }}
                                >
                                    Delete
                                </button>
                            )}
                        </div>
                    </div>
                ))}
                {allComments.length === 0 && <p className={styles.noComments}>No comments yet.</p>}
                {user && id && (
                    <div id="art-comment-composer">
                        <CommentComposer
                            postId={id}
                            onCreated={fetchArt}
                            createCommentFn={createArtComment}
                            uploadMediaFn={uploadArtCommentMedia}
                        />
                    </div>
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
