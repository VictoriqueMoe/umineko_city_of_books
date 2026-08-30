import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WebPushControls } from "../../hooks/useWebPush";
import { renderWithProviders } from "../../test-utils/render";
import { WebPushToggle } from "./WebPushToggle";

const { useWebPush } = vi.hoisted(() => ({ useWebPush: vi.fn() }));

vi.mock("../../hooks/useWebPush", () => ({ useWebPush }));

const setEnabled = vi.fn();

function controls(overrides: Partial<WebPushControls> = {}): WebPushControls {
    return { available: true, enabled: false, busy: false, error: "", setEnabled, ...overrides };
}

beforeEach(() => {
    setEnabled.mockReset();
    useWebPush.mockReturnValue(controls());
});

describe("WebPushToggle when web push is available", () => {
    it("offers the toggle in the off position", () => {
        // given
        renderWithProviders(<WebPushToggle />);

        // when
        const toggle = screen.getByRole("switch", { name: "Push Notifications" });

        // then
        expect(toggle).toHaveAttribute("aria-checked", "false");
    });

    it("shows the toggle on for a device that is already subscribed", () => {
        // given
        useWebPush.mockReturnValue(controls({ enabled: true }));

        // when
        renderWithProviders(<WebPushToggle />);

        // then
        expect(screen.getByRole("switch", { name: "Push Notifications" })).toHaveAttribute("aria-checked", "true");
    });

    it("asks to subscribe this device when switched on", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<WebPushToggle />);

        // when
        await user.click(screen.getByRole("switch", { name: "Push Notifications" }));

        // then
        expect(setEnabled).toHaveBeenCalledWith(true);
    });

    it("asks to unsubscribe this device when switched off", async () => {
        // given
        useWebPush.mockReturnValue(controls({ enabled: true }));
        const user = userEvent.setup();
        renderWithProviders(<WebPushToggle />);

        // when
        await user.click(screen.getByRole("switch", { name: "Push Notifications" }));

        // then
        expect(setEnabled).toHaveBeenCalledWith(false);
    });

    it("surfaces the reason the change did not take", () => {
        // given
        useWebPush.mockReturnValue(controls({ error: "Could not turn notifications on: messaging/unsupported" }));

        // when
        renderWithProviders(<WebPushToggle />);

        // then
        expect(screen.getByText(/messaging\/unsupported/)).toBeInTheDocument();
    });

    it("locks the toggle while the change is still in flight", () => {
        // given
        useWebPush.mockReturnValue(controls({ busy: true }));

        // when
        renderWithProviders(<WebPushToggle />);

        // then
        expect(screen.getByRole("switch", { name: "Push Notifications" })).toBeDisabled();
    });
});

describe("WebPushToggle where web push cannot work", () => {
    it("stays hidden entirely", () => {
        // given
        useWebPush.mockReturnValue(controls({ available: false }));

        // when
        const { container } = renderWithProviders(<WebPushToggle />);

        // then
        expect(container).toBeEmptyDOMElement();
    });
});
