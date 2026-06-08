import { Capacitor } from "@capacitor/core";
import { PushNotifications, type Token, type ActionPerformed } from "@capacitor/push-notifications";
import type { Notification } from "../types/api";
import { registerDeviceToken } from "../api/endpoints";
import { getNotificationRoute } from "./notifications";

type NavigateFn = (path: string) => void;

let initialised = false;
let navigateFn: NavigateFn | null = null;

export function routeFromPushData(data: Record<string, unknown>): string | null {
    if (typeof data.type !== "string") {
        return null;
    }

    const notif = {
        type: data.type,
        reference_id: typeof data.reference_id === "string" ? data.reference_id : "",
        reference_type: typeof data.reference_type === "string" ? data.reference_type : "",
        actor: { username: typeof data.actor_username === "string" ? data.actor_username : "" },
    } as Notification;

    return getNotificationRoute(notif);
}

export async function initPush(navigate: NavigateFn): Promise<void> {
    navigateFn = navigate;

    if (!Capacitor.isNativePlatform() || initialised) {
        return;
    }
    initialised = true;

    await PushNotifications.addListener("registration", (token: Token) => {
        registerDeviceToken(token.value, Capacitor.getPlatform()).catch(() => {});
    });

    await PushNotifications.addListener("registrationError", () => {});

    await PushNotifications.addListener("pushNotificationActionPerformed", (action: ActionPerformed) => {
        const data = (action.notification.data ?? {}) as Record<string, unknown>;
        const route = routeFromPushData(data);
        if (route && navigateFn) {
            navigateFn(route);
        }
    });

    const permission = await PushNotifications.requestPermissions();
    if (permission.receive !== "granted") {
        return;
    }

    await PushNotifications.register();
}
