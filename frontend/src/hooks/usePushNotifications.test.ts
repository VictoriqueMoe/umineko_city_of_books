import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfo, UserProfile, WebPushConfig } from "../types/api";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import { usePushNotifications } from "./usePushNotifications";

const { desktop, native, webPush, deviceTokens } = vi.hoisted(() => ({
    desktop: { ensureNotificationPermission: vi.fn() },
    native: { initPush: vi.fn() },
    webPush: {
        initWebPushRouting: vi.fn(),
        resumeWebPush: vi.fn(),
        webPushConfigured: vi.fn(),
    },
    deviceTokens: { registerToken: vi.fn(), unregisterToken: vi.fn() },
}));

vi.mock("../platform/desktopNotifications", () => desktop);
vi.mock("../platform/pushNative", () => native);
vi.mock("../platform/webPush", () => webPush);
vi.mock("./mutations/device", () => ({ useDeviceTokens: () => deviceTokens }));

const member = makeUser({ id: "user-1", username: "battler" });

const keys: WebPushConfig = {
    vapid_key: "vapid",
    api_key: "api",
    project_id: "project",
    sender_id: "sender",
    app_id: "app",
};

function renderPush(options: { user?: UserProfile | null; siteInfo?: Partial<SiteInfo> } = {}) {
    return renderHook(() => usePushNotifications(), {
        wrapper: providerWrapper({ user: options.user ?? null, siteInfo: options.siteInfo }),
    });
}

beforeEach(() => {
    desktop.ensureNotificationPermission.mockResolvedValue(undefined);
    native.initPush.mockResolvedValue(undefined);
    webPush.resumeWebPush.mockResolvedValue(undefined);
    webPush.initWebPushRouting.mockReturnValue(() => {});
    webPush.webPushConfigured.mockReturnValue(false);
});

describe("usePushNotifications", () => {
    it("asks for notification permission and wires up push once somebody is signed in", () => {
        // given
        const signedIn = member;

        // when
        renderPush({ user: signedIn });

        // then
        expect(desktop.ensureNotificationPermission).toHaveBeenCalledOnce();
        expect(native.initPush).toHaveBeenCalledOnce();
    });

    it("leaves notifications alone for a signed out visitor", () => {
        // given
        const visitor = null;

        // when
        renderPush({ user: visitor });

        // then
        expect(desktop.ensureNotificationPermission).not.toHaveBeenCalled();
        expect(native.initPush).not.toHaveBeenCalled();
    });

    it("hands the native push bootstrap the device token registration it may not import itself", () => {
        // given
        const signedIn = member;

        // when
        renderPush({ user: signedIn });

        // then
        expect(native.initPush).toHaveBeenCalledWith({
            navigate: expect.any(Function),
            registerToken: deviceTokens.registerToken,
        });
    });

    it("resumes web push when the site has push keys configured", () => {
        // given
        webPush.webPushConfigured.mockReturnValue(true);

        // when
        renderPush({ user: member, siteInfo: { web_push: keys } });

        // then
        expect(webPush.resumeWebPush).toHaveBeenCalledWith(keys, deviceTokens);
    });

    it("leaves web push alone when the site has no push keys", () => {
        // given
        webPush.webPushConfigured.mockReturnValue(false);

        // when
        renderPush({ user: member });

        // then
        expect(webPush.resumeWebPush).not.toHaveBeenCalled();
    });

    it("registers push once when the site info arrives as a fresh object saying the same thing", () => {
        // given
        const { rerender } = renderPush({ user: member });

        // when
        rerender();
        rerender();

        // then
        expect(native.initPush).toHaveBeenCalledOnce();
    });

    it("stops listening for a push notification tap when it unmounts", () => {
        // given
        const stopRouting = vi.fn();
        webPush.initWebPushRouting.mockReturnValue(stopRouting);

        // when
        const { unmount } = renderPush({ user: member });
        unmount();

        // then
        expect(stopRouting).toHaveBeenCalledOnce();
    });
});
