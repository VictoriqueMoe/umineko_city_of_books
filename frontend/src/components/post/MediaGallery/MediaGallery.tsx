import { useState } from "react";
import type { PostMedia } from "../../../types/api";
import { Lightbox } from "../../Lightbox/Lightbox";
import styles from "./MediaGallery.module.css";

interface MediaGalleryProps {
    media: PostMedia[];
}

export function MediaGallery({ media }: MediaGalleryProps) {
    const [lightboxIdx, setLightboxIdx] = useState<number | null>(null);
    const [lastOrientation, setLastOrientation] = useState<"landscape" | "portrait" | null>(null);

    if (media.length === 0) {
        return null;
    }

    const gridClass = media.length === 1 ? styles.grid1 : media.length === 2 ? styles.grid2 : styles.gridMany;
    const lastIdx = media.length - 1;
    const oddMany = media.length >= 3 && media.length % 2 === 1;

    function measureLast(width: number, height: number) {
        setLastOrientation(width >= height ? "landscape" : "portrait");
    }

    function itemClass(i: number): string {
        if (!oddMany || i !== lastIdx || lastOrientation === null) {
            return styles.item;
        }
        const variant = lastOrientation === "landscape" ? styles.lastSpan : styles.lastCentered;
        return `${styles.item} ${variant}`;
    }

    return (
        <>
            <div className={`${styles.gallery} ${gridClass}`}>
                {media.map((item, i) => (
                    <div key={item.id} className={itemClass(i)}>
                        {item.media_type === "video" ? (
                            <video
                                className={styles.media}
                                src={item.media_url}
                                poster={item.thumbnail_url || undefined}
                                controls
                                preload="metadata"
                                onLoadedMetadata={
                                    oddMany && i === lastIdx
                                        ? e => measureLast(e.currentTarget.videoWidth, e.currentTarget.videoHeight)
                                        : undefined
                                }
                            />
                        ) : (
                            <img
                                className={styles.media}
                                src={item.media_url}
                                alt=""
                                width={520}
                                height={510}
                                loading="lazy"
                                decoding="async"
                                onLoad={
                                    oddMany && i === lastIdx
                                        ? e => measureLast(e.currentTarget.naturalWidth, e.currentTarget.naturalHeight)
                                        : undefined
                                }
                                onClick={() => setLightboxIdx(i)}
                            />
                        )}
                    </div>
                ))}
            </div>

            {lightboxIdx !== null && media[lightboxIdx]?.media_type === "image" && (
                <Lightbox src={media[lightboxIdx].media_url} onClose={() => setLightboxIdx(null)} />
            )}
        </>
    );
}
