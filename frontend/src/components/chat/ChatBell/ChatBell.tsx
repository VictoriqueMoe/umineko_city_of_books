import { useNavigate } from "react-router";
import { useNotifications } from "../../../hooks/useNotifications.ts";
import styles from "./ChatBell.module.css";

export function ChatBell() {
    const { chatUnreadCount } = useNotifications();
    const navigate = useNavigate();

    return (
        <button className={styles.btn} onClick={() => navigate("/chat")} aria-label="Direct messages">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path
                    d="M2 3h12a1 1 0 011 1v8a1 1 0 01-1 1H2a1 1 0 01-1-1V4a1 1 0 011-1z"
                    stroke="currentColor"
                    strokeWidth="1.2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
                <path
                    d="M1.5 4.2L8 9l6.5-4.8"
                    stroke="currentColor"
                    strokeWidth="1.2"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                />
            </svg>
            {chatUnreadCount > 0 && (
                <span className={styles.badge}>{chatUnreadCount > 99 ? "99+" : chatUnreadCount}</span>
            )}
        </button>
    );
}
