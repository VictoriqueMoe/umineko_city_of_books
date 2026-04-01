import { useState } from "react";
import type { PostMedia } from "../../../types/api";
import { Modal } from "../../Modal/Modal";
import styles from "./MediaGallery.module.css";

interface MediaGalleryProps {
    media: PostMedia[];
}

export function MediaGallery({ media }: MediaGalleryProps) {
    const [lightboxIdx, setLightboxIdx] = useState<number | null>(null);

    if (media.length === 0) {
        return null;
    }

    const gridClass = media.length === 1 ? styles.grid1 : media.length === 2 ? styles.grid2 : styles.gridMany;

    return (
        <>
            <div className={`${styles.gallery} ${gridClass}`}>
                {media.map((item, i) => (
                    <div key={item.id} className={styles.item}>
                        {item.media_type === "video" ? (
                            <video
                                className={styles.media}
                                src={item.media_url}
                                poster={item.thumbnail_url || undefined}
                                controls
                                preload="metadata"
                            />
                        ) : (
                            <img
                                className={styles.media}
                                src={item.media_url}
                                alt=""
                                loading="lazy"
                                onClick={() => setLightboxIdx(i)}
                            />
                        )}
                    </div>
                ))}
            </div>

            {lightboxIdx !== null && media[lightboxIdx]?.media_type === "image" && (
                <Modal isOpen onClose={() => setLightboxIdx(null)} title="">
                    <div className={styles.lightbox}>
                        <img src={media[lightboxIdx].media_url} alt="" />
                    </div>
                </Modal>
            )}
        </>
    );
}
