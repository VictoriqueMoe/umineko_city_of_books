import type { PostMedia } from "../../types/api";
import styles from "./ExistingMediaGrid.module.css";

interface ExistingMediaGridProps {
    media: PostMedia[];
    pendingIds: number[];
    onToggle: (id: number) => void;
    removeLabel: string;
    preferThumbnail?: boolean;
    className?: string;
}

export function ExistingMediaGrid({
    media,
    pendingIds,
    onToggle,
    removeLabel,
    preferThumbnail,
    className,
}: ExistingMediaGridProps) {
    return (
        <div className={`${styles.grid}${className ? ` ${className}` : ""}`}>
            {media.map(m => {
                const pending = pendingIds.includes(m.id);
                const itemClass = `${styles.item}${pending ? ` ${styles.itemPending}` : ""}`;
                const imageSrc = preferThumbnail ? m.thumbnail_url || m.media_url : m.media_url;

                return (
                    <div key={m.id} className={itemClass}>
                        {m.media_type === "video" ? (
                            <video src={m.media_url} className={styles.thumb} />
                        ) : (
                            <img src={imageSrc} className={styles.thumb} alt="" />
                        )}
                        <button
                            type="button"
                            className={styles.remove}
                            onClick={() => onToggle(m.id)}
                            aria-label={pending ? "Undo remove" : removeLabel}
                            title={pending ? "Undo remove" : "Remove on save"}
                        >
                            {pending ? "↺" : "×"}
                        </button>
                    </div>
                );
            })}
        </div>
    );
}
