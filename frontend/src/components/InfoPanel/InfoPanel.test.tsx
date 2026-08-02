import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { InfoPanel } from "./InfoPanel";

describe("InfoPanel", () => {
    it("presents its title as a heading", () => {
        // given
        const title = "House rules";

        // when
        renderWithProviders(
            <InfoPanel title={title}>
                <p>Be kind to the goats</p>
            </InfoPanel>,
        );

        // then
        expect(screen.getByRole("heading", { name: title })).toBeInTheDocument();
    });

    it("renders its children below the heading", () => {
        // given
        const body = "Be kind to the goats";

        // when
        renderWithProviders(
            <InfoPanel title="House rules">
                <p>{body}</p>
            </InfoPanel>,
        );

        // then
        const heading = screen.getByRole("heading", { name: "House rules" });
        const paragraph = screen.getByText(body);
        expect(paragraph).toBeInTheDocument();
        expect(heading.compareDocumentPosition(paragraph) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    });

    it("renders several children in the order they were given", () => {
        // given
        const items = ["first clue", "second clue"];

        // when
        renderWithProviders(
            <InfoPanel title="Clues">
                <p>{items[0]}</p>
                <p>{items[1]}</p>
            </InfoPanel>,
        );

        // then
        const paragraphs = screen.getAllByText(/clue$/);
        expect(paragraphs.map(node => node.textContent)).toEqual(items);
    });

    it("renders interactive children that keep working", () => {
        // given
        const markup = (
            <InfoPanel title="Clues">
                <a href="/rules">Read the rules</a>
            </InfoPanel>
        );

        // when
        renderWithProviders(markup);

        // then
        expect(screen.getByRole("link", { name: "Read the rules" })).toHaveAttribute("href", "/rules");
    });
});
