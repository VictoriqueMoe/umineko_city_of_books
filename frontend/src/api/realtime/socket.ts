import { getAuthToken } from "../authToken";
import { isNativeApp } from "../../platform/capabilities";

type FrameHandler = (raw: string) => void;

type StatusHandler = () => void;

type SocketState = {
    sessionKey: string | null;
    refCount: number;
    socket: WebSocket | null;
    backoff: number;
    reconnectTimer: ReturnType<typeof setTimeout> | null;
    keepaliveTimer: ReturnType<typeof setInterval> | null;
    probeTimer: ReturnType<typeof setTimeout> | null;
    lastMessageAt: number;
    visibilityBound: boolean;
};

const MAX_BACKOFF = 30000;
const KEEPALIVE_INTERVAL_MS = 20_000;
const STALE_THRESHOLD_MS = 90_000;
const PROBE_TIMEOUT_MS = 5_000;

const state: SocketState = {
    sessionKey: null,
    refCount: 0,
    socket: null,
    backoff: 1000,
    reconnectTimer: null,
    keepaliveTimer: null,
    probeTimer: null,
    lastMessageAt: 0,
    visibilityBound: false,
};

const frameHandlers = new Set<FrameHandler>();
const statusHandlers = new Set<StatusHandler>();

function buildSocketUrl(): string {
    const apiBase = import.meta.env.VITE_API_BASE ?? "";
    const httpOrigin = apiBase || window.location.origin;
    const wsOrigin = httpOrigin.replace(/^http/, "ws");
    const wsUrl = `${wsOrigin}/api/v1/ws`;

    const token = getAuthToken();
    const cookieWillBeSent = !isNativeApp() && httpOrigin === window.location.origin;

    if (token && !cookieWillBeSent) {
        return `${wsUrl}?token=${encodeURIComponent(token)}`;
    }

    return wsUrl;
}

function clearReconnectTimer(): void {
    if (state.reconnectTimer !== null) {
        clearTimeout(state.reconnectTimer);
        state.reconnectTimer = null;
    }
}

function clearKeepaliveTimer(): void {
    if (state.keepaliveTimer !== null) {
        clearInterval(state.keepaliveTimer);
        state.keepaliveTimer = null;
    }
    if (state.probeTimer !== null) {
        clearTimeout(state.probeTimer);
        state.probeTimer = null;
    }
}

function probeSocket(socket: WebSocket): void {
    if (socket.readyState !== WebSocket.OPEN) {
        return;
    }
    if (state.probeTimer !== null) {
        return;
    }

    const probedAt = Date.now();
    socket.send(JSON.stringify({ type: "ping", data: {} }));

    state.probeTimer = setTimeout(() => {
        state.probeTimer = null;
        if (state.socket !== socket) {
            return;
        }
        if (state.lastMessageAt < probedAt) {
            socket.close();
        }
    }, PROBE_TIMEOUT_MS);
}

function closeSocket(): void {
    clearReconnectTimer();
    clearKeepaliveTimer();
    if (state.socket) {
        state.socket.close();
        state.socket = null;
    }
}

function onVisible(): void {
    if (document.visibilityState !== "visible") {
        return;
    }
    const socket = state.socket;
    if (!socket) {
        return;
    }

    probeSocket(socket);
}

function bindVisibility(): void {
    if (state.visibilityBound) {
        return;
    }

    document.addEventListener("visibilitychange", onVisible);
    state.visibilityBound = true;
}

function unbindVisibility(): void {
    if (!state.visibilityBound) {
        return;
    }

    document.removeEventListener("visibilitychange", onVisible);
    state.visibilityBound = false;
}

function connect(): void {
    closeSocket();

    const socket = new WebSocket(buildSocketUrl());
    state.socket = socket;

    socket.onopen = () => {
        state.backoff = 1000;
        state.lastMessageAt = Date.now();
        for (const handler of statusHandlers) {
            handler();
        }

        clearKeepaliveTimer();
        state.keepaliveTimer = setInterval(() => {
            if (state.socket !== socket) {
                return;
            }
            if (!document.hidden && Date.now() - state.lastMessageAt > STALE_THRESHOLD_MS) {
                probeSocket(socket);
                return;
            }
            if (socket.readyState === WebSocket.OPEN) {
                socket.send(JSON.stringify({ type: "ping", data: {} }));
            }
        }, KEEPALIVE_INTERVAL_MS);
    };

    socket.onmessage = event => {
        state.lastMessageAt = Date.now();

        const raw = event.data as string;

        for (const handler of frameHandlers) {
            handler(raw);
        }
    };

    socket.onclose = () => {
        if (state.socket !== socket) {
            return;
        }
        state.socket = null;
        clearKeepaliveTimer();
        if (state.refCount === 0) {
            return;
        }
        const cap = Math.min(state.backoff, MAX_BACKOFF);
        state.backoff = Math.min(cap * 2, MAX_BACKOFF);

        const delay = Math.random() * cap;

        state.reconnectTimer = setTimeout(() => {
            connect();
        }, delay);
    };

    socket.onerror = () => {
        socket.close();
    };
}

export function openRealtimeSocket(sessionKey: string): void {
    if (state.sessionKey !== sessionKey) {
        closeSocket();
        state.sessionKey = sessionKey;
        state.refCount = 0;
    }

    state.refCount += 1;
    bindVisibility();

    if (state.socket === null) {
        connect();
    }
}

export function closeRealtimeSocket(sessionKey: string): void {
    if (state.sessionKey !== sessionKey) {
        return;
    }

    state.refCount -= 1;

    if (state.refCount > 0) {
        return;
    }

    state.refCount = 0;
    state.sessionKey = null;

    unbindVisibility();
    closeSocket();

    state.backoff = 1000;
    state.lastMessageAt = 0;
}

export function sendRaw(payload: string): void {
    if (state.socket && state.socket.readyState === WebSocket.OPEN) {
        state.socket.send(payload);
    }
}

export function onFrame(handler: FrameHandler): () => void {
    frameHandlers.add(handler);

    return () => {
        frameHandlers.delete(handler);
    };
}

export function onStatus(handler: StatusHandler): () => void {
    statusHandlers.add(handler);

    return () => {
        statusHandlers.delete(handler);
    };
}
