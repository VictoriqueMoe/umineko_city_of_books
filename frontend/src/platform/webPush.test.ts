import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WebPushConfig } from "../types/api";
import {
    disableWebPush,
    enableWebPush,
    initWebPushRouting,
    resumeWebPush,
    webPushConfigured,
    webPushEnabled,
    webPushSupported,
    type WebPushDeps,
} from "./webPush";

const { firebaseApp } = vi.hoisted(() => ({
    firebaseApp: { initializeApp: vi.fn(), getApps: vi.fn() },
}));

const { firebaseMessaging } = vi.hoisted(() => ({
    firebaseMessaging: {
        getMessaging: vi.fn(),
        register: vi.fn(),
        onRegistered: vi.fn(),
        unregister: vi.fn(),
        onUnregistered: vi.fn(),
    },
}));

const { pushRouting } = vi.hoisted(() => ({ pushRouting: { routeFromPushData: vi.fn() } }));

vi.mock("firebase/app", () => firebaseApp);
vi.mock("firebase/messaging", () => firebaseMessaging);
vi.mock("./pushNative", () => pushRouting);

const FID_KEY = "web_push_fid";
const FID = "fid-abc123";

const config: WebPushConfig = {
    vapid_key: "vapid",
    api_key: "api",
    project_id: "project",
    sender_id: "sender",
    app_id: "app",
};

const registration = { scope: "/" };

const registerToken = vi.fn();
const unregisterToken = vi.fn();
const deps: WebPushDeps = { registerToken, unregisterToken };

let serviceWorkerStub: ReturnType<typeof stubServiceWorker>;
let swListeners: Array<(event: MessageEvent) => void>;
let registeredCallback: ((fid: string) => void) | null;
let unregisteredCallback: ((fid: string) => void) | null;

function stubServiceWorker() {
    swListeners = [];
    const serviceWorker = {
        register: vi.fn(() => Promise.resolve(registration)),
        addEventListener: vi.fn((_type: string, listener: (event: MessageEvent) => void) => {
            swListeners.push(listener);
        }),
        removeEventListener: vi.fn((_type: string, listener: (event: MessageEvent) => void) => {
            swListeners = swListeners.filter(entry => entry !== listener);
        }),
    };

    Object.defineProperty(window.navigator, "serviceWorker", { value: serviceWorker, configurable: true });

    return serviceWorker;
}

function stubNotification(permission: NotificationPermission, requested: NotificationPermission = permission) {
    vi.stubGlobal("Notification", {
        permission,
        requestPermission: vi.fn(() => Promise.resolve(requested)),
    });
}

beforeEach(async () => {
    vi.resetModules();
    registeredCallback = null;
    unregisteredCallback = null;
    serviceWorkerStub = stubServiceWorker();
    vi.stubGlobal("PushManager", class {});
    stubNotification("granted");

    firebaseApp.getApps.mockReturnValue([]);
    firebaseApp.initializeApp.mockReturnValue({ name: "app" });
    firebaseMessaging.getMessaging.mockReturnValue({ name: "messaging" });

    firebaseMessaging.onRegistered.mockImplementation((_messaging: unknown, cb: (fid: string) => void) => {
        registeredCallback = cb;

        return () => {
            registeredCallback = null;
        };
    });

    firebaseMessaging.onUnregistered.mockImplementation((_messaging: unknown, cb: (fid: string) => void) => {
        unregisteredCallback = cb;

        return () => {
            unregisteredCallback = null;
        };
    });

    firebaseMessaging.register.mockImplementation(() => {
        if (!registeredCallback) {
            return Promise.reject(new Error("invalid-on-registered-handler"));
        }

        registeredCallback(FID);

        return Promise.resolve();
    });

    firebaseMessaging.unregister.mockImplementation(() => {
        if (unregisteredCallback) {
            unregisteredCallback(FID);
        }

        return Promise.resolve();
    });

    registerToken.mockResolvedValue(undefined);
    unregisterToken.mockResolvedValue(undefined);

    await disableWebPush(config, deps).catch(() => {});
    registerToken.mockClear();
    unregisterToken.mockClear();
    firebaseMessaging.onRegistered.mockClear();
    firebaseMessaging.onUnregistered.mockClear();
    firebaseMessaging.register.mockClear();
    firebaseMessaging.unregister.mockClear();
    registeredCallback = null;
    unregisteredCallback = null;
    window.localStorage.removeItem(FID_KEY);
});

