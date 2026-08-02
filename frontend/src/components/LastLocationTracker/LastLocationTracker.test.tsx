import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Link } from "react-router";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { LastLocationTracker } from "./LastLocationTracker";

const { recordLocation } = vi.hoisted(() => ({ recordLocation: vi.fn() }));

vi.mock("../../utils/lastLocation", () => ({ recordLocation }));

describe("LastLocationTracker", () => {
    it("renders nothing of its own", () => {
        // given
        const route = "/theories";

        // when
        const { container } = renderWithProviders(<LastLocationTracker />, { route });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("records the location the visitor arrives on", () => {
        // given
        const route = "/theories";

        // when
        renderWithProviders(<LastLocationTracker />, { route });

        // then
        expect(recordLocation).toHaveBeenCalledWith("/theories", "", "");
    });

    it("hands the path, query and fragment over separately", () => {
        // given
        const route = "/theory/12?tab=replies#reply-3";

        // when
        renderWithProviders(<LastLocationTracker />, { route });

        // then
        expect(recordLocation).toHaveBeenCalledWith("/theory/12", "?tab=replies", "#reply-3");
    });

    it("records the new location after the visitor navigates", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(
            <>
                <LastLocationTracker />
                <Link to="/ships?sort=new#top">go to ships</Link>
            </>,
            { route: "/theories" },
        );

        // when
        await user.click(screen.getByRole("link", { name: "go to ships" }));

        // then
        expect(recordLocation).toHaveBeenCalledTimes(2);
        expect(recordLocation).toHaveBeenLastCalledWith("/ships", "?sort=new", "#top");
    });

    it("records nothing further when the tree re-renders on the same location", () => {
        // given
        const { rerender } = renderWithProviders(<LastLocationTracker />, { route: "/theories" });

        // when
        rerender(<LastLocationTracker />);

        // then
        expect(recordLocation).toHaveBeenCalledOnce();
    });
});
