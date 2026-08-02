import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { BoardToken } from "./SnakesLaddersBoard";
import { SnakesLaddersBoard } from "./SnakesLaddersBoard";

const tokens: BoardToken[] = [
    { color: "#c0392b", ring: "#f1c40f", initial: "B" },
    { color: "#2e6db4", ring: "#f6e7a8", initial: "E" },
];

function groupOf(initial: string): Element {
    const label = screen.getByText(initial);
    const group = label.closest("g");
    if (!group) {
        throw new Error(`no group around token ${initial}`);
    }
    return group;
}

describe("SnakesLaddersBoard", () => {
    it("presents the whole board as one labelled picture", () => {
        // given
        const positions = [0, 0];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(screen.getByRole("img", { name: "Snakes and ladders board" })).toBeInTheDocument();
    });

    it("numbers every square from the first to the hundredth", () => {
        // given
        const positions = [0, 0];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(screen.getByText("1")).toBeInTheDocument();
        expect(screen.getByText("57")).toBeInTheDocument();
        expect(screen.getByText("100")).toBeInTheDocument();
    });

    it("stars the winning square", () => {
        // given
        const positions = [0, 0];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(screen.getByText("★")).toBeInTheDocument();
    });

    it("gives every player their own token", () => {
        // given
        const positions = [1, 4];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(screen.getByText("B")).toBeInTheDocument();
        expect(screen.getByText("E")).toBeInTheDocument();
    });

    it("parks a player who has not started on the start line", () => {
        // given
        const positions = [0, 1];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(groupOf("B").getAttribute("style")).toContain("translate(486.4px, 1042px)");
    });

    it("stands a token in the middle of the square it sits on", () => {
        // given
        const positions = [1, 0];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(groupOf("B").getAttribute("style")).toContain("translate(41.5px, 956px)");
    });

    it("winds the token back along the snaking rows", () => {
        // given
        const positions = [11, 0];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(groupOf("B").getAttribute("style")).toContain("translate(941.5px, 856px)");
    });

    it("draws nothing for a position that has no token behind it", () => {
        // given
        const positions = [1, 2, 3];

        // when
        renderWithProviders(<SnakesLaddersBoard positions={positions} tokens={tokens} />);

        // then
        expect(screen.getAllByText(/^[BE]$/)).toHaveLength(2);
    });

    it("rings the square the last roll finished on", () => {
        // given
        const lastTo = 20;

        // when
        const { container } = renderWithProviders(
            <SnakesLaddersBoard positions={[20, 0]} tokens={tokens} lastTo={lastTo} />,
        );

        // then
        const ring = container.querySelector("rect[stroke='#d4af37']");
        expect(ring).not.toBeNull();
        expect(ring).toHaveAttribute("x", "3");
        expect(ring).toHaveAttribute("y", "803");
    });

    it("rings nothing before anyone has rolled", () => {
        // given
        const lastTo = null;

        // when
        const { container } = renderWithProviders(
            <SnakesLaddersBoard positions={[0, 0]} tokens={tokens} lastTo={lastTo} />,
        );

        // then
        expect(container.querySelector("rect[stroke='#d4af37']")).toBeNull();
    });
});
