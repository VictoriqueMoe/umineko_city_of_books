import { useState } from "react";
import styles from "./SpoilerImage.module.css";

interface SpoilerImageProps {
    src: string;
    alt?: string;
    isSpoiler: boolean;
    className?: string;
    imageClassName?: string;
    onClick?: () => void;
    loading?: "lazy" | "eager";
    onError?: (e: React.SyntheticEvent<HTMLImageElement>) => void;
}

export function SpoilerImage({
    src,
    alt = "",
    isSpoiler,
    className,
    imageClassName,
    onClick,
    loading,
    onError,
}: SpoilerImageProps) {
    const [revealed, setRevealed] = useState(false);
    const blurred = isSpoiler && !revealed;

    function handleClick() {
        if (blurred) {
            setRevealed(true);
            return;
        }
        onClick?.();
    }

    return (
        <div className={`${styles.wrap}${className ? ` ${className}` : ""}`} onClick={handleClick}>
            <img
                src={src}
                alt={alt}
                className={`${styles.image}${blurred ? ` ${styles.blurred}` : ""}${imageClassName ? ` ${imageClassName}` : ""}`}
                loading={loading}
                onError={onError}
            />
            {blurred && (
                <div className={styles.overlay}>
                    <span className={styles.label}>Spoiler</span>
                    <span className={styles.hint}>Click to reveal</span>
                </div>
            )}
        </div>
    );
}
