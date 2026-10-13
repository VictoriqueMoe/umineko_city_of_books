import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { AboutPage } from "./AboutPage";

function pinToday(today: Date): void {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(today);
}

describe("AboutPage", () => {
    it("introduces Victoria with her photo", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        expect(screen.getByRole("heading", { level: 1, name: "Hi, I'm Victoria" })).toBeInTheDocument();
        expect(screen.getByText("I made this place.")).toBeInTheDocument();

        const photo = screen.getByRole("img", { name: "Victoria" });
        expect(photo).toHaveAttribute("src", "/about-creator.webp");
        expect(photo).toHaveAttribute("width", "900");
        expect(photo).toHaveAttribute("height", "1200");
        expect(photo).toHaveAttribute("loading", "eager");
        expect(screen.getByText("My Erika Furudo cosplay")).toBeInTheDocument();
    });

    it("renders every section heading", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        expect(screen.getByRole("heading", { level: 2, name: "Who I am" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 2, name: "Why I built it" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 2, name: "How it was built" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 2, name: "What I love here" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 2, name: "Find me elsewhere" })).toBeInTheDocument();
    });

    it("works out her age from today's date once her birthday has passed", () => {
        // given
        pinToday(new Date(2026, 9, 13, 12, 0, 0));

        // when
        renderWithProviders(<AboutPage />);

        // then
        expect(
            screen.getByText("I'm a German girl living in the UK, born on 4 October 1994, which makes me 32."),
        ).toBeInTheDocument();
    });

    it("works out her age from today's date before her birthday comes round", () => {
        // given
        pinToday(new Date(2026, 9, 3, 12, 0, 0));

        // when
        renderWithProviders(<AboutPage />);

        // then
        expect(
            screen.getByText("I'm a German girl living in the UK, born on 4 October 1994, which makes me 31."),
        ).toBeInTheDocument();
    });

    it("links out to the quote site in a new tab without leaking the opener", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        const link = screen.getByRole("link", { name: "quotes.auaurora.moe" });
        expect(link).toHaveAttribute("href", "https://quotes.auaurora.moe/");
        expect(link).toHaveAttribute("target", "_blank");
        expect(link).toHaveAttribute("rel", "noopener noreferrer");
    });

    it("links the provenance notice to PROVENANCE.md in a new tab without leaking the opener", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        expect(screen.getByText(/^Coded by hand, no AI\./)).toBeInTheDocument();
        const link = screen.getByRole("link", { name: "Here's how" });
        expect(link).toHaveAttribute(
            "href",
            "https://github.com/VictoriqueMoe/umineko_city_of_books/blob/main/PROVENANCE.md",
        );
        expect(link).toHaveAttribute("target", "_blank");
        expect(link).toHaveAttribute("rel", "noopener noreferrer");
    });

    it("links out to each of her socials in a new tab without leaking the opener", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        const socials = screen.getByRole("list", { name: "My socials" });
        expect(socials.children).toHaveLength(4);

        const expected = [
            { name: "X @VictoriqueM", href: "https://x.com/VictoriqueM" },
            { name: "X @FeatherineFAA", href: "https://x.com/FeatherineFAA" },
            { name: "GitHub VictoriqueMoe", href: "https://github.com/VictoriqueMoe" },
        ];
        for (const social of expected) {
            const link = screen.getByRole("link", { name: social.name });
            expect(link).toHaveAttribute("href", social.href);
            expect(link).toHaveAttribute("target", "_blank");
            expect(link).toHaveAttribute("rel", "noopener noreferrer");
        }
    });

    it("shows her Discord username as plain text because Discord has no profile URL", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        const handle = screen.getByText("victorique.moe");
        expect(handle).toBeInTheDocument();
        expect(handle.closest("a")).toBeNull();
    });

    it("links each of her favourite corners to its own page", () => {
        // given
        const user = makeUser();

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        expect(screen.getByRole("link", { name: /^Chat rooms/ })).toHaveAttribute("href", "/rooms");
        expect(screen.getByRole("link", { name: /^Games/ })).toHaveAttribute("href", "/games/live");
        expect(screen.getByRole("link", { name: /^Live streaming from OBS/ })).toHaveAttribute("href", "/live");
        expect(screen.getByRole("link", { name: /^The art people upload/ })).toHaveAttribute("href", "/gallery");
        expect(screen.getByRole("link", { name: /^Mysteries/ })).toHaveAttribute("href", "/mysteries");
        expect(screen.getByRole("link", { name: /^The Game Board/ })).toHaveAttribute("href", "/game-board");
    });

    it("shows the page to visitors who are not signed in", () => {
        // given
        const user = null;

        // when
        renderWithProviders(<AboutPage />, { user });

        // then
        expect(screen.getByRole("heading", { level: 1, name: "Hi, I'm Victoria" })).toBeInTheDocument();
        expect(screen.getByRole("heading", { level: 2, name: "Who I am" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /^Mysteries/ })).toHaveAttribute("href", "/mysteries");
    });
});
