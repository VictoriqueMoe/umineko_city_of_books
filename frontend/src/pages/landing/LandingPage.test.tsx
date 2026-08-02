import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { LandingPage } from "./LandingPage";

vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: (props: { page: string }) => <div>{`rules for ${props.page}`}</div>,
}));

vi.mock("./LiveActivity", () => ({
    LiveActivity: () => <div data-testid="live-activity" />,
}));

describe("LandingPage", () => {
    it("greets the visitor with the site's own name", () => {
        // given
        const siteInfo = { site_name: "Rokkenjima Salon" };

        // when
        renderWithProviders(<LandingPage />, { siteInfo });

        // then
        expect(screen.getByRole("heading", { level: 1, name: "Rokkenjima Salon" })).toBeInTheDocument();
    });

    it("shows the rules that belong to the landing page", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(screen.getByText("rules for landing")).toBeInTheDocument();
    });

    it("invites a signed out visitor to sign in or peek at the board", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(screen.getByRole("link", { name: "Sign in to Play" })).toHaveAttribute("href", "/login");
        expect(screen.getByRole("link", { name: "Peek at the Board" })).toHaveAttribute("href", "/game-board");
    });

    it("closes with a second invitation for a signed out visitor", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(screen.getByRole("heading", { name: "Ready to sit at the table?" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Sign in or Register" })).toHaveAttribute("href", "/login");
    });

    it("sends a signed in member straight to the game board", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(screen.getByRole("link", { name: "Enter the Game Board" })).toHaveAttribute("href", "/game-board");
        expect(screen.queryByRole("link", { name: "Sign in to Play" })).not.toBeInTheDocument();
    });

    it("drops the closing sign-up pitch once the member is signed in", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(screen.queryByRole("heading", { name: "Ready to sit at the table?" })).not.toBeInTheDocument();
        expect(screen.queryByRole("link", { name: "Sign in or Register" })).not.toBeInTheDocument();
    });

    it("explains all four colours of truth", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(screen.getByText("RED TRUTH")).toBeInTheDocument();
        expect(screen.getByText("BLUE TRUTH")).toBeInTheDocument();
        expect(screen.getByText("GOLD TRUTH")).toBeInTheDocument();
        expect(screen.getByText("PURPLE TRUTH")).toBeInTheDocument();
    });

    it("offers a seat at every part of the site", () => {
        // given
        const expected = [
            "/game-board",
            "/theories",
            "/mysteries",
            "/ships",
            "/fanfiction",
            "/gallery",
            "/journals",
            "/rooms",
        ];

        // when
        renderWithProviders(<LandingPage />, { user: makeUser() });

        // then
        const cards = screen.getAllByRole("heading", { level: 3 });
        expect(cards).toHaveLength(expected.length);
        for (const to of expected) {
            expect(document.querySelector(`a[href="${to}"]`)).not.toBeNull();
        }
    });

    it("makes room for the live activity board", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(screen.getByTestId("live-activity")).toBeInTheDocument();
    });

    it("titles the browser tab as a welcome", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<LandingPage />, { user });

        // then
        expect(document.title).toContain("Welcome");
    });
});
