import { Link } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import styles from "./LockBanner.module.css";

export function LockBanner() {
    const { user } = useAuth();
    if (!user || !user.locked) {
        return null;
    }

    return (
        <div className={styles.banner}>
            <span className={styles.text}>
                Your account is locked. You can still read the site and send direct messages to site staff.
                {user.lock_reason ? ` Reason: ${user.lock_reason}` : ""}
            </span>
            <Link to="/users" className={styles.button}>
                Find a moderator
            </Link>
        </div>
    );
}