afterEach(() => {
    Reflect.deleteProperty(window.navigator, "serviceWorker");
});

describe("webPushConfigured", () => {
    it("accepts a config with every field filled in", () => {
        // given
        const complete = config;

        // when
        const got = webPushConfigured(complete);

        // then
        expect(got).toBe(true);
    });

    it("rejects a config that is missing any single field", () => {
        // given
        const keys = Object.keys(config) as Array<keyof WebPushConfig>;

        // when
        const results = keys.map(key => webPushConfigured({ ...config, [key]: "" }));

        // then
        expect(results.every(result => result === false)).toBe(true);
    });

    it("rejects a missing config", () => {
        // given
        const missing = undefined;

        // when
        const got = webPushConfigured(missing);

        // then
        expect(got).toBe(false);
    });
});

describe("webPushSupported", () => {
    it("reports support when the browser has service workers and a push manager", () => {
        // given
        const browser = window.navigator;

        // when
        const got = webPushSupported();

        // then
        expect(browser.serviceWorker).toBeDefined();
        expect(got).toBe(true);
    });

    it("reports no support when the browser has no push manager, as in an iOS Safari tab", () => {
        // given
        vi.unstubAllGlobals();
        stubNotification("default");

        // when
        const got = webPushSupported();

        // then
        expect(got).toBe(false);
    });
});

describe("webPushEnabled", () => {
    it("is on when an installation id was stored and permission is granted", () => {
        // given
        window.localStorage.setItem(FID_KEY, FID);

        // when
        const got = webPushEnabled();

        // then
        expect(got).toBe(true);
    });

    it("is off when permission was revoked after the installation id was stored", () => {
        // given
        window.localStorage.setItem(FID_KEY, FID);
        stubNotification("denied");

        // when
        const got = webPushEnabled();

        // then
        expect(got).toBe(false);
    });

    it("is off when nothing was ever registered", () => {
        // given
        window.localStorage.removeItem(FID_KEY);

        // when
        const got = webPushEnabled();

        // then
        expect(got).toBe(false);
    });
});

describe("enableWebPush", () => {
    it("registers the installation id against the account and remembers it for this device", async () => {
        // given
        stubNotification("default", "granted");

        // when
        await enableWebPush(config, deps);

        // then
        expect(firebaseMessaging.register).toHaveBeenCalledWith(
            { name: "messaging" },
            { vapidKey: "vapid", serviceWorkerRegistration: registration },
        );
        expect(registerToken).toHaveBeenCalledWith(FID, "web");
        expect(window.localStorage.getItem(FID_KEY)).toBe(FID);
    });

    it("installs the onRegistered handler before registering, which the SDK requires", async () => {
        // given
        const order: string[] = [];
        firebaseMessaging.onRegistered.mockImplementation((_m: unknown, cb: (fid: string) => void) => {
            order.push("onRegistered");
            registeredCallback = cb;

            return () => {};
        });
        firebaseMessaging.register.mockImplementation(() => {
            order.push("register");
            registeredCallback?.(FID);

            return Promise.resolve();
        });

        // when
        await enableWebPush(config, deps);

        // then
        expect(order).toEqual(["onRegistered", "register"]);
    });

    it("asks for permission before anything else runs, because WebKit revokes the subscription otherwise", async () => {
        // given
        const order: string[] = [];
        vi.stubGlobal("Notification", {
            permission: "default",
            requestPermission: vi.fn(() => {
                order.push("requestPermission");

                return Promise.resolve("granted");
            }),
        });
        serviceWorkerStub.register.mockImplementation(() => {
            order.push("serviceWorker.register");

            return Promise.resolve(registration);
        });

        // when
        await enableWebPush(config, deps);

        // then
        expect(order).toEqual(["requestPermission", "serviceWorker.register"]);
    });

    it("uses our own service worker rather than the one the SDK would pick", async () => {
        // given
        const serviceWorker = window.navigator.serviceWorker;

        // when
        await enableWebPush(config, deps);

        // then
        expect(serviceWorker.register).toHaveBeenCalledWith("/push-sw.js");
    });

    it("gives up without registering anything when permission is refused", async () => {
        // given
        stubNotification("default", "denied");

        // when
        const attempt = enableWebPush(config, deps);

        // then
        await expect(attempt).rejects.toThrow("notification permission was not granted");
        expect(registerToken).not.toHaveBeenCalled();
        expect(window.localStorage.getItem(FID_KEY)).toBeNull();
    });

    it("fails loudly when the SDK registers without ever reporting an installation id", async () => {
        // given
        firebaseMessaging.register.mockImplementation(() => Promise.resolve());

        // when
        const attempt = enableWebPush(config, deps);

        // then
        await expect(attempt).rejects.toThrow("push service did not return an installation id");
    });
});

