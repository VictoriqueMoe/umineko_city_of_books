import { type PropsWithChildren, useCallback, useEffect, useRef, useState } from "react";
import type { Notification, UserProfile, WSMessage } from "../types/api";
import { NotificationContext, type WSMessageHandler } from "./notificationContextValue";
import { useAuth } from "../hooks/useAuth";
import * as api from "../api/endpoints";
import { showDesktopNotification } from "../utils/notifications";
import { playNotificationSound } from "../utils/sound";

const MAX_BACKOFF = 30000;

export function NotificationProvider({ children }: PropsWithChildren) {
    const { user, setUser } = useAuth();
    const [notifications, setNotifications] = useState<Notification[]>([]);
    const [unreadCount, setUnreadCount] = useState(0);
    const [chatUnreadCount, setChatUnreadCount] = useState(0);
    const [liveGamesCount, setLiveGamesCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [wsEpoch, setWsEpoch] = useState(0);
    const wsRef = useRef<WebSocket | null>(null);
    const backoffRef = useRef(1000);
    const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const wsListenersRef = useRef<Set<WSMessageHandler>>(new Set());
    const userRef = useRef(user);
    userRef.current = user;

    const clearReconnectTimer = useCallback(() => {
        if (reconnectTimerRef.current !== null) {
            clearTimeout(reconnectTimerRef.current);
            reconnectTimerRef.current = null;
        }
    }, []);

    const closeSocket = useCallback(() => {
        clearReconnectTimer();
        if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
        }
    }, [clearReconnectTimer]);

    const connectWs = useCallback(() => {
        closeSocket();

        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        const wsUrl = `${protocol}//${window.location.host}/api/v1/ws`;
        const socket = new WebSocket(wsUrl);
        wsRef.current = socket;

        socket.onopen = () => {
            backoffRef.current = 1000;
            setWsEpoch(n => n + 1);
            window.dispatchEvent(new CustomEvent("site-info-refresh"));
        };

        socket.onmessage = event => {
            try {
                const msg: WSMessage = JSON.parse(event.data);
                if (msg.type === "notification") {
                    const notif = msg.data as Notification;
                    setNotifications(prev => [notif, ...prev]);
                    setUnreadCount(prev => prev + 1);
                    showDesktopNotification(notif);
                    if (userRef.current?.play_notification_sound ?? true) {
                        playNotificationSound();
                    }
                }
                if (msg.type === "role_changed") {
                    const data = msg.data as { user_id?: string; role?: string };
                    if (data.user_id && userRef.current && data.user_id === userRef.current.id) {
                        setUser({ ...userRef.current, role: (data.role ?? "") as UserProfile["role"] });
                    }
                }
                if (
                    msg.type === "top_detective_changed" ||
                    msg.type === "top_gm_changed" ||
                    msg.type === "vanity_roles_changed"
                ) {
                    window.dispatchEvent(new CustomEvent("site-info-refresh"));
                }
                if (msg.type === "chat_unread_bumped" || msg.type === "chat_read") {
                    const data = msg.data as { total?: number };
                    if (typeof data.total === "number") {
                        setChatUnreadCount(data.total);
                    }
                }
                if (msg.type === "live_games_count") {
                    const data = msg.data as { count?: number };
                    if (typeof data.count === "number") {
                        setLiveGamesCount(data.count);
                    }
                }
                if (msg.type === "secret_closed") {
                    window.dispatchEvent(new CustomEvent("secret-closed", { detail: msg.data }));
                }
                for (const handler of wsListenersRef.current) {
                    handler(msg);
                }
            } catch {
                // ignore
            }
        };

        socket.onclose = () => {
            wsRef.current = null;
            const delay = Math.min(backoffRef.current, MAX_BACKOFF);
            backoffRef.current = delay * 2;
            reconnectTimerRef.current = setTimeout(() => {
                connectWs();
            }, delay);
        };

        socket.onerror = () => {
            socket.close();
        };
    }, [closeSocket, setUser]);

    useEffect(() => {
        api.listLiveGameRooms()
            .then(res => {
                setLiveGamesCount(res.total ?? 0);
            })
            .catch(() => {
                // ignore
            });
    }, [user]);

    useEffect(() => {
        if (!user) {
            closeSocket();
            setNotifications([]);
            setUnreadCount(0);
            setChatUnreadCount(0);
            return;
        }

        api.getUnreadCount()
            .then(res => {
                setUnreadCount(res.count);
            })
            .catch(() => {
                // ignore
            });

        api.getChatUnreadCount()
            .then(res => {
                setChatUnreadCount(res.count);
            })
            .catch(() => {
                // ignore
            });

        connectWs();

        return () => {
            closeSocket();
        };
    }, [user, connectWs, closeSocket]);

    // Page titles are now handled by usePageTitle hook per page

    const markRead = useCallback(async (id: number) => {
        await api.markNotificationRead(id);
        setNotifications(prev =>
            prev.map(n => {
                if (n.id === id) {
                    return { ...n, read: true };
                }
                return n;
            }),
        );
        setUnreadCount(prev => Math.max(0, prev - 1));
        api.getUnreadCount()
            .then(res => setUnreadCount(res.count))
            .catch(() => {});
    }, []);

    const markAllRead = useCallback(async () => {
        await api.markAllNotificationsRead();
        setNotifications(prev => prev.map(n => ({ ...n, read: true })));
        setUnreadCount(0);
    }, []);

    const refreshNotifications = useCallback(async () => {
        setLoading(true);
        try {
            const res = await api.getNotifications({ limit: 20 });
            setNotifications(res.notifications);
            const unread = res.notifications.filter(n => !n.read).length;
            setUnreadCount(unread);
        } catch {
            // ignore
        } finally {
            setLoading(false);
        }
    }, []);

    const addWSListener = useCallback((handler: WSMessageHandler) => {
        wsListenersRef.current.add(handler);
        return () => {
            wsListenersRef.current.delete(handler);
        };
    }, []);

    const sendWSMessage = useCallback((msg: object) => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify(msg));
        }
    }, []);

    return (
        <NotificationContext.Provider
            value={{
                notifications,
                unreadCount,
                chatUnreadCount,
                liveGamesCount,
                loading,
                markRead,
                markAllRead,
                refreshNotifications,
                addWSListener,
                sendWSMessage,
                wsEpoch,
            }}
        >
            {children}
        </NotificationContext.Provider>
    );
}
