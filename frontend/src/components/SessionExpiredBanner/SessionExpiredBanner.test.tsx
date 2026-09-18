import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { SessionExpiredBanner } from "./SessionExpiredBanner";

const { useSessionLost } = vi.hoisted(() => ({
    useSessionLost: vi.fn(),
}));

vi.mock("../../hooks/useSessionLost", () => ({ useSessionLost }));

beforeEach(() => {
    useSessionLost.mockReturnValue(false);
});

describe("SessionExpiredBanner", () => {
    it("renders nothing while the session is intact", () => {
        // given
        useSessionLost.mockReturnValue(false);

        // when
        const { container } = renderWithProviders(<SessionExpiredBanner />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("tells a reader who has been signed out to reload and sign in again", () => {
        // given
        useSessionLost.mockReturnValue(true);

        // when
        renderWithProviders(<SessionExpiredBanner />);

        // then
        expect(screen.getByRole("alert")).toHaveTextContent(
            "You have been signed out. Copy anything you were writing, then reload the page and sign in again.",
        );
    });

    it("offers a button to reload with", () => {
        // given
        useSessionLost.mockReturnValue(true);

        // when
        renderWithProviders(<SessionExpiredBanner />);

        // then
        expect(screen.getByRole("button", { name: "Reload now" })).toBeInTheDocument();
    });
});
