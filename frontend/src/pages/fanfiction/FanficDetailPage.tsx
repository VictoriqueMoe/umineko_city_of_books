import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { FanficDetail, PostComment } from "../../types/api";
import {
    createFanficComment,
    deleteFanfic,
    deleteFanficChapter,
    deleteFanficComment,
    favouriteFanfic,
    getFanfic,
    likeFanficComment,
    unfavouriteFanfic,
    unlikeFanficComment,
    updateFanficComment,
    uploadFanficCommentMedia,
} from "../../api/endpoints";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { CommentItem } from "../../components/post/CommentItem/CommentItem";
import { CommentComposer } from "../../components/post/CommentComposer/CommentComposer";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ShareButton } from "../../components/ShareButton/ShareButton";
import { relativeTime } from "../../utils/notifications";
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
    const [fanfic, setFanfic] = useState<FanficDetail | null>(null);
    const [loading, setLoading] = useState(true);
    const [lightboxOpen, setLightboxOpen] = useState(false);
    const [favouriting, setFavouriting] = useState(false);
    usePageTitle(fanfic?.title ?? "Fanfic");

    const fetchFanfic = useCallback(() => {
        if (!id) {
            return;
        }
        getFanfic(id)
            .then(data => {
                data.comments = data.comments ?? [];
                data.chapters = data.chapters ?? [];
                data.genres = data.genres ?? [];
                data.tags = data.tags ?? [];
                data.characters = data.characters ?? [];
                setFanfic(data);
            })
            .catch(() => setFanfic(null))
            .finally(() => setLoading(false));
    }, [id]);

    useEffect(() => {
        fetchFanfic();
    }, [fetchFanfic]);

    async function handleFavourite() {
        if (!fanfic || favouriting) {
            return;
        }
        setFavouriting(true);
        try {
            if (fanfic.user_favourited) {
                await unfavouriteFanfic(fanfic.id);
                setFanfic({
                    ...fanfic,
                    user_favourited: false,
                    favourite_count: fanfic.favourite_count - 1,
                });
            } else {
                await favouriteFanfic(fanfic.id);
                setFanfic({
                    ...fanfic,
                    user_favourited: true,
                    favourite_count: fanfic.favourite_count + 1,
                });
            }
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
        await deleteFanfic(fanfic.id);
        navigate("/fanfiction");
    }

    if (loading) {
        return <div className="loading">Loading...</div>;
    }

    if (!fanfic) {
        return <div className="empty-state">Fanfic not found.</div>;
    }

    const isAuthor = user?.id === fanfic.author.id;
    const canEdit = isAuthor || can(user?.role, "edit_any_post");
    const canDelete = isAuthor || can(user?.role, "delete_any_post");

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/fanfiction")}>
                &larr; All Fanfiction
            </span>

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
                            <h1 className={styles.detailTitle}>{fanfic.title}</h1>
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
                            <span>{relativeTime(fanfic.published_at)}</span>
                        </div>

                        <div className={styles.detailBadges}>
                            <span className={`${styles.detailBadge} ${ratingBadgeClass(fanfic.rating)}`}>
                                {fanfic.rating}
                            </span>
                            <span className={`${styles.detailBadge} ${statusBadgeClass(fanfic.status)}`}>
                                {fanfic.status}
                            </span>
                            <span className={`${styles.detailBadge} ${styles.detailBadgeSeries}`}>{fanfic.series}</span>
                            <span className={`${styles.detailBadge} ${styles.detailBadgeLang}`}>{fanfic.language}</span>
                            {fanfic.genres.map(g => (
                                <span key={g} className={`${styles.detailBadge} ${styles.badgeGenre}`}>
                                    {g}
                                </span>
                            ))}
                            {fanfic.tags.map(t => (
                                <span key={t} className={`${styles.detailBadge} ${styles.badgeTag}`}>
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
                                {c.character_name}
                            </span>
                        ))}
                    </div>
                )}

                {fanfic.summary && <p className={styles.summary}>{fanfic.summary}</p>}

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
                                            <span className={styles.tocItemTitle}>{ch.title}</span>
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
                                                        await deleteFanficChapter(ch.id);
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

            <div className={styles.commentsSection}>
                <h3 className={styles.commentsTitle}>
                    Comments {fanfic.comments.length > 0 && `(${fanfic.comments.length})`}
                </h3>
                {fanfic.comments.map(c => (
                    <CommentItem
                        key={c.id}
                        comment={c as unknown as PostComment}
                        postId={fanfic.id}
                        onDelete={fetchFanfic}
                        highlightedId={undefined}
                        linkPrefix="/fanfiction"
                        reportType="fanfic_comment"
                        likeFn={likeFanficComment}
                        unlikeFn={unlikeFanficComment}
                        deleteFn={deleteFanficComment}
                        updateFn={updateFanficComment}
                        createCommentFn={createFanficComment}
                        uploadMediaFn={uploadFanficCommentMedia}
                        viewerBlocked={fanfic.viewer_blocked}
                    />
                ))}
                {fanfic.comments.length === 0 && <p className="empty-state">No comments yet.</p>}
                {fanfic.viewer_blocked && <p className="empty-state">You cannot interact with this fanfic.</p>}
                {user && id && !fanfic.viewer_blocked && (
                    <CommentComposer
                        postId={id}
                        onCreated={fetchFanfic}
                        createCommentFn={createFanficComment}
                        uploadMediaFn={uploadFanficCommentMedia}
                    />
                )}
            </div>

            {lightboxOpen && fanfic.cover_image_url && (
                <Lightbox src={fanfic.cover_image_url} alt={fanfic.title} onClose={() => setLightboxOpen(false)} />
            )}
        </div>
    );
}
