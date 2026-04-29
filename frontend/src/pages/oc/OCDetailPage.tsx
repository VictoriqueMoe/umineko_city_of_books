import { useState } from "react";
import { Link, useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { useOC } from "../../api/queries/oc";
import {
    useCreateOCComment,
    useDeleteOCComment,
    useFavouriteOC,
    useLikeOCComment,
    useUnlikeOCComment,
    useUpdateOCComment,
    useUploadOCCommentMedia,
    useVoteOC,
} from "../../api/mutations/oc";
import { useAuth } from "../../hooks/useAuth";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { renderRich } from "../../utils/richText";
import type { PostComment } from "../../types/api";
import shipStyles from "../ships/ShipPages.module.css";

export function OCDetailPage() {
    const { id } = useParams<{ id: string }>();
    const location = useLocation();
    const { oc, loading, refresh } = useOC(id ?? "");
    const { user: currentUser } = useAuth();
    const navigate = useNavigate();
    usePageTitle(oc?.name ?? "OC");

    const voteMutation = useVoteOC(id ?? "");
    const favouriteMutation = useFavouriteOC();
    const createComment = useCreateOCComment(id ?? "");
    const updateComment = useUpdateOCComment();
    const deleteComment = useDeleteOCComment();
    const likeComment = useLikeOCComment();
    const unlikeComment = useUnlikeOCComment();
    const uploadCommentMedia = useUploadOCCommentMedia();
    const [error, setError] = useState("");
    const [lightbox, setLightbox] = useState<{ src: string; alt: string } | null>(null);
    const hash = location.hash;
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;
    useScrollToHash(!loading && !!oc, highlightedComment ? `comment-${highlightedComment}` : null);

    if (loading) {
        return (
            <div className={shipStyles.page}>
                <div className="loading">Loading OC...</div>
            </div>
        );
    }
    if (!oc) {
        return (
            <div className={shipStyles.page}>
                <div className="empty-state">OC not found.</div>
            </div>
        );
    }

    const seriesLabel = oc.series === "custom" ? (oc.custom_series_name ?? "Custom") : oc.series;
    const isOwner = currentUser?.id === oc.author.id;

    async function handleVote(value: number) {
        try {
            await voteMutation.mutateAsync(oc!.user_vote === value ? 0 : value);
            await refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to vote");
        }
    }

    async function handleFavourite() {
        try {
            await favouriteMutation.mutateAsync(oc!.id);
            await refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to favourite");
        }
    }

    return (
        <div className={shipStyles.page}>
            {error && <ErrorBanner message={error} />}

            <div className={shipStyles.detailHeader}>
                {oc.image_url && (
                    <img
                        className={shipStyles.detailImage}
                        src={oc.image_url}
                        alt={oc.name}
                        style={{ cursor: "zoom-in" }}
                        onClick={() => setLightbox({ src: oc.image_url ?? "", alt: oc.name })}
                    />
                )}
                <div className={shipStyles.detailBody}>
                    <h1 className={shipStyles.detailTitle}>{oc.name}</h1>
                    <div className={shipStyles.detailMeta}>
                        <ProfileLink user={oc.author} size="small" />
                        <span className={`${shipStyles.characterPill} ${shipStyles.characterPillOc}`}>
                            {seriesLabel}
                        </span>
                    </div>
                    {oc.description && <div className={shipStyles.detailDescription}>{renderRich(oc.description)}</div>}
                    <div className={shipStyles.voteRow}>
                        <Button variant="ghost" size="small" onClick={() => handleVote(1)} disabled={!currentUser}>
                            {oc.user_vote === 1 ? "▲" : "△"}
                        </Button>
                        <span className={shipStyles.voteScore}>
                            {oc.vote_score > 0 ? "+" : ""}
                            {oc.vote_score}
                        </span>
                        <Button variant="ghost" size="small" onClick={() => handleVote(-1)} disabled={!currentUser}>
                            {oc.user_vote === -1 ? "▼" : "▽"}
                        </Button>
                        <Button
                            variant={oc.user_favourited ? "primary" : "ghost"}
                            size="small"
                            onClick={handleFavourite}
                            disabled={!currentUser}
                        >
                            {oc.user_favourited ? "♥" : "♡"} {oc.favourite_count}
                        </Button>
                        {isOwner && (
                            <Button variant="secondary" size="small" onClick={() => navigate(`/oc/${oc.id}/edit`)}>
                                Edit
                            </Button>
                        )}
                    </div>
                </div>
            </div>

            {oc.gallery.length > 0 && (
                <div style={{ marginTop: "1.5rem" }}>
                    <h2>Gallery</h2>
                    <div
                        style={{
                            display: "grid",
                            gridTemplateColumns: "repeat(auto-fill, minmax(180px, 1fr))",
                            gap: "0.75rem",
                        }}
                    >
                        {oc.gallery.map(img => (
                            <figure key={img.id} style={{ margin: 0 }}>
                                <img
                                    src={img.thumbnail_url || img.image_url}
                                    alt={img.caption ?? ""}
                                    style={{ width: "100%", borderRadius: "6px", cursor: "zoom-in" }}
                                    onClick={() => setLightbox({ src: img.image_url, alt: img.caption ?? "" })}
                                />
                                {img.caption && <figcaption style={{ fontSize: "0.85rem" }}>{img.caption}</figcaption>}
                            </figure>
                        ))}
                    </div>
                </div>
            )}

            <CommentsSection
                comments={(oc.comments ?? []) as unknown as PostComment[]}
                targetId={oc.id}
                user={currentUser}
                onChanged={refresh}
                blockedText="You cannot interact with this OC."
                viewerBlocked={oc.viewer_blocked}
                highlightedId={highlightedComment ?? undefined}
                linkPrefix="/oc"
                reportType="oc_comment"
                likeFn={commentId => likeComment.mutateAsync(commentId).then(() => {})}
                unlikeFn={commentId => unlikeComment.mutateAsync(commentId).then(() => {})}
                deleteFn={commentId => deleteComment.mutateAsync(commentId).then(() => {})}
                updateFn={(commentId, body) => updateComment.mutateAsync({ id: commentId, body }).then(() => {})}
                createCommentFn={(_ocId, body, parentId) => createComment.mutateAsync({ body, parentId })}
                uploadMediaFn={(commentId, file) => uploadCommentMedia.mutateAsync({ commentId, file })}
            />

            <div style={{ marginTop: "1rem" }}>
                <Link to="/oc">← Back to all OCs</Link>
            </div>

            {lightbox && <Lightbox src={lightbox.src} alt={lightbox.alt} onClose={() => setLightbox(null)} />}
        </div>
    );
}
