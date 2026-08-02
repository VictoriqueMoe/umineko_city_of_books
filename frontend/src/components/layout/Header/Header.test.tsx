import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import { Header } from "./Header";

function noop() {}

describe("Header", () => {
    it("offers the sign in control and no bells to a signed out visitor", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<Header onToggleSidebar={noop} />, { user });

        // then
        expect(screen.getByRole("button", { name: "Sign In" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Notifications" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Direct messages" })).not.toBeInTheDocument();
    });

    it("swaps the sign in control for the account menu once someone is signed in", () => {
        // given
        const account = makeUser({ display_name: "Bernkastel" });

        // when
        renderWithProviders(<Header onToggleSidebar={noop} />, { user: account });

        // then
        expect(screen.queryByRole("button", { name: "Sign In" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Bernkastel/ })).toBeInTheDocument();
    });

    it("shows both bells to a signed in member", () => {
        // given
        const account = makeUser();

        // when
        renderWithProviders(<Header onToggleSidebar={noop} />, {
            user: account,
            notification: { unreadCount: 2, chatUnreadCount: 5 },
        });

        // then
        expect(screen.getByRole("button", { name: "Notifications" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Direct messages" })).toBeInTheDocument();
        expect(screen.getByText("2")).toBeInTheDocument();
        expect(screen.getByText("5")).toBeInTheDocument();
    });

    it("commits to neither branch while the session is still loading", () => {
        // given
        const auth = { loading: true };

        // when
        renderWithProviders(<Header onToggleSidebar={noop} />, { user: null, auth });

        // then
        expect(screen.queryByRole("button", { name: "Sign In" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Notifications" })).not.toBeInTheDocument();
    });

    it("holds back the bells and the account menu while a known user is still loading", () => {
        // given
        const account = makeUser({ display_name: "Bernkastel" });

        // when
        renderWithProviders(<Header onToggleSidebar={noop} />, { user: account, auth: { loading: true } });

        // then
        expect(screen.queryByRole("button", { name: /Bernkastel/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Direct messages" })).not.toBeInTheDocument();
    });

    it("keeps the search field and the theme selector on show for everyone", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<Header onToggleSidebar={noop} />, { user, auth: { loading: true } });

        // then
        expect(screen.getByPlaceholderText("Search the site...")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Theme/ })).toBeInTheDocument();
    });

    it("asks its owner to toggle the sidebar when the menu button is pressed", async () => {
        // given
        const onToggleSidebar = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<Header onToggleSidebar={onToggleSidebar} />);

        // when
        await user.click(screen.getByRole("button", { name: "Toggle menu" }));

        // then
        expect(onToggleSidebar).toHaveBeenCalledOnce();
    });
});
