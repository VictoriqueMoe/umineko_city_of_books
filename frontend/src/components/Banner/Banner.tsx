import type { AriaRole, ReactNode } from "react";
import { Link } from "react-router";
import styles from "./Banner.module.css";

export type BannerColour = "red" | "amber" | "blue" | "teal" | "purple" | "gold";

interface BannerProps {
    colour: BannerColour;
    children: ReactNode;
    actions?: ReactNode;
    role?: AriaRole;
    label?: string;
}

interface BannerButtonProps {
    onClick: () => void;
    disabled?: boolean;
    children: ReactNode;
}

interface BannerLinkProps {
    to: string;
    children: ReactNode;
}

interface BannerDismissProps {
    onDismiss: () => void;
}

interface BannerNoteProps {
    children: ReactNode;
}

export function Banner({ colour, children, actions, role, label }: BannerProps) {
    return (
        <div className={`${styles.banner} ${styles[colour]}`} role={role} aria-label={label}>
            <div dir="auto" className={styles.text}>
                {children}
            </div>
            {actions}
        </div>
    );
}

export function BannerButton({ onClick, disabled = false, children }: BannerButtonProps) {
    return (
        <button type="button" onClick={onClick} disabled={disabled} className={styles.button}>
            {children}
        </button>
    );
}

export function BannerLink({ to, children }: BannerLinkProps) {
    return (
        <Link to={to} className={styles.button}>
            {children}
        </Link>
    );
}

export function BannerDismiss({ onDismiss }: BannerDismissProps) {
    return (
        <button type="button" onClick={onDismiss} className={styles.dismiss} aria-label="Dismiss">
            &times;
        </button>
    );
}

export function BannerNote({ children }: BannerNoteProps) {
    return <span className={styles.note}>{children}</span>;
}
