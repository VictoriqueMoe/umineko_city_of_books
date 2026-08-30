import type { Gallery } from "../../../types/api";
import styles from "./GalleryCoverPreview.module.css";

interface PreviewImage {
    thumbnail_url: string;
    full_url: string;
}

interface GalleryCoverPreviewProps {
    gallery: Gallery;
}

function CoverImg({ img, className }: { img: PreviewImage; className: string }) {
    return (
        <img
            src={img.thumbnail_url || img.full_url}
            alt=""
            className={className}
            onError={e => {
                const el = e.currentTarget;
                if (!img.full_url || el.dataset.fallbackTried === "1") {
                    return;
                }

                el.dataset.fallbackTried = "1";
                el.src = img.full_url;
            }}
        />
    );
}

function Mosaic({ images }: { images: PreviewImage[] }) {
    if (images.length === 1) {
        return <CoverImg img={images[0]} className={styles.coverImage} />;
    }

    if (images.length === 2) {
        return (
            <div className={styles.preview2}>
                <CoverImg img={images[0]} className={styles.previewImg} />
                <CoverImg img={images[1]} className={styles.previewImg} />
            </div>
        );
    }

    return (
        <div className={styles.preview3}>
            <CoverImg img={images[0]} className={styles.previewMain} />
            <div className={styles.previewSide}>
                <CoverImg img={images[1]} className={styles.previewImg} />
                {images[2] && <CoverImg img={images[2]} className={styles.previewImg} />}
            </div>
        </div>
    );
}

export function GalleryCoverPreview({ gallery }: GalleryCoverPreviewProps) {
    const cover = gallery.cover_thumbnail_url || gallery.cover_image_url;
    const previews = gallery.preview_images ?? [];

    return (
        <div className={styles.cover}>
            {cover ? (
                <CoverImg
                    img={{ thumbnail_url: cover, full_url: gallery.cover_image_url }}
                    className={styles.coverImage}
                />
            ) : previews.length > 0 ? (
                <Mosaic images={previews} />
            ) : (
                <div className={styles.placeholder}>Empty</div>
            )}
        </div>
    );
}