describe("disableWebPush", () => {
    it("unregisters with the SDK, withdraws the id from the account and forgets it", async () => {
        // given
        await enableWebPush(config, deps);
        unregisterToken.mockClear();

        // when
        await disableWebPush(config, deps);

        // then
        expect(firebaseMessaging.unregister).toHaveBeenCalled();
        expect(unregisterToken).toHaveBeenCalledWith(FID);
        expect(window.localStorage.getItem(FID_KEY)).toBeNull();
    });

    it("still tears down the subscription when nothing was stored", async () => {
        // given
        window.localStorage.removeItem(FID_KEY);

        // when
        await disableWebPush(config, deps);

        // then
        expect(firebaseMessaging.unregister).toHaveBeenCalled();
    });
});

describe("resumeWebPush", () => {
    it("reattaches the handlers so a rotated installation id reaches the backend", async () => {
        // given
        window.localStorage.setItem(FID_KEY, FID);

        // when
        await resumeWebPush(config, deps);
        registeredCallback?.("fid-rotated");

        // then
        expect(registerToken).toHaveBeenCalledWith("fid-rotated", "web");
        expect(window.localStorage.getItem(FID_KEY)).toBe("fid-rotated");
    });

    it("does nothing on a device that never turned push on", async () => {
        // given
        window.localStorage.removeItem(FID_KEY);

        // when
        await resumeWebPush(config, deps);

        // then
        expect(firebaseMessaging.onRegistered).not.toHaveBeenCalled();
    });

    it("does nothing when the site has no web push keys", async () => {
        // given
        window.localStorage.setItem(FID_KEY, FID);

        // when
        await resumeWebPush({ ...config, vapid_key: "" }, deps);

        // then
        expect(firebaseMessaging.onRegistered).not.toHaveBeenCalled();
    });
});

describe("initWebPushRouting", () => {
    it("navigates to the notification target when the service worker reports a click", () => {
        // given
        const navigate = vi.fn();
        pushRouting.routeFromPushData.mockReturnValue("/theory/abc");
        initWebPushRouting(navigate);

        // when
        for (const listener of swListeners) {
            listener({ data: { type: "push-notification-click", data: { type: "reply" } } } as MessageEvent);
        }

        // then
        expect(navigate).toHaveBeenCalledWith("/theory/abc");
    });

    it("ignores service worker messages that are not notification clicks", () => {
        // given
        const navigate = vi.fn();
        initWebPushRouting(navigate);

        // when
        for (const listener of swListeners) {
            listener({ data: { type: "something-else" } } as MessageEvent);
        }

        // then
        expect(navigate).not.toHaveBeenCalled();
    });

    it("stops listening once the caller unsubscribes", () => {
        // given
        const navigate = vi.fn();
        const stop = initWebPushRouting(navigate);

        // when
        stop();

        // then
        expect(swListeners).toHaveLength(0);
    });
});
