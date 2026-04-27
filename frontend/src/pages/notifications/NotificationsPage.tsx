import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { useQueryClient } from "@tanstack/react-query";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { Notification, WSMessage } from "../../types/api";
import { useNotifications as useNotificationsQuery } from "../../api/queries/notification";
import { queryKeys } from "../../api/queryKeys";
import { useNotifications } from "../../hooks/useNotifications";
import {
    formatContentEditedText,
    getCategoryLabel,
    getCategoryOrder,
    getNotificationRoute,
    getNotificationText,
    groupByCategory,
    isContentEditedNotification,
    type NotificationCategory,
    relativeTime,
} from "../../utils/notifications";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import styles from "./NotificationsPage.module.css";

export function NotificationsPage() {
    usePageTitle("Notifications");
    const navigate = useNavigate();
    const { markRead, markAllRead, unreadCount, addWSListener } = useNotifications();
    const queryClient = useQueryClient();
    const [activeFilter, setActiveFilter] = useState<NotificationCategory | "all" | "unread">("unread");
    const [markingId, setMarkingId] = useState<number | null>(null);

    const notifQuery = useNotificationsQuery(50, 0);
    const notifications = notifQuery.notifications;
    const total = notifQuery.total;
    const loading = notifQuery.loading;
    const fetchAll = async () => {
        await notifQuery.refresh();
    };
    const listKey = queryKeys.notifications.list({ limit: 50, offset: 0 });

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type !== "notification") {
                return;
            }
            const notif = msg.data as Notification;
            queryClient.setQueryData<{ notifications: Notification[]; total: number }>(listKey, prev => {
                if (!prev) {
                    return prev;
                }
                if (prev.notifications.some(n => n.id === notif.id)) {
                    return prev;
                }
                return { notifications: [notif, ...prev.notifications], total: prev.total + 1 };
            });
        });
    }, [addWSListener, queryClient, listKey]);

    async function handleClick(notif: Notification) {
        if (!notif.read) {
            await handleMarkReadOnly(notif);
        }
        navigate(getNotificationRoute(notif));
    }

    async function handleMarkReadOnly(notif: Notification) {
        if (notif.read || markingId === notif.id) {
            return;
        }
        setMarkingId(notif.id);
        try {
            await markRead(notif.id);
            queryClient.setQueryData<{ notifications: Notification[]; total: number }>(listKey, prev => {
                if (!prev) {
                    return prev;
                }
                return {
                    ...prev,
                    notifications: prev.notifications.map(n => {
                        if (n.id === notif.id) {
                            return { ...n, read: true };
                        }
                        return n;
                    }),
                };
            });
        } finally {
            setMarkingId(current => (current === notif.id ? null : current));
        }
    }

    async function handleMarkAllRead() {
        await markAllRead();
        queryClient.setQueryData<{ notifications: Notification[]; total: number }>(listKey, prev =>
            prev ? { ...prev, notifications: prev.notifications.map(n => ({ ...n, read: true })) } : prev,
        );
    }

    const grouped = groupByCategory(notifications);
    const unreadNotifications = notifications.filter(n => !n.read);
    const hasMore = notifications.length < total;

    const availableCategories = getCategoryOrder().filter(cat => {
        const items = grouped.get(cat);
        return items && items.length > 0;
    });

    return (
        <div className={styles.page}>
            <div className={styles.topBar}>
                <h1 className={styles.title}>Notifications</h1>
                {unreadCount > 0 && (
                    <button className={styles.markAllBtn} onClick={handleMarkAllRead}>
                        Mark all as read
                    </button>
                )}
            </div>

            {loading && notifications.length === 0 ? (
                <div className={styles.empty}>Loading notifications...</div>
            ) : notifications.length === 0 ? (
                <div className={styles.empty}>No notifications yet</div>
            ) : (
                <>
                    <div className={styles.tabs}>
                        <button
                            className={`${styles.tab}${activeFilter === "unread" ? ` ${styles.tabActive}` : ""}`}
                            onClick={() => setActiveFilter("unread")}
                        >
                            Unread
                            {unreadCount > 0 && <span className={styles.tabBadge}>{unreadCount}</span>}
                        </button>
                        <button
                            className={`${styles.tab}${activeFilter === "all" ? ` ${styles.tabActive}` : ""}`}
                            onClick={() => setActiveFilter("all")}
                        >
                            All
                        </button>
                        {availableCategories.map(cat => {
                            const items = grouped.get(cat)!;
                            const catUnread = items.filter(n => !n.read).length;
                            return (
                                <button
                                    key={cat}
                                    className={`${styles.tab}${activeFilter === cat ? ` ${styles.tabActive}` : ""}`}
                                    onClick={() => setActiveFilter(cat)}
                                >
                                    {getCategoryLabel(cat)}
                                    {catUnread > 0 && <span className={styles.tabBadge}>{catUnread}</span>}
                                </button>
                            );
                        })}
                    </div>

                    {activeFilter === "unread" ? (
                        <div className={styles.flatList}>
                            {unreadNotifications.length === 0 ? (
                                <div className={styles.empty}>No unread notifications</div>
                            ) : (
                                unreadNotifications.map(notif => (
                                    <div
                                        key={notif.id}
                                        className={`${styles.item} ${styles.unread}`}
                                        onClick={() => handleClick(notif)}
                                    >
                                        <ProfileLink user={notif.actor} size="small" showName={false} />
                                        <div className={styles.itemContent}>
                                            <div className={styles.itemText}>
                                                <NotificationText notif={notif} />
                                            </div>
                                            <div className={styles.itemFooter}>
                                                <div className={styles.itemTime}>{relativeTime(notif.created_at)}</div>
                                                {!notif.read && (
                                                    <button
                                                        type="button"
                                                        className={styles.inlineMarkReadBtn}
                                                        onClick={event => {
                                                            event.stopPropagation();
                                                            void handleMarkReadOnly(notif);
                                                        }}
                                                        disabled={markingId === notif.id}
                                                    >
                                                        {markingId === notif.id ? "Marking..." : "Mark as read"}
                                                    </button>
                                                )}
                                            </div>
                                        </div>
                                    </div>
                                ))
                            )}
                        </div>
                    ) : (
                        getCategoryOrder().map(cat => {
                            if (activeFilter !== "all" && activeFilter !== cat) {
                                return null;
                            }
                            const items = grouped.get(cat);
                            if (!items || items.length === 0) {
                                return null;
                            }
                            const catUnread = items.filter(n => !n.read).length;
                            return (
                                <CategorySection
                                    key={cat}
                                    category={cat}
                                    notifications={items}
                                    unreadCount={catUnread}
                                    onClick={handleClick}
                                    onMarkRead={handleMarkReadOnly}
                                    markingId={markingId}
                                />
                            );
                        })
                    )}

                    {hasMore && (
                        <div className={styles.loadMore}>
                            <Button variant="ghost" size="small" onClick={() => fetchAll()} disabled={loading}>
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
    onMarkRead,
    markingId,
}: {
    category: NotificationCategory;
    notifications: Notification[];
    unreadCount: number;
    onClick: (notif: Notification) => void;
    onMarkRead: (notif: Notification) => Promise<void>;
    markingId: number | null;
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
                            <div className={styles.itemFooter}>
                                <div className={styles.itemTime}>{relativeTime(notif.created_at)}</div>
                                {!notif.read && (
                                    <button
                                        type="button"
                                        className={styles.inlineMarkReadBtn}
                                        onClick={event => {
                                            event.stopPropagation();
                                            void onMarkRead(notif);
                                        }}
                                        disabled={markingId === notif.id}
                                    >
                                        {markingId === notif.id ? "Marking..." : "Mark as read"}
                                    </button>
                                )}
                            </div>
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

function NotificationText({ notif }: { notif: Notification }) {
    if (notif.count > 1 && notif.type === "chat_room_message" && notif.message) {
        return <>{notif.message}</>;
    }

    if (isContentEditedNotification(notif)) {
        const { message, role, actorName } = formatContentEditedText(notif);
        return (
            <>
                {message} by {role} <strong>{actorName}</strong>
            </>
        );
    }

    if (!notif.actor?.display_name) {
        return <>{getNotificationText(notif)}</>;
    }

    return (
        <>
            <strong>{notif.actor.display_name}</strong> {getNotificationText(notif)}
        </>
    );
}
