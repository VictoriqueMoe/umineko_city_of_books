import { createContext } from "react";
import type { WSMessage } from "../types/api";

export type WSMessageHandler = (msg: WSMessage) => void;

export interface NotificationContextValue {
    unreadCount: number;
    chatUnreadCount: number;
    liveGamesCount: number;
    markRead: (id: number) => Promise<void>;
    markAllRead: () => Promise<void>;
    addWSListener: (handler: WSMessageHandler) => () => void;
    sendWSMessage: (msg: object) => void;
    wsEpoch: number;
}

export const NotificationContext = createContext<NotificationContextValue | null>(null);
