import { beforeEach, describe, expect, it, vi } from "vitest";
import type { RegistrationError, Token } from "@capacitor/push-notifications";

const { capacitor } = vi.hoisted(() => ({
    capacitor: { isNativePlatform: vi.fn(), getPlatform: vi.fn() },
}));

const { pushNotifications } = vi.hoisted(() => ({
    pushNotifications: { addListener: vi.fn(), requestPermissions: vi.fn(), register: vi.fn() },
}));

vi.mock("@capacitor/core", () => ({ Capacitor: capacitor }));
vi.mock("@capacitor/push-notifications", () => ({ PushNotifications: pushNotifications }));

type Listeners = {
    registration?: (token: Token) => void;
    registrationError?: (failure: RegistrationError) => void;
};

let listeners: Listeners;

async function loadPushModule(): Promise<typeof import("./pushNative")> {
    vi.resetModules();
    return import("./pushNative");
}

function deps(overrides: Partial<{ navigate: () => void; registerToken: () => Promise<void> }> = {}) {
    return {
        navigate: vi.fn(),
        registerToken: vi.fn(() => Promise.resolve()),
        ...overrides,
    };
}

beforeEach(() => {
    listeners = {};
    capacitor.isNativePlatform.mockReturnValue(true);
    capacitor.getPlatform.mockReturnValue("android");
    pushNotifications.addListener.mockImplementation((event: keyof Listeners, handler: unknown) => {
        listeners[event] = handler as never;
        return Promise.resolve({ remove: () => Promise.resolve() });
    });
    pushNotifications.requestPermissions.mockResolvedValue({ receive: "granted" });
    pushNotifications.register.mockResolvedValue(undefined);
});

describe("routeFromPushData", () => {
    it("routes a new follower to their profile", async () => {
        const { routeFromPushData } = await loadPushModule();

        expect(routeFromPushData({ type: "new_follower", actor_username: "alice" })).toBe("/user/alice");
    });

    it("routes a chat room invite to the room", async () => {
        const { routeFromPushData } = await loadPushModule();

        expect(routeFromPushData({ type: "chat_room_invite", reference_id: "room-1" })).toBe("/rooms/room-1");
    });

    it("returns null when the payload has no type", async () => {
        const { routeFromPushData } = await loadPushModule();

        expect(routeFromPushData({})).toBeNull();
    });
});

describe("initPush", () => {
    it("does nothing in a browser session", async () => {
        // given
        capacitor.isNativePlatform.mockReturnValue(false);
        const { initPush } = await loadPushModule();

        // when
        await initPush(deps());

        // then
        expect(pushNotifications.addListener).not.toHaveBeenCalled();
        expect(pushNotifications.register).not.toHaveBeenCalled();
    });

    it("sends the token the device produced through the injected registration call", async () => {
        // given
        const { initPush } = await loadPushModule();
        const injected = deps();
        await initPush(injected);

        // when
        listeners.registration?.({ value: "token-abc" });

        // then
        expect(injected.registerToken).toHaveBeenCalledWith("token-abc", "android");
    });

    it("reports the reason the push service refused to register this device", async () => {
        // given
        const { initPush } = await loadPushModule();
        const { setPlatformErrorReporter } = await import("./errorReporter");
        const report = vi.fn();
        setPlatformErrorReporter(report);
        await initPush(deps());

        // when
        listeners.registrationError?.({ error: "SERVICE_NOT_AVAILABLE" });

        // then
        expect(report).toHaveBeenCalledWith(expect.objectContaining({ message: "SERVICE_NOT_AVAILABLE" }), {
            source: "native-push",
        });
    });

    it("reports a registration call that the server rejected", async () => {
        // given
        const { initPush } = await loadPushModule();
        const { setPlatformErrorReporter } = await import("./errorReporter");
        const report = vi.fn();
        setPlatformErrorReporter(report);
        const failure = new Error("device endpoint is down");
        await initPush(deps({ registerToken: vi.fn(() => Promise.reject(failure)) }));

        // when
        listeners.registration?.({ value: "token-abc" });
        await vi.waitFor(() => {
            expect(report).toHaveBeenCalled();
        });

        // then
        expect(report).toHaveBeenCalledWith(failure, { source: "native-push" });
    });

    it("does not register with the operating system when permission is refused", async () => {
        // given
        pushNotifications.requestPermissions.mockResolvedValue({ receive: "denied" });
        const { initPush } = await loadPushModule();

        // when
        await initPush(deps());

        // then
        expect(pushNotifications.register).not.toHaveBeenCalled();
    });

    it("registers with the operating system once permission is granted", async () => {
        // given
        const { initPush } = await loadPushModule();

        // when
        await initPush(deps());

        // then
        expect(pushNotifications.register).toHaveBeenCalledOnce();
    });
});
