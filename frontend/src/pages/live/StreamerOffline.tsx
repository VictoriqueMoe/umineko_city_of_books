import { Link } from "react-router";
import styles from "./live.module.css";

export function StreamerOffline({ username }: { username: string | undefined }) {
    return (
        <div className={styles.page}>
            <div className={styles.offlineCard}>
                <h1 dir="auto" className={styles.offlineTitle}>
                    {username ? `${username} is not live right now` : "Stream not found."}
                </h1>
                {username && (
                    <p className={styles.offlineHint}>
                        This page is {username}&apos;s permanent stream address. Bookmark it and it will be here
                        whenever they go live.
                    </p>
                )}
                <div className={styles.offlineLinks}>
                    {username && (
                        <Link to={`/user/${username}`} className={styles.backLink}>
                            View profile
                        </Link>
                    )}
                    <Link to="/live" className={styles.backLink}>
                        {"←"} All live streams
                    </Link>
                </div>
            </div>
        </div>
    );
}
