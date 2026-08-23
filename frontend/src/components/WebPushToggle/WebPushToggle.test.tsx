import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WebPushConfig } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { WebPushToggle } from "./WebPushToggle";

const { webPush } = vi.hoisted(() => ({
    webPush: {
        enableWebPush: vi.fn(),
        disableWebPush: vi.fn(),
        webPushConfigured: vi.fn(),
        webPushEnabled: vi.fn(),
        webPushSupported: vi.fn(),
    },
}));

vi.mock("../../utils/webPush", () => webPush);

const config: WebPushConfig = {
    vapid_key: "vapid",
    api_key: "api",
    project_id: "project",
    sender_id: "sender",
    app_id: "app",
};

function renderToggle(pushEnabled = true) {
    return renderWithProviders(<WebPushToggle />, { siteInfo: { push_enabled: pushEnabled, web_push: config } });
}

beforeEach(() => {
    webPush.webPushConfigured.mockReturnValue(true);
    webPush.webPushSupported.mockReturnValue(true);
    webPush.webPushEnabled.mockReturnValue(false);
    webPush.enableWebPush.mockResolvedValue(undefined);
    webPush.disableWebPush.mockResolvedValue(undefined);
});

describe("WebPushToggle when the site and browser both support web push", () => {
    it("offers the toggle in the off position", () => {
        // given
        renderToggle();

        // when
        const toggle = screen.getByRole("switch", { name: "Push Notifications" });

        // then
        expect(toggle).toHaveAttribute("aria-checked", "false");
    });

    it("subscribes this device when switched on", async () => {
        // given
        const user = userEvent.setup();
        renderToggle();

        // when
        await user.click(screen.getByRole("switch", { name: "Push Notifications" }));

        // then
        expect(webPush.enableWebPush).toHaveBeenCalledWith(config);
        await waitFor(() => {
            expect(screen.getByRole("switch", { name: "Push Notifications" })).toHaveAttribute("aria-checked", "true");
        });
    });

    it("unsubscribes this device when switched off", async () => {
        // given
        webPush.webPushEnabled.mockReturnValue(true);
        const user = userEvent.setup();
        renderToggle();

        // when
        await user.click(screen.getByRole("switch", { name: "Push Notifications" }));

        // then
        expect(webPush.disableWebPush).toHaveBeenCalledWith(config);
        await waitFor(() => {
            expect(screen.getByRole("switch", { name: "Push Notifications" })).toHaveAttribute("aria-checked", "false");
        });
    });

    it("explains the failure and stays off when the browser blocks notifications", async () => {
        // given
        webPush.enableWebPush.mockRejectedValue(new Error("notification permission was not granted"));
        const user = userEvent.setup();
        renderToggle();

        // when
        await user.click(screen.getByRole("switch", { name: "Push Notifications" }));

        // then
        expect(await screen.findByText(/Your browser may have blocked them/)).toBeInTheDocument();
        expect(screen.getByRole("switch", { name: "Push Notifications" })).toHaveAttribute("aria-checked", "false");
    });
});

describe("WebPushToggle where web push cannot work", () => {
    it("stays hidden when the site has no web push keys configured", () => {
        // given
        webPush.webPushConfigured.mockReturnValue(false);

        // when
        renderToggle();

        // then
        expect(screen.queryByRole("switch", { name: "Push Notifications" })).toBeNull();
    });

    it("stays hidden in a browser without push support, such as an iOS Safari tab", () => {
        // given
        webPush.webPushSupported.mockReturnValue(false);

        // when
        renderToggle();

        // then
        expect(screen.queryByRole("switch", { name: "Push Notifications" })).toBeNull();
    });

    it("stays hidden when an admin has push switched off, so nobody subscribes to silence", () => {
        // given
        const pushDisabled = false;

        // when
        renderToggle(pushDisabled);

        // then
        expect(screen.queryByRole("switch", { name: "Push Notifications" })).toBeNull();
    });
});
