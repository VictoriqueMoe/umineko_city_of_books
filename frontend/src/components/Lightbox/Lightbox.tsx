import { useEffect } from "react";
import styles from "./Lightbox.module.css";

interface LightboxProps {
    src: string;
    alt?: string;
    onClose: () => void;
}

export function Lightbox({ src, alt = "", onClose }: LightboxProps) {
    useEffect(() => {
        function onKey(e: KeyboardEvent) {
            if (e.key === "Escape") {
                onClose();
            }
        }
        window.addEventListener("keydown", onKey);
        const prevOverflow = document.body.style.overflow;
        document.body.style.overflow = "hidden";
        return () => {
            window.removeEventListener("keydown", onKey);
            document.body.style.overflow = prevOverflow;
        };
    }, [onClose]);

    return (
        <div className={styles.overlay} onClick={onClose} role="dialog" aria-modal="true">
            <button className={styles.close} onClick={onClose} aria-label="Close">
                {"\u2715"}
            </button>
            <img className={styles.image} src={src} alt={alt} onClick={e => e.stopPropagation()} />
        </div>
    );
}
