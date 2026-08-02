import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { CheckersState, CheckersStats, GameRoom, GameRoomPlayer, User } from "../../../types/api";
import { CheckersBoardView } from "./CheckersBoardView";
import styles from "./CheckersBoardView.module.css";

const SQUARE_NAME = /^[a-h][1-8]$/;

const redViewer = makeUser({ id: "u-red", username: "battler", display_name: "Battler" });
const blackViewer = makeUser({ id: "u-black", username: "beatrice", display_name: "Beatrice" });

const onMove = vi.fn<(move: { from: string; path: string[] }) => Promise<void>>();
const onResign = vi.fn<() => Promise<void>>();
const onOfferDraw = vi.fn<() => Promise<void>>();
const onAcceptDraw = vi.fn<() => Promise<void>>();
const onDeclineDraw = vi.fn<() => Promise<void>>();

function boardWith(pieces: Record<string, string>): string {
    const cells = new Array<string>(64).fill(".");
    for (const [square, piece] of Object.entries(pieces)) {
        const col = square.charCodeAt(0) - "a".charCodeAt(0);
        const row = Number(square[1]) - 1;
        cells[row * 8 + col] = piece;
    }
    return cells.join("");
}

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "u-red";
    return {
        user_id: id,
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        role: "player",
        slot: 0,
        joined: true,
        connected: true,
        user: { id, username: "battler", display_name: "Battler" },
        ...overrides,
    };
}

function makeState(overrides: Partial<CheckersState> = {}): CheckersState {
    return {
        board: boardWith({ c3: "r", h8: "b" }),
        turn: 0,
        total_moves: 0,
        red_captures: 0,
        black_captures: 0,
        red_crownings: 0,
        black_crownings: 0,
        moves_since_capture: 0,
        ...overrides,
    };
}

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "checkers",
        status: "active",
        state: makeState(),
        turn_user_id: "u-red",
        created_by: "u-red",
        created_at: "2026-08-02T10:00:00.000Z",
        updated_at: "2026-08-02T10:00:00.000Z",
        players: [
            makePlayer({ user_id: "u-red", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "u-black", slot: 1, display_name: "Beatrice", username: "beatrice" }),
        ],
        watcher_count: 0,
        ...overrides,
    };
}

function makeStats(overrides: Partial<CheckersStats> = {}): CheckersStats {
    return {
        total_moves: 30,
        red_moves: 15,
        black_moves: 15,
        red_captures: 4,
        black_captures: 2,
        red_crownings: 1,
        black_crownings: 0,
        result_reason: "no_pieces",
        duration_seconds: 300,
        final_board: boardWith({ c3: "R" }),
        red_pieces_left: 6,
        black_pieces_left: 0,
        ...overrides,
    };
}

function renderBoard(room: GameRoom, viewer: User | null, isSpectator = false) {
    return renderWithProviders(
        <CheckersBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onMove={onMove}
            onResign={onResign}
            onOfferDraw={onOfferDraw}
            onAcceptDraw={onAcceptDraw}
            onDeclineDraw={onDeclineDraw}
        />,
    );
}

function square(name: string): HTMLElement {
    return screen.getByRole("button", { name });
}

