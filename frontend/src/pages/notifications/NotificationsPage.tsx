import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import type { Notification } from "../../types/api";
import { getNotifications } from "../../api/endpoints";
import { useNotifications } from "../../hooks/useNotifications";
import {
    type NotificationCategory,
    getNotificationRoute,
    getNotificationText,
    getCategoryLabel,
    getCategoryOrder,
    groupByCategory,
    isContentEditedNotification,
    formatContentEditedText,
    relativeTime,
} from "../../utils/notifications";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import styles from "./NotificationsPage.module.css";

export function NotificationsPage() {
    const navigate = useNavigate();
    const { markRead, markAllRead, unreadCount } = useNotifications();
    const [notifications, setNotifications] = useState<Notification[]>([]);
    const [loading, setLoading] = useState(true);
    const [total, setTotal] = useState(0);

    const fetchAll = useCallback(async (offset = 0) => {
        setLoading(true);
        try {
            const res = await getNotifications({ limit: 50, offset });
            if (offset === 0) {
                setNotifications(res.notifications);
            } else {
                setNotifications(prev => [...prev, ...res.notifications]);
            }
            setTotal(res.total);
        } catch {
            // ignore
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchAll();
    }, [fetchAll]);

    async function handleClick(notif: Notification) {
        if (!notif.read) {
            await markRead(notif.id);
            setNotifications(prev =>
                prev.map(n => {
                    if (n.id === notif.id) {
                        return { ...n, read: true };
                    }
                    return n;
                }),
            );
        }
        navigate(getNotificationRoute(notif));
    }

    async function handleMarkAllRead() {
        await markAllRead();
        setNotifications(prev => prev.map(n => ({ ...n, read: true })));
    }

    const grouped = groupByCategory(notifications);
    const hasMore = notifications.length < total;

    return (
        <div className={styles.page}>
            <div className={styles.topBar}>
                <h1 className={styles.title}>Notifications</h1>
                {unreadCount > 0 && (
                    <Button variant="ghost" size="small" onClick={handleMarkAllRead}>
                        Mark all as read
                    </Button>
                )}
            </div>

            {loading && notifications.length === 0 ? (
                <div className={styles.empty}>Loading notifications...</div>
            ) : notifications.length === 0 ? (
                <div className={styles.empty}>No notifications yet</div>
            ) : (
                <>
                    {getCategoryOrder().map(cat => {
                        const items = grouped.get(cat);
                        if (!items || items.length === 0) {
                            return null;
                        }
                        const unread = items.filter(n => !n.read).length;
                        return (
                            <CategorySection
                                key={cat}
                                category={cat}
                                notifications={items}
                                unreadCount={unread}
                                onClick={handleClick}
                            />
                        );
                    })}

                    {hasMore && (
                        <div className={styles.loadMore}>
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => fetchAll(notifications.length)}
                                disabled={loading}
                            >
                                {loading ? "Loading..." : "Load more"}
                            </Button>
                        </div>
                    )}
                </>
            )}
        </div>
    );
}

function CategorySection({
    category,
    notifications,
    unreadCount,
    onClick,
}: {
    category: NotificationCategory;
    notifications: Notification[];
    unreadCount: number;
    onClick: (notif: Notification) => void;
}) {
    return (
        <div className={styles.categorySection}>
            <div className={styles.categoryHeader}>
                <span className={styles.categoryLabel}>{getCategoryLabel(category)}</span>
                {unreadCount > 0 && <span className={styles.categoryBadge}>{unreadCount}</span>}
            </div>
            <div className={styles.list}>
                {notifications.map(notif => (
                    <div
                        key={notif.id}
                        className={`${styles.item}${notif.read ? "" : ` ${styles.unread}`}`}
                        onClick={() => onClick(notif)}
                    >
                        <ProfileLink user={notif.actor} size="small" showName={false} />
                        <div className={styles.itemContent}>
                            <div className={styles.itemText}>
                                <NotificationText notif={notif} />
                            </div>
                            <div className={styles.itemTime}>{relativeTime(notif.created_at)}</div>
                        </div>
                    </div>
                ))}
            </div>
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
