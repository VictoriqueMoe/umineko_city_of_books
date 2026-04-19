import { type ReactNode, useEffect } from "react";
import styles from "./Toast.module.css";

export type ToastVariant = "default" | "success" | "error" | "arcane";

interface ToastProps {
    children: ReactNode;
    variant?: ToastVariant;
    duration?: number;
    onDismiss?: () => void;
}

export function Toast({ children, variant = "default", duration = 4000, onDismiss }: ToastProps) {
    useEffect(() => {
        if (!onDismiss || duration <= 0) {
            return;
        }
        const id = window.setTimeout(onDismiss, duration);
        return () => {
            window.clearTimeout(id);
        };
    }, [onDismiss, duration]);

    return (
        <div role="status" aria-live="polite" className={`${styles.toast} ${styles[variant]}`}>
            {children}
        </div>
    );
}
