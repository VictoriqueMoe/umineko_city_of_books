import type { WebPushConfig } from "../types/api";
import { routeFromPushData } from "./pushNative";

const SW_PATH = "/push-sw.js";
const FID_KEY = "web_push_fid";
const PLATFORM_WEB = "web";

type NavigateFn = (path: string) => void;

export interface WebPushDeps {
    registerToken: (token: string, platform: string) => Promise<void>;
    unregisterToken: (token: string) => Promise<void>;
}

let latestFid = "";
let stopRegistered: (() => void) | null = null;
let stopUnregistered: (() => void) | null = null;

function readFid(): string | null {
    try {
        return window.localStorage.getItem(FID_KEY);
    } catch {
        return null;
    }
}

function storeFid(fid: string): void {
    try {
        window.localStorage.setItem(FID_KEY, fid);
    } catch {
        return;
    }
}

function clearFid(): void {
    try {
        window.localStorage.removeItem(FID_KEY);
    } catch {
        return;
    }
}

export function webPushConfigured(config: WebPushConfig | undefined): boolean {
    if (!config) {
        return false;
    }

    return Boolean(config.vapid_key && config.api_key && config.project_id && config.sender_id && config.app_id);
}

export function webPushSupported(): boolean {
    return "serviceWorker" in navigator && "PushManager" in window && "Notification" in window;
}

export function webPushEnabled(): boolean {
    if (!webPushSupported()) {
        return false;
    }

    return readFid() !== null && Notification.permission === "granted";
}

async function messagingHandles(config: WebPushConfig) {
    const registration = await navigator.serviceWorker.register(SW_PATH);
    const [{ initializeApp, getApps }, { getMessaging }] = await Promise.all([
        import("firebase/app"),
        import("firebase/messaging"),
    ]);

    const existing = getApps();
    const app =
        existing.length > 0
            ? existing[0]
            : initializeApp({
                  apiKey: config.api_key,
                  projectId: config.project_id,
                  messagingSenderId: config.sender_id,
                  appId: config.app_id,
              });

    return { registration, messaging: getMessaging(app) };
}

async function attachHandlers(config: WebPushConfig, { registerToken, unregisterToken }: WebPushDeps) {
    const { registration, messaging } = await messagingHandles(config);
    const { onRegistered, onUnregistered } = await import("firebase/messaging");

    if (!stopRegistered) {
        stopRegistered = onRegistered(messaging, fid => {
            latestFid = fid;
            storeFid(fid);
            registerToken(fid, PLATFORM_WEB).catch(() => {});
        });
    }

    if (!stopUnregistered) {
        stopUnregistered = onUnregistered(messaging, fid => {
            clearFid();
            unregisterToken(fid).catch(() => {});
        });
    }

    return { registration, messaging };
}

function detachHandlers(): void {
    if (stopRegistered) {
        stopRegistered();
        stopRegistered = null;
    }

    if (stopUnregistered) {
        stopUnregistered();
        stopUnregistered = null;
    }
}

export async function enableWebPush(config: WebPushConfig, deps: WebPushDeps): Promise<void> {
    const permission = await Notification.requestPermission();

    if (permission !== "granted") {
        throw new Error("notification permission was not granted");
    }

    const { registration, messaging } = await attachHandlers(config, deps);
    const { register } = await import("firebase/messaging");

    latestFid = "";

    await register(messaging, {
        vapidKey: config.vapid_key,
        serviceWorkerRegistration: registration,
    });

    if (!latestFid) {
        throw new Error("push service did not return an installation id");
    }

    await deps.registerToken(latestFid, PLATFORM_WEB);
    storeFid(latestFid);
}

export async function disableWebPush(config: WebPushConfig, deps: WebPushDeps): Promise<void> {
    const fid = readFid();

    clearFid();
    latestFid = "";

    const { messaging } = await attachHandlers(config, deps);
    const { unregister } = await import("firebase/messaging");

    await unregister(messaging).catch(() => {});

    if (fid) {
        await deps.unregisterToken(fid).catch(() => {});
    }

    detachHandlers();
}

export async function resumeWebPush(config: WebPushConfig, deps: WebPushDeps): Promise<void> {
    if (!webPushConfigured(config) || !webPushEnabled()) {
        return;
    }

    await attachHandlers(config, deps);
}

export function initWebPushRouting(navigate: NavigateFn): () => void {
    if (!("serviceWorker" in navigator)) {
        return () => {};
    }

    function handleMessage(event: MessageEvent) {
        const payload = event.data as { type?: string; data?: Record<string, unknown> } | null;

        if (!payload || payload.type !== "push-notification-click") {
            return;
        }

        const route = routeFromPushData(payload.data ?? {});

        if (route) {
            navigate(route);
        }
    }

    navigator.serviceWorker.addEventListener("message", handleMessage);

    return () => navigator.serviceWorker.removeEventListener("message", handleMessage);
}
