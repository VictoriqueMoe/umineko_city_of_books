import {createContext} from "react";
import type {Notification} from "../types/api";

export interface NotificationContextValue {
    notifications: Notification[];
    unreadCount: number;
    loading: boolean;
    markRead: (id: number) => Promise<void>;
    markAllRead: () => Promise<void>;
    refreshNotifications: () => Promise<void>;
}

export const NotificationContext = createContext<NotificationContextValue | null>(null);
