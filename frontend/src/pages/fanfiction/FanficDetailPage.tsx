import { useCallback, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useScrollToHash } from "../../hooks/useScrollToHash";
import { useFanfic } from "../../hooks/queries/fanfic";
import {
    useDeleteFanfic,
    useDeleteFanficChapter,
    useFavouriteFanfic,
    useUnfavouriteFanfic,
} from "../../hooks/mutations/fanfic";
import { useAuth } from "../../hooks/useAuth";
import { useCommentHandlers } from "../../hooks/useCommentHandlers";
import { contentPermissions } from "../../domain/contentPermissions";
import { errorMessage } from "../../utils/errorMessage";
import { Button } from "../../components/Button/Button";
import { ErrorBanner } from "../../components/ErrorBanner/ErrorBanner";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentsSection } from "../../components/post/CommentsSection/CommentsSection";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ShareButton } from "../../components/ShareButton/ShareButton";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import { renderRich } from "../../components/richText/richText";
import styles from "./FanficPages.module.css";

function ratingBadgeClass(rating: string): string {
    if (rating === "K") {
        return `${styles.badge} ${styles.badgeRatingK}`;
    }
    if (rating === "K+") {
        return `${styles.badge} ${styles.badgeRatingKPlus}`;
    }
    if (rating === "T") {
        return `${styles.badge} ${styles.badgeRatingT}`;
    }
    if (rating === "M") {
        return `${styles.badge} ${styles.badgeRatingM}`;
    }
    return styles.badge;
}

function statusBadgeClass(status: string): string {
    if (status === "Complete") {
        return `${styles.badge} ${styles.badgeComplete}`;
    }
    return `${styles.badge} ${styles.badgeStatus}`;
}

function formatNumber(n: number): string {
    if (n >= 1_000_000) {
        return (n / 1_000_000).toFixed(1) + "M";
    }
    if (n >= 1_000) {
        return (n / 1_000).toFixed(1) + "K";
    }
    return n.toLocaleString();
}

