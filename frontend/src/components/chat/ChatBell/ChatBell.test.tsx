import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useLocation } from "react-router";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { ChatBell } from "./ChatBell";

function LocationProbe() {
    const location = useLocation();

    return <span>path:{location.pathname}</span>;
}

function renderBell(chatUnreadCount: number) {
    return renderWithProviders(
        <>
            <ChatBell />
            <LocationProbe />
        </>,
        { notification: { chatUnreadCount } },
    );
}

describe("ChatBell", () => {
    it("shows no badge while there is nothing unread", () => {
        // given
        const chatUnreadCount = 0;

        // when
        renderBell(chatUnreadCount);

        // then
        expect(screen.getByRole("button", { name: "Direct messages" })).toBeInTheDocument();
        expect(screen.queryByText("0")).not.toBeInTheDocument();
    });

    it("badges the exact number of unread conversations", () => {
        // given
        const chatUnreadCount = 7;

        // when
        renderBell(chatUnreadCount);

        // then
        expect(screen.getByText("7")).toBeInTheDocument();
    });

    it("still shows the exact count at the ninety nine boundary", () => {
        // given
        const chatUnreadCount = 99;

        // when
        renderBell(chatUnreadCount);

        // then
        expect(screen.getByText("99")).toBeInTheDocument();
    });

    it("caps the badge once there are more than ninety nine unread", () => {
        // given
        const chatUnreadCount = 100;

        // when
        renderBell(chatUnreadCount);

        // then
        expect(screen.getByText("99+")).toBeInTheDocument();
    });

    it("navigates to the chat page when pressed", async () => {
        // given
        const user = userEvent.setup();
        renderBell(0);
        expect(screen.getByText("path:/")).toBeInTheDocument();

        // when
        await user.click(screen.getByRole("button", { name: "Direct messages" }));

        // then
        expect(screen.getByText("path:/chat")).toBeInTheDocument();
    });
});
