import { createContext } from "react";

export interface NotificationContextValue {
    unreadCount: number;
    chatUnreadCount: number;
    liveGamesCount: number;
    liveStreamsCount: number;
    markRead: (id: number) => Promise<void>;
    markAllRead: () => Promise<void>;
}

export const NotificationContext = createContext<NotificationContextValue | null>(null);
