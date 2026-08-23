self.addEventListener("install", () => {
    self.skipWaiting();
});

self.addEventListener("activate", event => {
    event.waitUntil(self.clients.claim());
});

function readPayload(event) {
    if (!event.data) {
        return {};
    }

    try {
        return event.data.json() ?? {};
    } catch {
        return {};
    }
}

self.addEventListener("push", event => {
    const payload = readPayload(event);
    const notification = payload.notification ?? {};
    const data = payload.data ?? {};
    const title = notification.title || "City of Books";
    const body = notification.body || "You have a new notification";

    event.waitUntil(
        self.registration.showNotification(title, {
            body,
            icon: "/favicon/android-chrome-192x192.png",
            badge: "/favicon/favicon-32x32.png",
            data,
        }),
    );
});

self.addEventListener("notificationclick", event => {
    event.notification.close();

    const data = event.notification.data ?? {};

    event.waitUntil(
        self.clients.matchAll({ type: "window", includeUncontrolled: true }).then(matched => {
            for (const client of matched) {
                if ("focus" in client) {
                    client.postMessage({ type: "push-notification-click", data });

                    return client.focus();
                }
            }

            return self.clients.openWindow("/notifications");
        }),
    );
});
