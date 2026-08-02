import { type PropsWithChildren, useEffect, useId, useRef } from "react";
import styles from "./Modal.module.css";

interface ModalProps {
    isOpen: boolean;
    onClose: () => void;
    title: string;
}

export function Modal({ isOpen, onClose, title, children }: PropsWithChildren<ModalProps>) {
    const panelRef = useRef<HTMLDivElement>(null);
    const titleId = useId();

    useEffect(() => {
        if (!isOpen) {
            return;
        }

        function onKeyDown(e: KeyboardEvent) {
            if (e.key === "Escape") {
                onClose();
            }
        }

        document.addEventListener("keydown", onKeyDown);

        return () => {
            document.removeEventListener("keydown", onKeyDown);
        };
    }, [isOpen, onClose]);

    useEffect(() => {
        if (!isOpen) {
            return;
        }

        const previous = document.activeElement as HTMLElement | null;
        panelRef.current?.focus();

        return () => {
            previous?.focus();
        };
    }, [isOpen]);

    if (!isOpen) {
        return null;
    }

    return (
        <div className={styles.overlay} onClick={onClose}>
            <div
                ref={panelRef}
                className={styles.modal}
                role="dialog"
                aria-modal="true"
                aria-labelledby={titleId}
                tabIndex={-1}
                onClick={e => e.stopPropagation()}
            >
                <div className={styles.header}>
                    <h3 id={titleId}>{title}</h3>
                    <button type="button" className={styles.close} onClick={onClose}>
                        {"\u2715"}
                    </button>
                </div>
                <div className={styles.body}>{children}</div>
            </div>
        </div>
    );
}
