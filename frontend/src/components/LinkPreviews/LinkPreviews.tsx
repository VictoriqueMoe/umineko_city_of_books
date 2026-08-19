import { useMemo, useState } from "react";
import { useLinkPreview } from "../../api/queries/linkPreview";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { extractYouTubeIDs } from "../../utils/youtube";
import { YouTubeEmbed } from "../chat/YouTubeEmbed/YouTubeEmbed";
import { Lightbox } from "../Lightbox/Lightbox";
import type { LinkPreview } from "../../types/api";
import { previewableURLs } from "./urls";
import styles from "./LinkPreviews.module.css";

interface LinkPreviewsProps {
    body: string;
    authorCreatedAt?: string;
}

export function LinkPreviews({ body, authorCreatedAt }: LinkPreviewsProps) {
    const { new_account_hours } = useSiteInfo();
    const urls = useMemo(() => previewableURLs(body), [body]);

    if (urls.length === 0 || isNewAccount(authorCreatedAt, new_account_hours)) {
        return null;
    }

    return (
        <div className={styles.previews}>
            {urls.map(url => (
                <LinkPreviewItem key={url} url={url} />
            ))}
        </div>
    );
}

function isNewAccount(createdAt: string | undefined, hours: number): boolean {
    if (!createdAt || hours <= 0) {
        return false;
    }

    const created = new Date(createdAt).getTime();
    if (Number.isNaN(created)) {
        return false;
    }

    return Date.now() - created < hours * 60 * 60 * 1000;
}

function LinkPreviewItem({ url }: { url: string }) {
    const localIds = extractYouTubeIDs(url, 1);

    if (localIds.length > 0) {
        return <YouTubeEmbed videoIds={localIds} />;
    }

    return <RemotePreview url={url} />;
}

function RemotePreview({ url }: { url: string }) {
    const { preview } = useLinkPreview(url);

    if (!preview) {
        return null;
    }

    if (preview.type === "youtube") {
        if (!preview.video_id) {
            return null;
        }
        return <YouTubeEmbed videoIds={[preview.video_id]} />;
    }

    if (preview.type === "image") {
        return <ImagePreview url={preview.url} />;
    }

    if (preview.type === "video") {
        return <video className={styles.video} src={preview.url} controls preload="metadata" />;
    }

    if (preview.type === "link") {
        return <LinkCard preview={preview} />;
    }

    return null;
}

function ImagePreview({ url }: { url: string }) {
    const [lightboxOpen, setLightboxOpen] = useState(false);

    return (
        <>
            <img
                className={styles.image}
                src={url}
                alt=""
                loading="lazy"
                decoding="async"
                onClick={() => setLightboxOpen(true)}
            />
            {lightboxOpen && <Lightbox src={url} onClose={() => setLightboxOpen(false)} />}
        </>
    );
}

function LinkCard({ preview }: { preview: LinkPreview }) {
    if (!preview.title && !preview.description && !preview.image) {
        return null;
    }

    return (
        <a href={preview.url} target="_blank" rel="noopener noreferrer" className={styles.linkCard}>
            <div className={styles.linkBody}>
                {preview.site_name && <span className={styles.linkSite}>{preview.site_name}</span>}
                {preview.title && <span className={styles.linkTitle}>{preview.title}</span>}
                {preview.description && <span className={styles.linkDesc}>{preview.description}</span>}
            </div>
            {preview.image && (
                <div className={styles.linkImageWrap}>
                    <img src={preview.image} alt="" className={styles.linkImage} loading="lazy" />
                </div>
            )}
        </a>
    );
}
