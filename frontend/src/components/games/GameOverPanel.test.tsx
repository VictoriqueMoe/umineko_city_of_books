import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { GameOverPanel } from "./GameOverPanel";

describe("GameOverPanel", () => {
    it("renders nothing while the game is running and there is nothing to offer", () => {
        // given
        const isOver = false;

        // when
        const { container } = renderWithProviders(
            <GameOverPanel isOver={isOver} showChildren={false} resultText="You won" resultTone="win">
                <button type="button">Rematch</button>
            </GameOverPanel>,
        );

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("announces a win", () => {
        // given
        const resultText = "You won";

        // when
        renderWithProviders(<GameOverPanel isOver showChildren={false} resultText={resultText} resultTone="win" />);

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
    });

    it("announces a loss", () => {
        // given
        const resultText = "You lost";

        // when
        renderWithProviders(<GameOverPanel isOver showChildren={false} resultText={resultText} resultTone="loss" />);

        // then
        expect(screen.getByText("You lost")).toBeInTheDocument();
    });

    it("announces a draw", () => {
        // given
        const resultText = "Draw";

        // when
        renderWithProviders(<GameOverPanel isOver showChildren={false} resultText={resultText} resultTone="draw" />);

        // then
        expect(screen.getByText("Draw")).toBeInTheDocument();
    });

    it("announces a spectated result with no tone of its own", () => {
        // given
        const resultText = "Beatrice won";

        // when
        renderWithProviders(<GameOverPanel isOver showChildren={false} resultText={resultText} resultTone="neutral" />);

        // then
        expect(screen.getByText("Beatrice won")).toBeInTheDocument();
    });

    it("adds the reason beside the result", () => {
        // given
        const reasonText = "by forfeit";

        // when
        renderWithProviders(
            <GameOverPanel isOver showChildren={false} resultText="You won" resultTone="win" reasonText={reasonText} />,
        );

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("by forfeit")).toBeInTheDocument();
    });

    it("leaves out the reason when none is given", () => {
        // given
        const reasonText = undefined;

        // when
        const { container } = renderWithProviders(
            <GameOverPanel isOver showChildren={false} resultText="Draw" resultTone="draw" reasonText={reasonText} />,
        );

        // then
        expect(container.textContent).toBe("Draw");
    });

    it("leaves out an empty reason", () => {
        // given
        const reasonText = "";

        // when
        const { container } = renderWithProviders(
            <GameOverPanel isOver showChildren={false} resultText="Draw" resultTone="draw" reasonText={reasonText} />,
        );

        // then
        expect(container.textContent).toBe("Draw");
    });

    it("offers the children before the game is over", () => {
        // given
        const isOver = false;

        // when
        renderWithProviders(
            <GameOverPanel isOver={isOver} showChildren resultText="You won" resultTone="win">
                <button type="button">Offer draw</button>
            </GameOverPanel>,
        );

        // then
        expect(screen.getByRole("button", { name: "Offer draw" })).toBeInTheDocument();
        expect(screen.queryByText("You won")).not.toBeInTheDocument();
    });

    it("holds the children back when they are not wanted", () => {
        // given
        const showChildren = false;

        // when
        renderWithProviders(
            <GameOverPanel isOver showChildren={showChildren} resultText="You lost" resultTone="loss">
                <button type="button">Rematch</button>
            </GameOverPanel>,
        );

        // then
        expect(screen.getByText("You lost")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Rematch" })).not.toBeInTheDocument();
    });

    it("shows the result and the children together once the game is over", () => {
        // given
        const resultText = "You won";

        // when
        renderWithProviders(
            <GameOverPanel isOver showChildren resultText={resultText} resultTone="win" reasonText="by resignation">
                <button type="button">Rematch</button>
            </GameOverPanel>,
        );

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("by resignation")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Rematch" })).toBeInTheDocument();
    });
});
