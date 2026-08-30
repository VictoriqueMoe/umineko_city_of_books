import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfo, WebPushConfig } from "../types/api";
import { providerWrapper } from "../test-utils/render";
import { useWebPush } from "./useWebPush";

const { webPush } = vi.hoisted(() => ({
    webPush: {
        enableWebPush: vi.fn(),
        disableWebPush: vi.fn(),
        webPushConfigured: vi.fn(),
        webPushEnabled: vi.fn(),
        webPushSupported: vi.fn(),
    },
}));

const { deviceTokens } = vi.hoisted(() => ({
    deviceTokens: { registerToken: vi.fn(), unregisterToken: vi.fn() },
}));

vi.mock("../platform/webPush", () => webPush);
vi.mock("./mutations/device", () => ({ useDeviceTokens: () => deviceTokens }));

const config: WebPushConfig = {
    vapid_key: "vapid",
    api_key: "api",
    project_id: "project",
    sender_id: "sender",
    app_id: "app",
};

function renderWebPush(siteInfo: Partial<SiteInfo> = {}) {
    return renderHook(() => useWebPush(), {
        wrapper: providerWrapper({ siteInfo: { push_enabled: true, web_push: config, ...siteInfo } }),
    });
}

beforeEach(() => {
    webPush.webPushConfigured.mockReturnValue(true);
    webPush.webPushSupported.mockReturnValue(true);
    webPush.webPushEnabled.mockReturnValue(false);
    webPush.enableWebPush.mockResolvedValue(undefined);
    webPush.disableWebPush.mockResolvedValue(undefined);
});

describe("useWebPush availability", () => {
    it("is available when the site, the browser and the keys all allow it", () => {
        // given
        const { result } = renderWebPush();

        // when
        const available = result.current.available;

        // then
        expect(available).toBe(true);
    });

    it("is unavailable when the site has no web push keys configured", () => {
        // given
        webPush.webPushConfigured.mockReturnValue(false);

        // when
        const { result } = renderWebPush();

        // then
        expect(result.current.available).toBe(false);
    });

    it("is unavailable in a browser without push support, such as an iOS Safari tab", () => {
        // given
        webPush.webPushSupported.mockReturnValue(false);

        // when
        const { result } = renderWebPush();

        // then
        expect(result.current.available).toBe(false);
    });

    it("is unavailable when an admin has push switched off, so nobody subscribes to silence", () => {
        // given
        const pushDisabled = false;

        // when
        const { result } = renderWebPush({ push_enabled: pushDisabled });

        // then
        expect(result.current.available).toBe(false);
    });

    it("starts in the position this device is already in", () => {
        // given
        webPush.webPushEnabled.mockReturnValue(true);

        // when
        const { result } = renderWebPush();

        // then
        expect(result.current.enabled).toBe(true);
    });
});

describe("useWebPush switching on", () => {
    it("subscribes this device, handing the platform the account calls it may not import", async () => {
        // given
        const { result } = renderWebPush();

        // when
        act(() => {
            result.current.setEnabled(true);
        });

        // then
        expect(webPush.enableWebPush).toHaveBeenCalledWith(config, deviceTokens);
        await waitFor(() => {
            expect(result.current.enabled).toBe(true);
        });
    });

    it("surfaces the underlying reason and stays off when subscribing fails", async () => {
        // given
        webPush.enableWebPush.mockRejectedValue(new Error("messaging/unsupported-browser"));
        const { result } = renderWebPush();

        // when
        act(() => {
            result.current.setEnabled(true);
        });

        // then
        await waitFor(() => {
            expect(result.current.error).toBe("Could not turn notifications on: messaging/unsupported-browser");
        });
        expect(result.current.enabled).toBe(false);
    });

    it("says so rather than showing an empty reason when the failure carries no message", async () => {
        // given
        webPush.enableWebPush.mockRejectedValue(new Error(""));
        const { result } = renderWebPush();

        // when
        act(() => {
            result.current.setEnabled(true);
        });

        // then
        await waitFor(() => {
            expect(result.current.error).toBe("Could not turn notifications on: the browser gave no reason");
        });
    });

    it("holds the control busy until the platform has finished", async () => {
        // given
        let finish = () => {};
        webPush.enableWebPush.mockReturnValue(
            new Promise<void>(resolve => {
                finish = resolve;
            }),
        );
        const { result } = renderWebPush();

        // when
        act(() => {
            result.current.setEnabled(true);
        });

        // then
        expect(result.current.busy).toBe(true);
        await act(async () => {
            finish();
        });
        expect(result.current.busy).toBe(false);
    });
});

describe("useWebPush switching off", () => {
    it("unsubscribes this device", async () => {
        // given
        webPush.webPushEnabled.mockReturnValue(true);
        const { result } = renderWebPush();

        // when
        act(() => {
            result.current.setEnabled(false);
        });

        // then
        expect(webPush.disableWebPush).toHaveBeenCalledWith(config, deviceTokens);
        await waitFor(() => {
            expect(result.current.enabled).toBe(false);
        });
    });

    it("surfaces the underlying reason when unsubscribing fails", async () => {
        // given
        webPush.webPushEnabled.mockReturnValue(true);
        webPush.disableWebPush.mockRejectedValue(new Error("messaging/token-unsubscribe-failed"));
        const { result } = renderWebPush();

        // when
        act(() => {
            result.current.setEnabled(false);
        });

        // then
        await waitFor(() => {
            expect(result.current.error).toBe("Could not turn notifications off: messaging/token-unsubscribe-failed");
        });
        expect(result.current.enabled).toBe(true);
    });
});
