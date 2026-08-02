import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { MinesweeperState } from "../../../types/api";
import { MinesweeperBoard } from "./MinesweeperBoard";
import cellStyles from "./MinesweeperCell.module.css";

const WIDTH = 3;
const HEIGHT = 2;

function flags(...set: number[]): boolean[] {
    const out = new Array<boolean>(WIDTH * HEIGHT).fill(false);
    for (const idx of set) {
        out[idx] = true;
    }
    return out;
}

function makeState(overrides: Partial<MinesweeperState> = {}): MinesweeperState {
    return {
        phase: "playing",
        width: WIDTH,
        height: HEIGHT,
        mine_count: 1,
        characters: ["bernkastel", "erika"],
        revealed: [flags(0), flags()],
        flagged: [flags(1), flags()],
        revealed_count: [1, 0],
        values: [[2, 0, 0, 0, 0, 0], new Array<number>(WIDTH * HEIGHT).fill(0)],
        mines: flags(4),
        mines_placed: true,
        pending_clicks: [null, null],
        ...overrides,
    };
}

describe("MinesweeperBoard", () => {
    it("draws one cell for every square of the board", () => {
        // given
        const state = makeState();

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive />);

        // then
        expect(screen.getAllByRole("button")).toHaveLength(WIDTH * HEIGHT);
    });

    it("shows the numbers and flags belonging to the slot it was asked for", () => {
        // given
        const state = makeState();

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive />);

        // then
        expect(screen.getByRole("button", { name: "cell value 2" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "flagged" })).toBeInTheDocument();
        expect(screen.getAllByRole("button", { name: "hidden" })).toHaveLength(4);
    });

    it("leaves the other player's board untouched by this player's progress", () => {
        // given
        const state = makeState();

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={1} interactive={false} />);

        // then
        expect(screen.getAllByRole("button", { name: "hidden" })).toHaveLength(WIDTH * HEIGHT);
    });

    it("reports the square that was opened", async () => {
        // given
        const onReveal = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MinesweeperBoard state={makeState()} slot={0} interactive onReveal={onReveal} onFlag={vi.fn()} />,
        );

        // when
        await user.click(screen.getAllByRole("button", { name: "hidden" })[0]);

        // then
        expect(onReveal).toHaveBeenCalledWith(2, 0);
    });

    it("reports the square that was flagged with a right press", () => {
        // given
        const onFlag = vi.fn();
        renderWithProviders(
            <MinesweeperBoard state={makeState()} slot={0} interactive onReveal={vi.fn()} onFlag={onFlag} />,
        );

        // when
        fireEvent.mouseDown(screen.getAllByRole("button", { name: "hidden" })[0], { button: 2 });

        // then
        expect(onFlag).toHaveBeenCalledWith(2, 0);
    });

    it("turns an ordinary press into a flag while flag mode is on", async () => {
        // given
        const onReveal = vi.fn();
        const onFlag = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MinesweeperBoard state={makeState()} slot={0} interactive flagMode onReveal={onReveal} onFlag={onFlag} />,
        );

        // when
        await user.click(screen.getAllByRole("button", { name: "hidden" })[0]);

        // then
        expect(onFlag).toHaveBeenCalledWith(2, 0);
        expect(onReveal).not.toHaveBeenCalled();
    });

    it("locks a board that is only there to be watched", () => {
        // given
        const state = makeState();

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive={false} />);

        // then
        const cells = screen.getAllByRole("button");
        expect(cells.every(c => (c as HTMLButtonElement).disabled)).toBe(true);
    });

    it("locks the board while the witches are still being chosen", () => {
        // given
        const state = makeState({ phase: "char_select" });

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive />);

        // then
        const cells = screen.getAllByRole("button");
        expect(cells.every(c => (c as HTMLButtonElement).disabled)).toBe(true);
    });

    it("locks a square that has already been opened", () => {
        // given
        const state = makeState();

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive />);

        // then
        expect(screen.getByRole("button", { name: "cell value 2" })).toBeDisabled();
        expect(screen.getAllByRole("button", { name: "hidden" })[0]).toBeEnabled();
    });

    it("gives away every mine once the game is finished", () => {
        // given
        const state = makeState({ phase: "finished", winner_slot: 0 });

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive={false} />);

        // then
        expect(screen.getByRole("button", { name: "mine" })).toHaveTextContent("✸");
    });

    it("crowns the winner", () => {
        // given
        const state = makeState({ phase: "finished", winner_slot: 0 });

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive={false} />);

        // then
        expect(screen.getByText("Winner")).toBeInTheDocument();
        expect(screen.queryByText("Defeated")).not.toBeInTheDocument();
    });

    it("marks the loser as defeated", () => {
        // given
        const state = makeState({ phase: "finished", winner_slot: 0 });

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={1} interactive={false} />);

        // then
        expect(screen.getByText("Defeated")).toBeInTheDocument();
        expect(screen.queryByText("Winner")).not.toBeInTheDocument();
    });

    it("says nothing about the outcome while the game is still running", () => {
        // given
        const state = makeState();

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive />);

        // then
        expect(screen.queryByText("Winner")).not.toBeInTheDocument();
        expect(screen.queryByText("Defeated")).not.toBeInTheDocument();
    });

    it("singles out the mine the loser stepped on", () => {
        // given
        const state = makeState({
            phase: "finished",
            winner_slot: 0,
            mines: flags(5),
            hit_mine_x: 2,
            hit_mine_y: 1,
        });

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={1} interactive={false} />);

        // then
        expect(screen.getByRole("button", { name: "mine" })).toHaveClass(cellStyles.hitMine);
    });

    it("marks the square the player is waiting on", () => {
        // given
        const pendingClick = { x: 2, y: 0 };

        // when
        renderWithProviders(<MinesweeperBoard state={makeState()} slot={0} interactive pendingClick={pendingClick} />);

        // then
        expect(screen.getAllByRole("button", { name: "hidden" })[0]).toHaveClass(cellStyles.pending);
    });

    it("falls back to the square the server says is pending", () => {
        // given
        const state = makeState({ pending_clicks: [[2, 0], null] });

        // when
        renderWithProviders(<MinesweeperBoard state={state} slot={0} interactive />);

        // then
        expect(screen.getAllByRole("button", { name: "hidden" })[0]).toHaveClass(cellStyles.pending);
    });

    it("keeps a shrunken board blank and untouchable", async () => {
        // given
        const onReveal = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MinesweeperBoard state={makeState()} slot={0} interactive cellSize={14} onReveal={onReveal} />,
        );

        // when
        await user.click(screen.getAllByRole("button", { name: "hidden" })[0]);

        // then
        expect(onReveal).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "cell value 2" }).textContent).toBe("");
    });
});
