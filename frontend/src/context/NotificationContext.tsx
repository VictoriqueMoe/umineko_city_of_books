import { type PropsWithChildren, useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { Notification, UserProfile, WSMessage } from "../types/api";
import { NotificationContext, type WSMessageHandler } from "./notificationContextValue";
import { useAuth } from "../hooks/useAuth";
import { useUnreadCount } from "../api/queries/notification";
import { useChatUnreadCount } from "../api/queries/chat";
import { useLiveGameRooms } from "../api/queries/gameRoom";
import { useMarkAllNotificationsRead, useMarkNotificationRead } from "../api/mutations/notification";
import { queryKeys } from "../api/queryKeys";
import { showDesktopNotification } from "../utils/notifications";
import { playNotificationSound } from "../utils/sound";

const MAX_BACKOFF = 30000;
const KEEPALIVE_INTERVAL_MS = 20_000;
const STALE_THRESHOLD_MS = 45_000;

export function NotificationProvider({ children }: PropsWithChildren) {
    const { user, setUser } = useAuth();
    const qc = useQueryClient();
    const [wsEpoch, setWsEpoch] = useState(0);

    const unreadCountQuery = useUnreadCount();
    const chatUnreadCountQuery = useChatUnreadCount();
    const liveGamesQuery = useLiveGameRooms();
    const unreadCount = user ? unreadCountQuery.count : 0;
    const chatUnreadCount = user ? chatUnreadCountQuery.count : 0;
    const liveGamesCount = liveGamesQuery.total ?? 0;

    const markReadMutation = useMarkNotificationRead();
    const markAllReadMutation = useMarkAllNotificationsRead();

    const wsRef = useRef<WebSocket | null>(null);
    const backoffRef = useRef(1000);
    const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const keepaliveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
    const lastMessageAtRef = useRef(0);
    const wsListenersRef = useRef<Set<WSMessageHandler>>(new Set());
    const userRef = useRef(user);
    useEffect(() => {
        userRef.current = user;
    }, [user]);

    const clearReconnectTimer = useCallback(() => {
        if (reconnectTimerRef.current !== null) {
            clearTimeout(reconnectTimerRef.current);
            reconnectTimerRef.current = null;
        }
    }, []);

    const clearKeepaliveTimer = useCallback(() => {
        if (keepaliveTimerRef.current !== null) {
            clearInterval(keepaliveTimerRef.current);
            keepaliveTimerRef.current = null;
        }
    }, []);

    const closeSocket = useCallback(() => {
        clearReconnectTimer();
        clearKeepaliveTimer();
        if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
        }
    }, [clearReconnectTimer, clearKeepaliveTimer]);

    const connectWsRef = useRef<() => void>(() => {});

    const bumpUnread = useCallback(() => {
        qc.setQueryData<{ count: number }>(queryKeys.notifications.unreadCount(), prev => ({
            count: (prev?.count ?? 0) + 1,
        }));
    }, [qc]);

    const setChatUnreadCount = useCallback(
        (total: number) => {
            qc.setQueryData<{ count: number }>(["chat", "unread-count"], { count: total });
        },
        [qc],
    );

    const setLiveGamesCount = useCallback(
        (count: number) => {
            qc.setQueryData<{ rooms: unknown[]; total: number }>(["game-rooms", "live", ""], prev => ({
                rooms: prev?.rooms ?? [],
                total: count,
            }));
        },
        [qc],
    );

    const connectWs = useCallback(() => {
        closeSocket();

        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        const wsUrl = `${protocol}//${window.location.host}/api/v1/ws`;
        const socket = new WebSocket(wsUrl);
        wsRef.current = socket;

        socket.onopen = () => {
            backoffRef.current = 1000;
            lastMessageAtRef.current = Date.now();
            setWsEpoch(n => n + 1);
            window.dispatchEvent(new CustomEvent("site-info-refresh"));

            clearKeepaliveTimer();
            keepaliveTimerRef.current = setInterval(() => {
                if (wsRef.current !== socket) {
                    return;
                }
                if (Date.now() - lastMessageAtRef.current > STALE_THRESHOLD_MS) {
                    socket.close();
                    return;
                }
                if (socket.readyState === WebSocket.OPEN) {
                    socket.send(JSON.stringify({ type: "ping", data: {} }));
                }
            }, KEEPALIVE_INTERVAL_MS);
        };

        socket.onmessage = event => {
            lastMessageAtRef.current = Date.now();
            try {
                const msg: WSMessage = JSON.parse(event.data);
                if (msg.type === "pong") {
                    return;
                }
                if (msg.type === "notification") {
                    const notif = msg.data as Notification;
                    bumpUnread();
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
                if (msg.type === "lock_changed") {
                    const data = msg.data as { user_id?: string; locked?: boolean; lock_reason?: string };
                    if (data.user_id && userRef.current && data.user_id === userRef.current.id) {
                        setUser({
                            ...userRef.current,
                            locked: !!data.locked,
                            lock_reason: data.lock_reason ?? "",
                        });
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
                return;
            }
        };

        socket.onclose = () => {
            wsRef.current = null;
            clearKeepaliveTimer();
            const delay = Math.min(backoffRef.current, MAX_BACKOFF);
            backoffRef.current = delay * 2;
            reconnectTimerRef.current = setTimeout(() => {
                connectWsRef.current();
            }, delay);
        };

        socket.onerror = () => {
            socket.close();
        };
    }, [closeSocket, clearKeepaliveTimer, setUser, bumpUnread, setChatUnreadCount, setLiveGamesCount]);

    useEffect(() => {
        connectWsRef.current = connectWs;
    }, [connectWs]);

    useEffect(() => {
        function onVisible() {
            if (document.visibilityState !== "visible") {
                return;
            }
            const socket = wsRef.current;
            if (!socket) {
                return;
            }
            if (Date.now() - lastMessageAtRef.current > STALE_THRESHOLD_MS) {
                socket.close();
                return;
            }
            if (socket.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({ type: "ping", data: {} }));
            }
        }
        document.addEventListener("visibilitychange", onVisible);
        return () => {
            document.removeEventListener("visibilitychange", onVisible);
        };
    }, []);

    const userId = user?.id;
    useEffect(() => {
        if (!userId) {
            closeSocket();
            return;
        }
        connectWs();
        return () => {
            closeSocket();
        };
    }, [userId, closeSocket, connectWs]);

    const markRead = useCallback(
        async (id: number) => {
            await markReadMutation.mutateAsync(id);
            await unreadCountQuery.refresh();
        },
        [markReadMutation, unreadCountQuery],
    );

    const markAllRead = useCallback(async () => {
        await markAllReadMutation.mutateAsync();
        qc.setQueryData<{ count: number }>(queryKeys.notifications.unreadCount(), { count: 0 });
    }, [markAllReadMutation, qc]);

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
                unreadCount,
                chatUnreadCount,
                liveGamesCount,
                markRead,
                markAllRead,
                addWSListener,
                sendWSMessage,
                wsEpoch,
            }}
        >
            {children}
        </NotificationContext.Provider>
    );
}
