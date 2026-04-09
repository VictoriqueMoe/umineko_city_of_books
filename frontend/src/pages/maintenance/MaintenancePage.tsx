import { usePageTitle } from "../../hooks/usePageTitle";
import styles from "./MaintenancePage.module.css";

interface MaintenancePageProps {
    title?: string;
    message?: string;
}

export function MaintenancePage({ title, message }: MaintenancePageProps) {
    usePageTitle("Maintenance");
    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <h1 className={styles.title}>{title || "The game board is being prepared"}</h1>
                <p className={styles.message}>
                    {message || "Without love, it cannot be seen. Please check back shortly."}
                </p>
            </div>
        </div>
    );
}