describe("CheckersBoardView", () => {
    beforeEach(() => {
        onMove.mockResolvedValue(undefined);
        onResign.mockResolvedValue(undefined);
        onOfferDraw.mockResolvedValue(undefined);
        onAcceptDraw.mockResolvedValue(undefined);
        onDeclineDraw.mockResolvedValue(undefined);
    });

    it("draws all sixty four squares", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, redViewer);

        // then
        expect(screen.getAllByRole("button", { name: SQUARE_NAME })).toHaveLength(64);
    });

    it("shows the red player the eighth rank across the top", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, redViewer);

        // then
        expect(screen.getAllByRole("button", { name: SQUARE_NAME })[0]).toHaveAccessibleName("a8");
    });

    it("shows the black player the board from the other end", () => {
        // given
        const room = makeRoom({ turn_user_id: "u-black" });

        // when
        renderBoard(room, blackViewer);

        // then
        expect(screen.getAllByRole("button", { name: SQUARE_NAME })[0]).toHaveAccessibleName("h1");
    });

    it("crowns a king with a crown mark", () => {
        // given
        const room = makeRoom({ state: makeState({ board: boardWith({ c3: "R", h8: "b" }) }) });

        // when
        renderBoard(room, redViewer);

        // then
        expect(square("c3")).toHaveTextContent("♛");
    });

    it("leaves every square dead for a spectator", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, null, true);

        // then
        const squares = screen.getAllByRole("button", { name: SQUARE_NAME });
        expect(squares.every(s => (s as HTMLButtonElement).disabled)).toBe(true);
        expect(screen.queryByRole("button", { name: "Resign" })).not.toBeInTheDocument();
    });

    it("leaves every square dead while the opponent is to move", () => {
        // given
        const room = makeRoom({ turn_user_id: "u-black" });

        // when
        renderBoard(room, redViewer);

        // then
        const squares = screen.getAllByRole("button", { name: SQUARE_NAME });
        expect(squares.every(s => (s as HTMLButtonElement).disabled)).toBe(true);
    });

    it("marks the squares a selected piece may step to", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), redViewer);

        // when
        await user.click(square("c3"));

        // then
        expect(square("c3")).toHaveClass(styles.squareSelected);
        expect(square("b4")).toHaveClass(styles.squareTarget);
        expect(square("d4")).toHaveClass(styles.squareTarget);
    });

    it("ignores a click on an opponent piece", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), redViewer);

        // when
        await user.click(square("h8"));

        // then
        expect(square("h8")).not.toHaveClass(styles.squareSelected);
        expect(square("b4")).not.toHaveClass(styles.squareTarget);
    });

    it("sends the simple step the player picked", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), redViewer);

        // when
        await user.click(square("c3"));
        await user.click(square("b4"));

        // then
        await waitFor(() => {
            expect(onMove).toHaveBeenCalledWith({ from: "c3", path: ["b4"] });
        });
    });

    it("tells the player a capture is mandatory and refuses a quiet step", async () => {
        // given
        const user = userEvent.setup();
        const room = makeRoom({ state: makeState({ board: boardWith({ c3: "r", d4: "b" }) }) });
        renderBoard(room, redViewer);

        // when
        await user.click(square("c3"));

        // then
        expect(screen.getByText("Capture is mandatory this turn.")).toBeInTheDocument();
        expect(square("b4")).not.toHaveClass(styles.squareTarget);
        await user.click(square("b4"));
        expect(onMove).not.toHaveBeenCalled();
    });

    it("sends a single jump as soon as there is nothing left to capture", async () => {
        // given
        const user = userEvent.setup();
        const room = makeRoom({ state: makeState({ board: boardWith({ c3: "r", d4: "b" }) }) });
        renderBoard(room, redViewer);

        // when
        await user.click(square("c3"));
        await user.click(square("e5"));

        // then
        await waitFor(() => {
            expect(onMove).toHaveBeenCalledWith({ from: "c3", path: ["e5"] });
        });
    });

    it("holds the turn open until a chain of jumps is finished", async () => {
        // given
        const user = userEvent.setup();
        const room = makeRoom({ state: makeState({ board: boardWith({ b2: "r", c3: "b", e5: "b" }) }) });
        renderBoard(room, redViewer);

        // when
        await user.click(square("b2"));
        await user.click(square("d4"));

        // then
        expect(
            screen.getByText("Keep jumping - another capture is required before your turn ends."),
        ).toBeInTheDocument();
        expect(onMove).not.toHaveBeenCalled();
    });

    it("sends the whole jump chain once the last capture lands", async () => {
        // given
        const user = userEvent.setup();
        const room = makeRoom({ state: makeState({ board: boardWith({ b2: "r", c3: "b", e5: "b" }) }) });
        renderBoard(room, redViewer);

        // when
        await user.click(square("b2"));
        await user.click(square("d4"));
        await user.click(square("f6"));

        // then
        await waitFor(() => {
            expect(onMove).toHaveBeenCalledWith({ from: "b2", path: ["d4", "f6"] });
        });
    });

    it("shows why the server refused the move and drops the selection", async () => {
        // given
        const user = userEvent.setup();
        onMove.mockRejectedValue(new Error("not your turn"));
        renderBoard(makeRoom(), redViewer);

        // when
        await user.click(square("c3"));
        await user.click(square("b4"));

        // then
        expect(await screen.findByText("not your turn")).toBeInTheDocument();
        expect(square("c3")).not.toHaveClass(styles.squareSelected);
    });

    it("marks the squares the previous move passed through and the piece it took", () => {
        // given
        const room = makeRoom({
            state: makeState({
                board: boardWith({ e5: "r" }),
                last_move: { from: "c3", path: ["e5"], captured: ["d4"] },
            }),
        });

        // when
        renderBoard(room, redViewer);

        // then
        expect(square("c3")).toHaveClass(styles.squareLastMove);
        expect(square("e5")).toHaveClass(styles.squareLastMove);
        expect(square("d4")).toHaveTextContent("×");
    });

    it("falls back to the opening layout when the room has no board yet", () => {
        // given
        const room = makeRoom({ state: {} });

        // when
        renderBoard(room, redViewer);

        // then
        expect(square("a1")).not.toBeEmptyDOMElement();
        expect(square("a7")).not.toBeEmptyDOMElement();
        expect(square("a5")).toBeEmptyDOMElement();
    });

    it("offers accept and decline when the opponent asks for a draw", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom({ draw_offer_from_user_id: "u-black" }), redViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Accept draw" }));

        // then
        expect(onAcceptDraw).toHaveBeenCalledOnce();
    });

    it("declines the opponent's draw offer", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom({ draw_offer_from_user_id: "u-black" }), redViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Decline" }));

        // then
        expect(onDeclineDraw).toHaveBeenCalledOnce();
    });

    it("waits quietly on its own draw offer", () => {
        // given
        const room = makeRoom({ draw_offer_from_user_id: "u-red" });

        // when
        renderBoard(room, redViewer);

        // then
        expect(
            screen.getByText("Draw offered. Waiting for your opponent to respond. It is withdrawn if you make a move."),
        ).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Offer draw" })).not.toBeInTheDocument();
    });

    it("only resigns once the player confirms", async () => {
        // given
        const user = userEvent.setup();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        renderBoard(makeRoom(), redViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Resign this game?");
        expect(onResign).not.toHaveBeenCalled();
    });

    it("resigns when the player confirms", async () => {
        // given
        const user = userEvent.setup();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        renderBoard(makeRoom(), redViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        expect(onResign).toHaveBeenCalledOnce();
    });

    it("tells the winner how the finished game ended", () => {
        // given
        const room = makeRoom({ status: "finished", winner_user_id: "u-red", stats: makeStats() });

        // when
        renderBoard(room, redViewer);

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("by capturing all pieces")).toBeInTheDocument();
        expect(screen.getByText("Kings crowned")).toBeInTheDocument();
        expect(screen.getByText("Total moves: 30")).toBeInTheDocument();
    });

    it("keeps live stats in front of a spectator while the game runs", () => {
        // given
        const room = makeRoom({ status: "active", stats: makeStats() });

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByText("Live stats")).toBeInTheDocument();
        expect(screen.getByText("Pieces left")).toBeInTheDocument();
    });
});
