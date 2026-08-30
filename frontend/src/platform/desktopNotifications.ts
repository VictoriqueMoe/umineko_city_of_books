import type { Notification } from "../types/api";
import { getNotificationRoute, getNotificationText } from "../domain/notifications";
import type { TurnNotice } from "../domain/games/turnNotice";

const SITE_ICON = "/favicon/android-chrome-192x192.png";

export function isTabHidden(): boolean {
    return document.visibilityState !== "visible" || !document.hasFocus();
}

export function showDesktopNotification(notif: Notification): void {
    if (typeof window === "undefined" || !("Notification" in window)) {
        return;
    }
    if (window.Notification.permission !== "granted") {
        return;
    }
    if (!isTabHidden()) {
        return;
    }
    const actorName = notif.actor?.display_name || "";
    const body = getNotificationText(notif);
    const title = actorName ? `${actorName} ${body}` : body;
    const route = getNotificationRoute(notif);
    const osNotif = new window.Notification(title, {
        body: notif.message || "",
        icon: notif.actor?.avatar_url || SITE_ICON,
        badge: "/favicon/favicon-32x32.png",
        tag: `notif-${notif.id}`,
    });
    osNotif.onclick = () => {
        window.focus();
        window.location.href = route;
        osNotif.close();
    };
}

export function showGameTurnNotification(notice: TurnNotice): void {
    if (typeof window === "undefined" || !("Notification" in window)) {
        return;
    }
    if (window.Notification.permission !== "granted") {
        return;
    }

    const osNotif = new window.Notification(notice.title, {
        body: notice.body,
        icon: SITE_ICON,
        tag: notice.tag,
    });

    osNotif.onclick = () => {
        window.focus();
        window.location.href = notice.route;
        osNotif.close();
    };
}

export async function ensureNotificationPermission(): Promise<boolean> {
    if (typeof window === "undefined" || !("Notification" in window)) {
        return false;
    }
    if (window.Notification.permission === "granted") {
        return true;
    }
    if (window.Notification.permission === "denied") {
        return false;
    }
    try {
        const result = await window.Notification.requestPermission();
        return result === "granted";
    } catch {
        return false;
    }
}
