import { useCallback, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useNotifications } from "../../../hooks/useNotifications";
import { useClickOutside } from "../../../hooks/useClickOutside";
import {
    getNotificationText,
    getNotificationRoute,
    isContentEditedNotification,
    formatContentEditedText,
    relativeTime,
} from "../../../utils/notifications";
import type { Notification } from "../../../types/api";
import { Button } from "../../Button/Button";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./NotificationBell.module.css";

export function NotificationBell() {
    const { notifications, unreadCount, loading, markRead, markAllRead, refreshNotifications } = useNotifications();
    const [open, setOpen] = useState(false);
    const containerRef = useRef<HTMLDivElement>(null);
    const navigate = useNavigate();

    useClickOutside(containerRef, () => {
        setOpen(false);
    });

    const handleToggle = useCallback(() => {
        if (!open) {
            refreshNotifications();
        }
        setOpen(prev => !prev);
    }, [open, refreshNotifications]);

    const handleItemClick = useCallback(
        async (notif: Notification) => {
            if (!notif.read) {
                await markRead(notif.id);
            }
            setOpen(false);
            navigate(getNotificationRoute(notif));
        },
        [markRead, navigate],
    );

    const handleMarkAllRead = useCallback(async () => {
        await markAllRead();
    }, [markAllRead]);

    return (
        <div className={styles.bell} ref={containerRef}>
            <button className={styles.btn} onClick={handleToggle} aria-label="Notifications">
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

            {open && (
                <div className={styles.dropdown}>
                    <div className={styles.header}>
                        <h4>Notifications</h4>
                        {unreadCount > 0 && (
                            <Button variant="ghost" size="small" onClick={handleMarkAllRead}>
                                Mark all as read
                            </Button>
                        )}
                    </div>

                    {loading && notifications.length === 0 ? (
                        <div className={styles.empty}>Loading...</div>
                    ) : notifications.length === 0 ? (
                        <div className={styles.empty}>No notifications yet</div>
                    ) : (
                        notifications.map(notif => (
                            <div
                                key={notif.id}
                                className={`${styles.item}${notif.read ? "" : ` ${styles.unread}`}`}
                                onClick={() => handleItemClick(notif)}
                            >
                                <ProfileLink user={notif.actor} size="small" showName={false} />
                                <div className={styles.itemContent}>
                                    <div className={styles.itemText}>
                                        <NotificationText notif={notif} />
                                    </div>
                                    <div className={styles.itemTime}>{relativeTime(notif.created_at)}</div>
                                </div>
                            </div>
                        ))
                    )}
                </div>
            )}
        </div>
    );
}

function NotificationText({ notif }: { notif: Notification }) {
    if (isContentEditedNotification(notif)) {
        const { message, role, actorName } = formatContentEditedText(notif);
        return (
            <>
                {message} by {role} <strong>{actorName}</strong>
            </>
        );
    }

    return (
        <>
            <strong>{notif.actor.display_name}</strong> {getNotificationText(notif)}
        </>
    );
}
