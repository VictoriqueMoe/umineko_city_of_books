import { AudioAttachment } from "../../AudioAttachment/AudioAttachment";
import { SpoilerCover } from "../../SpoilerImage/SpoilerCover";
import { spoilerBlurClass } from "../../SpoilerImage/spoilerBlur";
import { SpoilerImage } from "../../SpoilerImage/SpoilerImage";
import type { PostMedia } from "../../../types/api";
import styles from "./ChatMessageMedia.module.css";

interface ChatMessageMediaProps {
    media: PostMedia;
    itemClassName: string;
    videoClassName?: string;
    onLightbox?: (src: string) => void;
}

export function ChatMessageMedia({ media, itemClassName, videoClassName, onLightbox }: ChatMessageMediaProps) {
    const isSpoiler = media.is_spoiler ?? false;

    if (media.media_type === "audio") {
        if (!isSpoiler) {
            return <AudioAttachment src={media.media_url} filename={media.filename} />;
        }

        return (
            <SpoilerCover isSpoiler contentKey={media.media_url} className={styles.audioCover}>
                {covered =>
                    covered ? (
                        <div className={styles.audioPlaceholder} />
                    ) : (
                        <AudioAttachment src={media.media_url} filename={media.filename} />
                    )
                }
            </SpoilerCover>
        );
    }

    if (media.media_type === "video") {
        const videoClass = videoClassName ? `${itemClassName} ${videoClassName}` : itemClassName;

        if (!isSpoiler) {
            return (
                <video
                    className={videoClass}
                    src={media.media_url}
                    controls
                    poster={media.thumbnail_url || undefined}
                />
            );
        }

        return (
            <SpoilerCover isSpoiler contentKey={media.media_url} className={styles.mediaCover}>
                {covered => (
                    <video
                        className={`${videoClass} ${spoilerBlurClass(covered)}`}
                        src={media.media_url}
                        controls={!covered}
                        poster={media.thumbnail_url || undefined}
                    />
                )}
            </SpoilerCover>
        );
    }

    if (!isSpoiler) {
        return (
            <img
                className={itemClassName}
                src={media.media_url}
                alt=""
                width={media.width || undefined}
                height={media.height || undefined}
                loading="lazy"
                decoding="async"
                onClick={() => onLightbox?.(media.media_url)}
            />
        );
    }

    return (
        <SpoilerImage
            src={media.media_url}
            isSpoiler
            className={styles.mediaCover}
            imageClassName={itemClassName}
            width={media.width || undefined}
            height={media.height || undefined}
            loading="lazy"
            onClick={() => onLightbox?.(media.media_url)}
        />
    );
}
