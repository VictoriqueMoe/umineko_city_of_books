import { createContext } from "react";
import type { Notification, WSMessage } from "../types/api";

export type WSMessageHandler = (msg: WSMessage) => void;

export interface NotificationContextValue {
    notifications: Notification[];
    unreadCount: number;
    chatUnreadCount: number;
    loading: boolean;
    markRead: (id: number) => Promise<void>;
    markAllRead: () => Promise<void>;
    refreshNotifications: () => Promise<void>;
    addWSListener: (handler: WSMessageHandler) => () => void;
    sendWSMessage: (msg: object) => void;
}

export const NotificationContext = createContext<NotificationContextValue | null>(null);
