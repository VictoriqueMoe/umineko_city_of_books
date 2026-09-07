import { useState, type MouseEvent as ReactMouseEvent, type ReactNode } from "react";
import { useResetOnChange } from "../../hooks/useResetOnChange";
import styles from "./SpoilerImage.module.css";

interface SpoilerCoverProps {
    isSpoiler: boolean;
    contentKey?: unknown;
    className?: string;
    compact?: boolean;
    onClick?: () => void;
    children: (covered: boolean) => ReactNode;
}

export function SpoilerCover({ isSpoiler, contentKey, className, compact, onClick, children }: SpoilerCoverProps) {
    const [revealed, setRevealed] = useState(false);
    const covered = isSpoiler && !revealed;

    useResetOnChange(contentKey, () => {
        setRevealed(false);
    });

    function handleClick(e: ReactMouseEvent<HTMLDivElement>) {
        if (covered) {
            e.preventDefault();
            e.stopPropagation();
            setRevealed(true);
            return;
        }

        onClick?.();
    }

    return (
        <div className={`${styles.wrap}${className ? ` ${className}` : ""}`} onClick={handleClick}>
            {children(covered)}
            {covered && <SpoilerOverlay compact={compact} />}
        </div>
    );
}

export function SpoilerOverlay({ compact }: { compact?: boolean }) {
    return (
        <div className={`${styles.overlay}${compact ? ` ${styles.overlayCompact}` : ""}`}>
            <span className={styles.label}>Spoiler</span>
            {!compact && <span className={styles.hint}>Click to reveal</span>}
        </div>
    );
}
