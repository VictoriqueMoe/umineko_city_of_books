import styles from "./ErrorBanner.module.css";

interface ErrorBannerProps {
    message: string;
}

export function ErrorBanner({ message }: ErrorBannerProps) {
    return <div className={styles.banner}>{message}</div>;
}
