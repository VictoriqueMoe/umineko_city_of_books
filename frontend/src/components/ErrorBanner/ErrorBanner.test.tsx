import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { ErrorBanner } from "./ErrorBanner";

describe("ErrorBanner", () => {
    it("shows the message it is given", () => {
        // given
        const message = "The seal could not be broken";

        // when
        renderWithProviders(<ErrorBanner message={message} />);

        // then
        expect(screen.getByText(message)).toBeInTheDocument();
    });

    it("replaces the message when a new failure arrives", () => {
        // given
        const { rerender } = renderWithProviders(<ErrorBanner message="The seal could not be broken" />);

        // when
        rerender(<ErrorBanner message="The letter was rejected" />);

        // then
        expect(screen.getByText("The letter was rejected")).toBeInTheDocument();
        expect(screen.queryByText("The seal could not be broken")).not.toBeInTheDocument();
    });

    it("still renders its container when the message is empty", () => {
        // given
        const message = "";

        // when
        const { container } = renderWithProviders(<ErrorBanner message={message} />);

        // then
        expect(container.firstElementChild).not.toBeNull();
        expect(container).toHaveTextContent("");
    });
});
