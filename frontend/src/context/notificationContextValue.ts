import { createContext } from "react";
import { REALTIME_EVENTS, type RealtimeEventName } from "../api/realtime/events";
import type { WSMessage } from "../types/api";

export type WSMessageHandler = (msg: WSMessage) => void;

export const SHIM_LISTENER_EVENTS: readonly RealtimeEventName[] = Object.values(REALTIME_EVENTS).filter(
    name => name !== REALTIME_EVENTS.PONG,
);

export interface NotificationContextValue {
    unreadCount: number;
    chatUnreadCount: number;
    liveGamesCount: number;
    liveStreamsCount: number;
    markRead: (id: number) => Promise<void>;
    markAllRead: () => Promise<void>;
    addWSListener: (handler: WSMessageHandler) => () => void;
    sendWSMessage: (msg: object) => void;
    wsEpoch: number;
}

export const NotificationContext = createContext<NotificationContextValue | null>(null);
