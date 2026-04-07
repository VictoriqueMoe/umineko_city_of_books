import { type PropsWithChildren, useCallback, useEffect, useRef, useState } from "react";
import type { Notification, WSMessage } from "../types/api";
import { NotificationContext, type WSMessageHandler } from "./notificationContextValue";
import { useAuth } from "../hooks/useAuth";
import * as api from "../api/endpoints";

const MAX_BACKOFF = 30000;

export function NotificationProvider({ children }: PropsWithChildren) {
    const { user } = useAuth();
    const [notifications, setNotifications] = useState<Notification[]>([]);
    const [unreadCount, setUnreadCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const wsRef = useRef<WebSocket | null>(null);
    const backoffRef = useRef(1000);
    const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const wsListenersRef = useRef<Set<WSMessageHandler>>(new Set());

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
        };

        socket.onmessage = event => {
            try {
                const msg: WSMessage = JSON.parse(event.data);
                if (msg.type === "notification") {
                    const notif = msg.data as Notification;
                    setNotifications(prev => [notif, ...prev]);
                    setUnreadCount(prev => prev + 1);
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
    }, [closeSocket]);

    useEffect(() => {
        if (!user) {
            closeSocket();
            setNotifications([]);
            setUnreadCount(0);
            return;
        }

        api.getUnreadCount()
            .then(res => {
                setUnreadCount(res.count);
            })
            .catch(() => {
                // ignore
            });

        connectWs();

        return () => {
            closeSocket();
        };
    }, [user, connectWs, closeSocket]);

    useEffect(() => {
        const base = "Umineko City of Books";
        if (unreadCount > 0) {
            document.title = `(${unreadCount}) ${base}`;
        } else {
            document.title = base;
        }
    }, [unreadCount]);

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
                loading,
                markRead,
                markAllRead,
                refreshNotifications,
                addWSListener,
                sendWSMessage,
            }}
        >
            {children}
        </NotificationContext.Provider>
    );
}