export function FanficDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const fanficId = id ?? "";
    const { hash } = useLocation();
    const highlightedComment = hash.startsWith("#comment-") ? hash.replace("#comment-", "") : null;
    const { fanfic, loading, refresh } = useFanfic(fanficId);
    useScrollToHash(!loading && !!fanfic, highlightedComment ? `comment-${highlightedComment}` : null);
    const [lightboxOpen, setLightboxOpen] = useState(false);
    const [favouriting, setFavouriting] = useState(false);
    const [actionError, setActionError] = useState("");
    usePageTitle(fanfic?.title ?? "Fanfic");

    const fetchFanfic = useCallback(() => {
        refresh();
    }, [refresh]);

    const favouriteMutation = useFavouriteFanfic();
    const unfavouriteMutation = useUnfavouriteFanfic();
    const deleteFanficMutation = useDeleteFanfic();
    const deleteChapterMutation = useDeleteFanficChapter(fanficId);
    const { createCommentFn, updateFn, deleteFn, likeFn, unlikeFn, uploadMediaFn } = useCommentHandlers(
        "fanfic",
        fanficId,
        { enabled: ["create", "update", "delete", "like", "unlike", "uploadMedia"] },
    );

    async function handleFavourite() {
        if (!fanfic || favouriting) {
            return;
        }
        setFavouriting(true);
        try {
            if (fanfic.user_favourited) {
                await unfavouriteMutation.mutateAsync(fanfic.id);
            } else {
                await favouriteMutation.mutateAsync(fanfic.id);
            }
            fetchFanfic();
        } catch {
            // ignore
        } finally {
            setFavouriting(false);
        }
    }

    async function handleDelete() {
        if (!fanfic || !window.confirm("Delete this fanfic? This cannot be undone.")) {
            return;
        }
        setActionError("");
        try {
            await deleteFanficMutation.mutateAsync(fanfic.id);
            navigate("/fanfiction");
        } catch (thrown) {
            setActionError(errorMessage(thrown, "Failed to delete fanfic"));
        }
    }

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    if (!fanfic) {
        return <div className="empty-state">Fanfic not found.</div>;
    }

    const { canEdit, canDelete } = contentPermissions(user, { family: "fanfic", authorId: fanfic.author.id });

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/fanfiction")}>
                &larr; All Fanfiction
            </span>

            {actionError && <ErrorBanner message={actionError} />}

            <div className={styles.detail}>
                <div className={styles.detailHeader}>
                    {fanfic.cover_image_url && (
                        <img
                            className={styles.detailCover}
                            src={fanfic.cover_image_url}
                            alt={fanfic.title}
                            onClick={() => setLightboxOpen(true)}
                        />
                    )}
                    <div className={styles.detailHeaderInfo}>
                        <div className={styles.detailTitleRow}>
                            <h1 dir="auto" className={styles.detailTitle}>
                                {fanfic.title}
                            </h1>
                            {user && (
                                <button
                                    className={`${styles.favouriteBtn}${fanfic.user_favourited ? ` ${styles.favouriteBtnActive}` : ""}`}
                                    onClick={handleFavourite}
                                    disabled={favouriting}
                                >
                                    {fanfic.user_favourited ? "\u2665" : "\u2661"} {fanfic.favourite_count}
                                </button>
                            )}
                            {(canEdit || canDelete) && (
                                <div style={{ display: "flex", gap: "0.5rem" }}>
                                    {canEdit && (
                                        <Button
                                            variant="secondary"
                                            size="small"
                                            onClick={() => navigate(`/fanfiction/${fanfic.id}/edit`)}
                                        >
                                            Edit
                                        </Button>
                                    )}
                                    {canEdit && !fanfic.is_oneshot && (
                                        <Button
                                            variant="secondary"
                                            size="small"
                                            onClick={() => navigate(`/fanfiction/${fanfic.id}/chapter/new`)}
                                        >
                                            Add Chapter
                                        </Button>
                                    )}
                                    {canDelete && (
                                        <Button variant="danger" size="small" onClick={handleDelete}>
                                            Delete
                                        </Button>
                                    )}
                                </div>
                            )}
                            <ShareButton contentId={fanfic.id} contentType="fanfic" contentTitle={fanfic.title} />
                        </div>

                        <div className={styles.detailByline}>
                            <ProfileLink user={fanfic.author} size="small" />
                            <RelativeTimestamp value={fanfic.published_at} />
                        </div>

                        <div className={styles.detailBadges}>
                            <span className={`${styles.detailBadge} ${ratingBadgeClass(fanfic.rating)}`}>
                                {fanfic.rating}
                            </span>
                            <span className={`${styles.detailBadge} ${statusBadgeClass(fanfic.status)}`}>
                                {fanfic.status}
                            </span>
                            <span dir="auto" className={`${styles.detailBadge} ${styles.detailBadgeSeries}`}>
                                {fanfic.series}
                            </span>
                            <span dir="auto" className={`${styles.detailBadge} ${styles.detailBadgeLang}`}>
                                {fanfic.language}
                            </span>
                            {fanfic.genres.map(g => (
                                <span key={g} className={`${styles.detailBadge} ${styles.badgeGenre}`}>
                                    {g}
                                </span>
                            ))}
                            {fanfic.tags.map(t => (
                                <span key={t} dir="auto" className={`${styles.detailBadge} ${styles.badgeTag}`}>
                                    {t}
                                </span>
                            ))}
                            {fanfic.is_pairing && (
                                <span className={`${styles.detailBadge} ${styles.badgePairing}`}>Pairing</span>
                            )}
                            {fanfic.contains_lemons && (
                                <span className={`${styles.detailBadge} ${styles.badgeLemons}`}>Contains Lemons</span>
                            )}
                        </div>

                        <div className={styles.detailStats}>
                            <div className={styles.detailStat}>
                                <span className={styles.detailStatValue}>{formatNumber(fanfic.word_count)}</span>
                                <span className={styles.detailStatLabel}>Words</span>
                            </div>
                            <div className={styles.detailStat}>
                                <span className={styles.detailStatValue}>{fanfic.chapter_count}</span>
                                <span className={styles.detailStatLabel}>
                                    {fanfic.chapter_count === 1 ? "Chapter" : "Chapters"}
                                </span>
                            </div>
                            <div className={styles.detailStat}>
                                <span className={styles.detailStatValue}>{formatNumber(fanfic.favourite_count)}</span>
                                <span className={styles.detailStatLabel}>Favourites</span>
                            </div>
                            <div className={styles.detailStat}>
                                <span className={styles.detailStatValue}>{formatNumber(fanfic.view_count)}</span>
                                <span className={styles.detailStatLabel}>Views</span>
                            </div>
                        </div>
                    </div>
                </div>

                {fanfic.characters.length > 0 && (
                    <div className={styles.detailCharacters}>
                        {fanfic.characters.map((c, i) => (
                            <span
                                key={`${c.series}-${c.character_id ?? c.character_name}-${i}`}
                                className={styles.charPill}
                            >
                                <span dir="auto">{c.character_name}</span>
                            </span>
                        ))}
                    </div>
                )}

                {fanfic.summary && (
                    <div dir="auto" className={styles.summary}>
                        {renderRich(fanfic.summary)}
                    </div>
                )}

                <div className={styles.tocSection}>
                    {fanfic.is_oneshot ? (
                        <Button variant="primary" onClick={() => navigate(`/fanfiction/${fanfic.id}/chapter/1`)}>
                            Read Story
                        </Button>
                    ) : (
                        <>
                            {fanfic.reading_progress > 0 && fanfic.reading_progress <= fanfic.chapters.length && (
                                <div style={{ marginBottom: "0.75rem" }}>
                                    <Button
                                        variant="primary"
                                        onClick={() =>
                                            navigate(`/fanfiction/${fanfic.id}/chapter/${fanfic.reading_progress}`)
                                        }
                                    >
                                        Continue from Chapter {fanfic.reading_progress}
                                    </Button>
                                </div>
                            )}
                            <h3 className={styles.tocTitle}>Chapters ({fanfic.chapters.length})</h3>
                            <ul className={styles.tocList}>
                                {fanfic.chapters.map(ch => (
                                    <li key={ch.chapter_number} className={styles.tocItem}>
                                        <span
                                            className={styles.tocItemLink}
                                            onClick={() =>
                                                navigate(`/fanfiction/${fanfic.id}/chapter/${ch.chapter_number}`)
                                            }
                                        >
                                            <span className={styles.tocItemNum}>{ch.chapter_number}.</span>
                                            <span dir="auto" className={styles.tocItemTitle}>
                                                {ch.title}
                                            </span>
                                            <span className={styles.tocItemWords}>
                                                {formatNumber(ch.word_count)} words
                                            </span>
                                        </span>
                                        {canEdit && (
                                            <div style={{ display: "flex", gap: "0.25rem" }}>
                                                <Button
                                                    variant="ghost"
                                                    size="small"
                                                    onClick={() =>
                                                        navigate(
                                                            `/fanfiction/${fanfic.id}/chapter/${ch.chapter_number}/edit`,
                                                        )
                                                    }
                                                >
                                                    Edit
                                                </Button>
                                                <Button
                                                    variant="ghost"
                                                    size="small"
                                                    onClick={async () => {
                                                        if (
                                                            !window.confirm(
                                                                `Delete chapter ${ch.chapter_number}? This cannot be undone.`,
                                                            )
                                                        ) {
                                                            return;
                                                        }
                                                        await deleteChapterMutation.mutateAsync(ch.id);
                                                        fetchFanfic();
                                                    }}
                                                >
                                                    Delete
                                                </Button>
                                            </div>
                                        )}
                                    </li>
                                ))}
                            </ul>
                        </>
                    )}
                </div>
            </div>

            <CommentsSection
                comments={fanfic.comments ?? []}
                targetId={fanfic.id}
                user={user}
                onChanged={fetchFanfic}
                blockedText="You cannot interact with this fanfic."
                viewerBlocked={fanfic.viewer_blocked}
                linkPrefix="/fanfiction"
                reportType="fanfic_comment"
                highlightedId={highlightedComment ?? undefined}
                likeFn={likeFn}
                unlikeFn={unlikeFn}
                deleteFn={deleteFn}
                updateFn={updateFn}
                createCommentFn={createCommentFn}
                uploadMediaFn={uploadMediaFn}
            />

            {lightboxOpen && fanfic.cover_image_url && (
                <Lightbox src={fanfic.cover_image_url} alt={fanfic.title} onClose={() => setLightboxOpen(false)} />
            )}
        </div>
    );
}
