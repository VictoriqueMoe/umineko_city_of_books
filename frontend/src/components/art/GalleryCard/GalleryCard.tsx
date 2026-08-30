import { Link } from "react-router";
import type { Gallery } from "../../../types/api";
import { GalleryCoverPreview } from "../GalleryCoverPreview/GalleryCoverPreview";
import styles from "./GalleryCard.module.css";

interface GalleryCardProps {
    gallery: Gallery;
}

export function GalleryCard({ gallery }: GalleryCardProps) {
    return (
        <Link to={`/gallery/view/${gallery.id}`} className={styles.card}>
            <GalleryCoverPreview gallery={gallery} />
            <div className={styles.info}>
                <span dir="auto" className={styles.name}>
                    {gallery.name}
                </span>
                <span className={styles.count}>{gallery.art_count} pieces</span>
            </div>
        </Link>
    );
}
