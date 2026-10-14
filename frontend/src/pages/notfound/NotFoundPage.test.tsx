import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SiteInfoSecret } from "../../types/api";
import { makeSiteSecret, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { NotFoundPage } from "./NotFoundPage";

const mocks = vi.hoisted(() => ({
    redirectToNotFound: vi.fn<() => void>(),
    shouldRedirectToNotFound: vi.fn<(pathname: string) => boolean>(),
}));

vi.mock("../../platform/notFoundRedirect", () => ({
    redirectToNotFound: mocks.redirectToNotFound,
    shouldRedirectToNotFound: mocks.shouldRedirectToNotFound,
}));

function makeSecret(overrides: Partial<SiteInfoSecret> = {}): SiteInfoSecret {
    return makeSiteSecret({
        title: "Witch's Epitaph",
        pieces: [{ id: "piece_11", letter: "K", tile: 11 }],
        ...overrides,
    });
}

describe("NotFoundPage", () => {
    beforeEach(() => {
        mocks.redirectToNotFound.mockReset();
        mocks.shouldRedirectToNotFound.mockReset();
        mocks.shouldRedirectToNotFound.mockReturnValue(false);
    });

    it("announces the missing fragment", () => {
        // given
        const ui = <NotFoundPage />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByText("404")).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: "This fragment was never written" })).toBeInTheDocument();
    });

    it("shows nothing while it sends the browser to the page the server answers with a 404", () => {
        // given
        const route = "/egfwergergh";
        mocks.shouldRedirectToNotFound.mockImplementation(pathname => pathname === route);

        // when
        renderWithProviders(<NotFoundPage />, { route });

        // then
        expect(mocks.shouldRedirectToNotFound).toHaveBeenCalledWith(route);
        expect(mocks.redirectToNotFound).toHaveBeenCalledTimes(1);
        expect(screen.queryByText("404")).not.toBeInTheDocument();
    });

    it("offers the way back to the city of books", () => {
        // given
        const ui = <NotFoundPage />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByRole("link", { name: "Back to the City of Books" })).toHaveAttribute("href", "/");
    });

    it("keeps the hidden sparkle away from a signed out visitor", () => {
        // given
        const signedOut = null;

        // when
        renderWithProviders(<NotFoundPage />, { user: signedOut, siteInfo: { listed_secrets: [makeSecret()] } });

        // then
        expect(screen.queryByRole("button", { name: "A curious sparkle" })).not.toBeInTheDocument();
    });

    it("hides a sparkle a signed in member has already collected", () => {
        // given
        const collected = new Set(["piece_11"]);

        // when
        renderWithProviders(<NotFoundPage />, {
            user: makeUser(),
            siteInfo: { listed_secrets: [makeSecret()] },
            theme: { hasSecret: id => collected.has(id) },
        });

        // then
        expect(screen.queryByRole("button", { name: "A curious sparkle" })).not.toBeInTheDocument();
    });

    it("shows the hidden sparkle to a signed in member still on the hunt", () => {
        // given
        const secrets = [makeSecret()];

        // when
        renderWithProviders(<NotFoundPage />, { user: makeUser(), siteInfo: { listed_secrets: secrets } });

        // then
        expect(screen.getByRole("button", { name: "A curious sparkle" })).toBeInTheDocument();
    });
});
