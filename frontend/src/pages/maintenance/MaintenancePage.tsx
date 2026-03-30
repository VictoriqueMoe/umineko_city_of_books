import styles from "./MaintenancePage.module.css";

export function MaintenancePage() {
    return (
        <div className={styles.page}>
            <div className={styles.card}>
                <h1 className={styles.title}>The game board is being prepared</h1>
                <p className={styles.message}>Without love, it cannot be seen. Please check back shortly.</p>
            </div>
        </div>
    );
}
