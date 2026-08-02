import { createEvent, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { MinesweeperCell } from "./MinesweeperCell";

describe("MinesweeperCell", () => {
    it("shows an unmarked hidden cell as empty", () => {
        // given
        const revealed = false;

        // when
        renderWithProviders(<MinesweeperCell revealed={revealed} flagged={false} mine={false} value={0} />);

        // then
        expect(screen.getByRole("button", { name: "hidden" }).textContent).toBe("");
    });

    it("shows a flag on a hidden cell that has been marked", () => {
        // given
        const flagged = true;

        // when
        renderWithProviders(<MinesweeperCell revealed={false} flagged={flagged} mine={false} value={0} />);

        // then
        expect(screen.getByRole("button", { name: "flagged" })).toHaveTextContent("⚑");
    });

    it("shows the neighbouring mine count on a revealed cell", () => {
        // given
        const value = 3;

        // when
        renderWithProviders(<MinesweeperCell revealed flagged={false} mine={false} value={value} />);

        // then
        expect(screen.getByRole("button", { name: "cell value 3" })).toHaveTextContent("3");
    });

    it("shows nothing on a revealed cell with no mines beside it", () => {
        // given
        const value = 0;

        // when
        renderWithProviders(<MinesweeperCell revealed flagged={false} mine={false} value={value} />);

        // then
        expect(screen.getByRole("button", { name: "cell value 0" }).textContent).toBe("");
    });

    it("shows the highest neighbouring count it has a colour for", () => {
        // given
        const value = 8;

        // when
        renderWithProviders(<MinesweeperCell revealed flagged={false} mine={false} value={value} />);

        // then
        expect(screen.getByRole("button", { name: "cell value 8" })).toHaveTextContent("8");
    });

    it("shows a mine once the cell has been revealed", () => {
        // given
        const mine = true;

        // when
        renderWithProviders(<MinesweeperCell revealed flagged={false} mine={mine} value={0} />);

        // then
        expect(screen.getByRole("button", { name: "mine" })).toHaveTextContent("✸");
    });

    it("keeps a hidden mine secret until the board gives it away", () => {
        // given
        const mine = true;

        // when
        renderWithProviders(<MinesweeperCell revealed={false} flagged={false} mine={mine} value={0} />);

        // then
        expect(screen.getByRole("button", { name: "hidden" }).textContent).toBe("");
    });

    it("shows a hidden mine when the finished board forces it open", () => {
        // given
        const forceShowMine = true;

        // when
        renderWithProviders(
            <MinesweeperCell revealed={false} flagged={false} mine forceShowMine={forceShowMine} value={0} />,
        );

        // then
        expect(screen.getByRole("button", { name: "mine" })).toHaveTextContent("✸");
        expect(screen.queryByRole("button", { name: "hidden" })).not.toBeInTheDocument();
    });

    it("prefers the mine over the flag when a flagged cell is forced open", () => {
        // given
        const flagged = true;

        // when
        renderWithProviders(<MinesweeperCell revealed={false} flagged={flagged} mine forceShowMine value={0} />);

        // then
        expect(screen.getByRole("button", { name: "mine" })).toHaveTextContent("✸");
        expect(screen.queryByRole("button", { name: "flagged" })).not.toBeInTheDocument();
    });

    it("keeps a forced open mine unnamed on a board that draws no content", () => {
        // given
        const hideContent = true;

        // when
        renderWithProviders(
            <MinesweeperCell revealed={false} flagged={false} mine forceShowMine value={0} hideContent={hideContent} />,
        );

        // then
        expect(screen.getByRole("button", { name: "hidden" }).textContent).toBe("");
    });

    it("draws nothing at all when the content is hidden", () => {
        // given
        const hideContent = true;

        // when
        renderWithProviders(<MinesweeperCell revealed flagged={false} mine value={4} hideContent={hideContent} />);

        // then
        expect(screen.getByRole("button", { name: "cell value 4" }).textContent).toBe("");
    });

    it("opens the cell on a left press", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MinesweeperCell revealed={false} flagged={false} mine={false} value={0} onClick={onClick} />,
        );

        // when
        await user.click(screen.getByRole("button", { name: "hidden" }));

        // then
        expect(onClick).toHaveBeenCalledOnce();
    });

    it("flags the cell on a right press and swallows the browser menu", () => {
        // given
        const onRightClick = vi.fn();
        const onClick = vi.fn();
        renderWithProviders(
            <MinesweeperCell
                revealed={false}
                flagged={false}
                mine={false}
                value={0}
                onClick={onClick}
                onRightClick={onRightClick}
            />,
        );
        const cell = screen.getByRole("button", { name: "hidden" });

        // when
        const press = createEvent.mouseDown(cell, { button: 2 });
        fireEvent(cell, press);

        // then
        expect(onRightClick).toHaveBeenCalledOnce();
        expect(onClick).not.toHaveBeenCalled();
        expect(press.defaultPrevented).toBe(true);
    });

    it("leaves the default alone for a left press", () => {
        // given
        const onClick = vi.fn();
        renderWithProviders(
            <MinesweeperCell revealed={false} flagged={false} mine={false} value={0} onClick={onClick} />,
        );
        const cell = screen.getByRole("button", { name: "hidden" });

        // when
        const press = createEvent.mouseDown(cell, { button: 0 });
        fireEvent(cell, press);

        // then
        expect(onClick).toHaveBeenCalledOnce();
        expect(press.defaultPrevented).toBe(false);
    });

    it("never lets the browser context menu open over the board", () => {
        // given
        renderWithProviders(<MinesweeperCell revealed={false} flagged={false} mine={false} value={0} />);
        const cell = screen.getByRole("button", { name: "hidden" });

        // when
        const menu = createEvent.contextMenu(cell);
        fireEvent(cell, menu);

        // then
        expect(menu.defaultPrevented).toBe(true);
    });

    it("ignores a right press when there is nothing to flag with", () => {
        // given
        const onClick = vi.fn();
        renderWithProviders(
            <MinesweeperCell revealed={false} flagged={false} mine={false} value={0} onClick={onClick} />,
        );
        const cell = screen.getByRole("button", { name: "hidden" });

        // when
        const press = createEvent.mouseDown(cell, { button: 2 });
        fireEvent(cell, press);

        // then
        expect(onClick).not.toHaveBeenCalled();
        expect(press.defaultPrevented).toBe(false);
    });

    it("ignores a middle press", () => {
        // given
        const onClick = vi.fn();
        const onRightClick = vi.fn();
        renderWithProviders(
            <MinesweeperCell
                revealed={false}
                flagged={false}
                mine={false}
                value={0}
                onClick={onClick}
                onRightClick={onRightClick}
            />,
        );
        const cell = screen.getByRole("button", { name: "hidden" });

        // when
        fireEvent.mouseDown(cell, { button: 1 });

        // then
        expect(onClick).not.toHaveBeenCalled();
        expect(onRightClick).not.toHaveBeenCalled();
    });

    it("does nothing at all while it is disabled", async () => {
        // given
        const onClick = vi.fn();
        const onRightClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(
            <MinesweeperCell
                revealed={false}
                flagged={false}
                mine={false}
                value={0}
                onClick={onClick}
                onRightClick={onRightClick}
                disabled
            />,
        );
        const cell = screen.getByRole("button", { name: "hidden" });

        // when
        await user.click(cell);
        fireEvent.mouseDown(cell, { button: 2 });

        // then
        expect(cell).toBeDisabled();
        expect(onClick).not.toHaveBeenCalled();
        expect(onRightClick).not.toHaveBeenCalled();
    });
});
