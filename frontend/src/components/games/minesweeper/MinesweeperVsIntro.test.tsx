import { act, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CHARACTERS } from "../../../games/minesweeper/characters";
import { renderWithProviders } from "../../../test-utils/render";
import { MinesweeperVsIntro } from "./MinesweeperVsIntro";

vi.mock("./MinesweeperLightningCanvas", () => ({
    MinesweeperLightningCanvas: () => <div data-testid="lightning" />,
}));

const bernkastel = CHARACTERS[0];
const erika = CHARACTERS[1];

describe("MinesweeperVsIntro", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it("puts the two witches face to face", () => {
        // given
        const onDone = vi.fn();

        // when
        renderWithProviders(<MinesweeperVsIntro myCharacter={bernkastel} opponentCharacter={erika} onDone={onDone} />);

        // then
        expect(screen.getByAltText("Bernkastel")).toBeInTheDocument();
        expect(screen.getByAltText("Erika Furudo")).toBeInTheDocument();
        expect(screen.getByText("VS")).toBeInTheDocument();
    });

    it("storms behind the two portraits", () => {
        // given
        const onDone = vi.fn();

        // when
        renderWithProviders(<MinesweeperVsIntro myCharacter={bernkastel} opponentCharacter={erika} onDone={onDone} />);

        // then
        expect(screen.getByTestId("lightning")).toBeInTheDocument();
    });

    it("holds the stage for the full flourish before standing down", () => {
        // given
        const onDone = vi.fn();
        renderWithProviders(<MinesweeperVsIntro myCharacter={bernkastel} opponentCharacter={erika} onDone={onDone} />);

        // when
        act(() => {
            vi.advanceTimersByTime(2399);
        });

        // then
        expect(onDone).not.toHaveBeenCalled();
        act(() => {
            vi.advanceTimersByTime(1);
        });
        expect(onDone).toHaveBeenCalledOnce();
    });

    it("stands down early when it is given a shorter flourish", () => {
        // given
        const onDone = vi.fn();
        renderWithProviders(
            <MinesweeperVsIntro myCharacter={bernkastel} opponentCharacter={erika} onDone={onDone} durationMs={500} />,
        );

        // when
        act(() => {
            vi.advanceTimersByTime(500);
        });

        // then
        expect(onDone).toHaveBeenCalledOnce();
    });

    it("leaves a seat blank when nobody is fighting from it", () => {
        // given
        const onDone = vi.fn();

        // when
        renderWithProviders(
            <MinesweeperVsIntro myCharacter={bernkastel} opponentCharacter={undefined} onDone={onDone} />,
        );

        // then
        expect(screen.getByAltText("Bernkastel")).toBeInTheDocument();
        expect(screen.queryByAltText("Erika Furudo")).not.toBeInTheDocument();
    });

    it("says nothing once it has been taken off the screen", () => {
        // given
        const onDone = vi.fn();
        const { unmount } = renderWithProviders(
            <MinesweeperVsIntro myCharacter={bernkastel} opponentCharacter={erika} onDone={onDone} />,
        );

        // when
        unmount();
        act(() => {
            vi.advanceTimersByTime(5000);
        });

        // then
        expect(onDone).not.toHaveBeenCalled();
    });
});
