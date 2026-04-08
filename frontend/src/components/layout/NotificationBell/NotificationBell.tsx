import { useNavigate } from "react-router";
import { useNotifications } from "../../../hooks/useNotifications";
import styles from "./NotificationBell.module.css";

export function NotificationBell() {
    const { unreadCount } = useNotifications();
    const navigate = useNavigate();

    return (
        <button className={styles.btn} onClick={() => navigate("/notifications")} aria-label="Notifications">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path
                    d="M8 1C5.79 1 4 2.79 4 5v3l-1.3 1.3a.5.5 0 00.35.85h9.9a.5.5 0 00.35-.85L12 8V5c0-2.21-1.79-4-4-4zM6.5 11.5a1.5 1.5 0 003 0"
                    stroke="currentColor"
                    strokeWidth="1.2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
            </svg>
            {unreadCount > 0 && <span className={styles.badge}>{unreadCount > 99 ? "99+" : unreadCount}</span>}
        </button>
    );
}
