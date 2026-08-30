import { useState } from "react";
import { useNavigate } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { Notification } from "../../types/api";
import { useNotificationFeed } from "../../hooks/useNotificationFeed";
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
} from "../../domain/notifications";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RelativeTimestamp } from "../../components/RelativeTimestamp/RelativeTimestamp";
import styles from "./NotificationsPage.module.css";

export function NotificationsPage() {
    usePageTitle("Notifications");
    const navigate = useNavigate();
    const { unreadCount } = useNotifications();
    const [activeFilter, setActiveFilter] = useState<NotificationCategory | "all" | "unread">("unread");

    const feed = useNotificationFeed();
    const notifications = feed.notifications;
    const loading = feed.loading;
    const markingId = feed.markingId;

    function markRead(notif: Notification): Promise<void> {
        return feed.markRead(notif).catch(() => {});
    }

    async function handleClick(notif: Notification) {
        if (!notif.read) {
            await markRead(notif);
        }
        navigate(getNotificationRoute(notif));
    }

    function handleMarkReadOnly(notif: Notification) {
        markRead(notif);
    }

    function handleMarkAllRead() {
        feed.markAllRead().catch(() => {});
    }

    const grouped = groupByCategory(notifications);
    const unreadNotifications = notifications.filter(n => !n.read);

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
                                            <div dir="auto" className={styles.itemText}>
                                                <NotificationText notif={notif} />
                                            </div>
                                            <div className={styles.itemFooter}>
                                                <div className={styles.itemTime}>
                                                    <RelativeTimestamp value={notif.created_at} />
                                                </div>
                                                {!notif.read && (
                                                    <button
                                                        type="button"
                                                        className={styles.inlineMarkReadBtn}
                                                        onClick={event => {
                                                            event.stopPropagation();
                                                            handleMarkReadOnly(notif);
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

                    {feed.hasMore && (
                        <div className={styles.loadMore}>
                            <Button variant="ghost" size="small" onClick={feed.loadMore} disabled={loading}>
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
    onMarkRead: (notif: Notification) => void;
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
                            <div dir="auto" className={styles.itemText}>
                                <NotificationText notif={notif} />
                            </div>
                            <div className={styles.itemFooter}>
                                <div className={styles.itemTime}>
                                    <RelativeTimestamp value={notif.created_at} />
                                </div>
                                {!notif.read && (
                                    <button
                                        type="button"
                                        className={styles.inlineMarkReadBtn}
                                        onClick={event => {
                                            event.stopPropagation();
                                            onMarkRead(notif);
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
                {message} by {role}{" "}
                <strong>
                    <bdi>{actorName}</bdi>
                </strong>
            </>
        );
    }

    if (!notif.actor?.display_name) {
        return <>{getNotificationText(notif)}</>;
    }

    return (
        <>
            <strong>
                <bdi>{notif.actor.display_name}</bdi>
            </strong>{" "}
            {getNotificationText(notif)}
        </>
    );
}
