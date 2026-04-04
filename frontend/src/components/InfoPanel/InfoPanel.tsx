import type { ReactNode } from "react";
import styles from "./InfoPanel.module.css";

interface InfoPanelProps {
    title: string;
    children: ReactNode;
}

export function InfoPanel({ title, children }: InfoPanelProps) {
    return (
        <div className={styles.panel}>
            <h3 className={styles.title}>{title}</h3>
            {children}
        </div>
    );
}
