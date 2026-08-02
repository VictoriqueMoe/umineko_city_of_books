import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { NotificationBell } from "./NotificationBell";

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));

vi.mock("react-router", async () => {
    const actual = await vi.importActual<typeof import("react-router")>("react-router");
    return { ...actual, useNavigate: () => navigate };
});

describe("NotificationBell", () => {
    it("shows no badge when nothing is waiting to be read", () => {
        // given
        const unreadCount = 0;

        // when
        renderWithProviders(<NotificationBell />, { notification: { unreadCount } });

        // then
        expect(screen.getByRole("button", { name: "Notifications" })).toHaveTextContent("");
        expect(screen.queryByText("0")).not.toBeInTheDocument();
    });

    it("shows the number of unread notifications", () => {
        // given
        const unreadCount = 7;

        // when
        renderWithProviders(<NotificationBell />, { notification: { unreadCount } });

        // then
        expect(screen.getByText("7")).toBeInTheDocument();
    });

    it("shows the exact count at the ninety nine boundary", () => {
        // given
        const unreadCount = 99;

        // when
        renderWithProviders(<NotificationBell />, { notification: { unreadCount } });

        // then
        expect(screen.getByText("99")).toBeInTheDocument();
    });

    it("caps the badge once there are more than ninety nine unread", () => {
        // given
        const unreadCount = 4321;

        // when
        renderWithProviders(<NotificationBell />, { notification: { unreadCount } });

        // then
        expect(screen.getByText("99+")).toBeInTheDocument();
        expect(screen.queryByText("4321")).not.toBeInTheDocument();
    });

    it("opens the notifications page when the bell is pressed", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<NotificationBell />, { notification: { unreadCount: 3 } });

        // when
        await user.click(screen.getByRole("button", { name: "Notifications" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/notifications");
    });
});
