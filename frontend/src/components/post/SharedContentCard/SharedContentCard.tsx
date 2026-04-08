import { Link } from "react-router";
import type { SharedContentPreview } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./SharedContentCard.module.css";

interface SharedContentCardProps {
    content: SharedContentPreview;
}

const typeLabels: Record<string, string> = {
    post: "Post",
    art: "Art",
    ship: "Ship",
    mystery: "Mystery",
    theory: "Theory",
    fanfic: "Fanfiction",
};

function truncate(text: string, maxLength: number): string {
    if (text.length <= maxLength) {
        return text;
    }
    return text.slice(0, maxLength) + "...";
}

function formatNumber(n: number): string {
    if (n >= 1000) {
        return (n / 1000).toFixed(1) + "k";
    }
    return String(n);
}

function SharedMediaGrid({ content }: { content: SharedContentPreview }) {
    const media = content.media;
    if (!media || media.length === 0) {
        return null;
    }
    const items = media.slice(0, 4);
    const count = items.length;
    return (
        <div
            className={`${styles.mediaGrid} ${count === 1 ? styles.mediaGrid1 : count === 2 ? styles.mediaGrid2 : styles.mediaGrid4}`}
        >
            {items.map((m, i) =>
                m.media_type.startsWith("video/") ? (
                    <video key={i} src={m.media_url} className={styles.mediaGridItem} muted />
                ) : (
                    <img key={i} src={m.thumbnail_url || m.media_url} alt="" className={styles.mediaGridItem} />
                ),
            )}
        </div>
    );
}

function PostContent({ content }: { content: SharedContentPreview }) {
    return (
        <>
            {content.author && <ProfileLink user={content.author} size="small" clickable={false} />}
            {content.body && <p className={styles.body}>{truncate(content.body, 200)}</p>}
            <SharedMediaGrid content={content} />
            <div className={styles.stats}>
                {content.like_count != null && content.like_count > 0 && (
                    <span>{formatNumber(content.like_count)} likes</span>
                )}
                {content.comment_count != null && content.comment_count > 0 && (
                    <span>{formatNumber(content.comment_count)} comments</span>
                )}
            </div>
        </>
    );
}

function ArtContent({ content }: { content: SharedContentPreview }) {
    return (
        <div className={styles.mediaRow}>
            {content.image_url && (
                <img src={content.image_url} alt={content.title ?? ""} className={styles.thumbnail} />
            )}
            <div className={styles.info}>
                {content.title && <span className={styles.title}>{content.title}</span>}
                {content.author && <ProfileLink user={content.author} size="small" clickable={false} />}
            </div>
        </div>
    );
}

function ShipContent({ content }: { content: SharedContentPreview }) {
    return (
        <div className={styles.mediaRow}>
            {content.image_url && (
                <img src={content.image_url} alt={content.title ?? ""} className={styles.thumbnail} />
            )}
            <div className={styles.info}>
                {content.title && <span className={styles.title}>{content.title}</span>}
                {content.author && <ProfileLink user={content.author} size="small" clickable={false} />}
                {content.vote_score != null && <span className={styles.badge}>Score: {content.vote_score}</span>}
            </div>
        </div>
    );
}

function MysteryContent({ content }: { content: SharedContentPreview }) {
    return (
        <div className={styles.info}>
            {content.title && <span className={styles.title}>{content.title}</span>}
            <div className={styles.badges}>
                {content.difficulty && <span className={styles.badge}>{content.difficulty}</span>}
                <span className={styles.badge}>{content.solved ? "Solved" : "Open"}</span>
            </div>
            {content.author && <ProfileLink user={content.author} size="small" clickable={false} />}
        </div>
    );
}

function TheoryContent({ content }: { content: SharedContentPreview }) {
    return (
        <div className={styles.info}>
            {content.title && <span className={styles.title}>{content.title}</span>}
            <div className={styles.badges}>
                {content.series && <span className={styles.badge}>{content.series}</span>}
                {content.credibility_score != null && (
                    <span className={styles.badge}>Credibility: {content.credibility_score}</span>
                )}
            </div>
            {content.author && <ProfileLink user={content.author} size="small" clickable={false} />}
        </div>
    );
}

function FanficContent({ content }: { content: SharedContentPreview }) {
    return (
        <div className={styles.mediaRow}>
            {content.image_url && (
                <img src={content.image_url} alt={content.title ?? ""} className={styles.thumbnail} />
            )}
            <div className={styles.info}>
                {content.title && <span className={styles.title}>{content.title}</span>}
                <div className={styles.badges}>
                    {content.series && <span className={styles.badge}>{content.series}</span>}
                    {content.rating && <span className={styles.badge}>{content.rating}</span>}
                    {content.word_count != null && (
                        <span className={styles.meta}>{formatNumber(content.word_count)} words</span>
                    )}
                </div>
                {content.author && <ProfileLink user={content.author} size="small" clickable={false} />}
            </div>
        </div>
    );
}

function ContentByType({ content }: { content: SharedContentPreview }) {
    switch (content.content_type) {
        case "post":
            return <PostContent content={content} />;
        case "art":
            return <ArtContent content={content} />;
        case "ship":
            return <ShipContent content={content} />;
        case "mystery":
            return <MysteryContent content={content} />;
        case "theory":
            return <TheoryContent content={content} />;
        case "fanfic":
            return <FanficContent content={content} />;
        default:
            return null;
    }
}

export function SharedContentCard({ content }: SharedContentCardProps) {
    if (content.deleted) {
        return (
            <div className={`${styles.card} ${styles.deleted}`}>
                <span className={styles.typeLabel}>{typeLabels[content.content_type] ?? content.content_type}</span>
                <p className={styles.deletedText}>This content is no longer available</p>
            </div>
        );
    }

    return (
        <Link to={content.url} className={styles.card}>
            <span className={styles.typeLabel}>{typeLabels[content.content_type] ?? content.content_type}</span>
            <ContentByType content={content} />
        </Link>
    );
}
