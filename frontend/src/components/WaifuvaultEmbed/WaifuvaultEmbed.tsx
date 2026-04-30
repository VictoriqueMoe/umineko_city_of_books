import { useState } from "react";
import { Lightbox } from "../Lightbox/Lightbox";
import type { WaifuvaultMediaKind } from "./detect";
import styles from "./WaifuvaultEmbed.module.css";

interface WaifuvaultEmbedProps {
    url: string;
    kind: WaifuvaultMediaKind;
}

export function WaifuvaultEmbed({ url, kind }: WaifuvaultEmbedProps) {
    const [lightboxOpen, setLightboxOpen] = useState(false);

    if (kind === "video") {
        return (
            <span className={styles.wrapper}>
                <video className={styles.video} src={url} controls preload="metadata" />
            </span>
        );
    }

    return (
        <>
            <span className={styles.wrapper}>
                <img
                    className={styles.image}
                    src={url}
                    alt=""
                    loading="lazy"
                    decoding="async"
                    onClick={() => setLightboxOpen(true)}
                />
            </span>
            {lightboxOpen && <Lightbox src={url} onClose={() => setLightboxOpen(false)} />}
        </>
    );
}
