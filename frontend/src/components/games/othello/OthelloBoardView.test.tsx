import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { GameRoom, GameRoomPlayer, OthelloState, OthelloStats, User } from "../../../types/api";
import { OthelloBoardView } from "./OthelloBoardView";
import styles from "./OthelloBoardView.module.css";

const SQUARE_NAME = /^[a-h][1-8]$/;

const blackViewer = makeUser({ id: "u-black", username: "battler", display_name: "Battler" });
const whiteViewer = makeUser({ id: "u-white", username: "beatrice", display_name: "Beatrice" });

const onMove = vi.fn<(move: { square: string }) => Promise<void>>();
const onResign = vi.fn<() => Promise<void>>();

function boardWith(pieces: Record<string, string>): string {
    const cells = new Array<string>(64).fill(".");
    for (const [square, piece] of Object.entries(pieces)) {
        const col = square.charCodeAt(0) - "a".charCodeAt(0);
        const row = Number(square[1]) - 1;
        cells[row * 8 + col] = piece;
    }
    return cells.join("");
}

const OPENING_BOARD = boardWith({ d4: "W", e4: "B", d5: "B", e5: "W" });

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "u-black";
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

function makeState(overrides: Partial<OthelloState> = {}): OthelloState {
    return {
        board: OPENING_BOARD,
        turn: 0,
        black_moves: 0,
        white_moves: 0,
        black_passes: 0,
        white_passes: 0,
        black_flips: 0,
        white_flips: 0,
        ...overrides,
    };
}

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "othello",
        status: "active",
        state: makeState(),
        turn_user_id: "u-black",
        created_by: "u-black",
        created_at: "2026-08-02T10:00:00.000Z",
        updated_at: "2026-08-02T10:00:00.000Z",
        players: [
            makePlayer({ user_id: "u-black", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "u-white", slot: 1, display_name: "Beatrice", username: "beatrice" }),
        ],
        watcher_count: 0,
        ...overrides,
    };
}

function makeStats(overrides: Partial<OthelloStats> = {}): OthelloStats {
    return {
        total_moves: 60,
        black_moves: 30,
        white_moves: 30,
        black_passes: 1,
        white_passes: 0,
        black_discs: 40,
        white_discs: 24,
        black_flips: 50,
        white_flips: 30,
        black_corners: 3,
        white_corners: 1,
        result_reason: "most_discs",
        duration_seconds: 420,
        final_board: OPENING_BOARD,
        ...overrides,
    };
}

function renderBoard(room: GameRoom, viewer: User | null, isSpectator = false) {
    return renderWithProviders(
        <OthelloBoardView room={room} viewer={viewer} isSpectator={isSpectator} onMove={onMove} onResign={onResign} />,
    );
}

function square(name: string): HTMLElement {
    return screen.getByRole("button", { name });
}

describe("OthelloBoardView", () => {
    beforeEach(() => {
        onMove.mockResolvedValue(undefined);
        onResign.mockResolvedValue(undefined);
    });

    it("draws all sixty four squares with the eighth rank first", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, blackViewer);

        // then
        const squares = screen.getAllByRole("button", { name: SQUARE_NAME });
        expect(squares).toHaveLength(64);
        expect(squares[0]).toHaveAccessibleName("a8");
    });

    it("sets out the four opening discs when the room has no board yet", () => {
        // given
        const room = makeRoom({ state: {} });

        // when
        renderBoard(room, blackViewer);

        // then
        expect(screen.getByText("Black: 2")).toBeInTheDocument();
        expect(screen.getByText("White: 2")).toBeInTheDocument();
    });

    it("counts the discs currently on the board", () => {
        // given
        const room = makeRoom({
            state: makeState({ board: boardWith({ d4: "W", e4: "B", d5: "B", e5: "W", c3: "B" }) }),
        });

        // when
        renderBoard(room, blackViewer);

        // then
        expect(screen.getByText("Black: 3")).toBeInTheDocument();
        expect(screen.getByText("White: 2")).toBeInTheDocument();
    });

    it("opens only the squares that would flip a disc for the player to move", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, blackViewer);

        // then
        expect(square("d3")).toBeEnabled();
        expect(square("c4")).toBeEnabled();
        expect(square("f5")).toBeEnabled();
        expect(square("e6")).toBeEnabled();
        expect(square("a1")).toBeDisabled();
    });

    it("offers the white player a different set of squares", () => {
        // given
        const room = makeRoom({ turn_user_id: "u-white" });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(square("e3")).toBeEnabled();
        expect(square("d3")).toBeDisabled();
    });

    it("sends the square the player placed a disc on", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), blackViewer);

        // when
        await user.click(square("d3"));

        // then
        await waitFor(() => {
            expect(onMove).toHaveBeenCalledWith({ square: "d3" });
        });
    });

    it("leaves every square dead while the opponent is to move", () => {
        // given
        const room = makeRoom({ turn_user_id: "u-white" });

        // when
        renderBoard(room, blackViewer);

        // then
        const squares = screen.getAllByRole("button", { name: SQUARE_NAME });
        expect(squares.every(s => (s as HTMLButtonElement).disabled)).toBe(true);
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

    it("says the server will pass for a player with no legal move", () => {
        // given
        const room = makeRoom({ state: makeState({ board: boardWith({ a1: "B" }) }) });

        // when
        renderBoard(room, blackViewer);

        // then
        expect(
            screen.getByText("You have no legal moves; the server will pass for you on the next sync."),
        ).toBeInTheDocument();
    });

    it("explains that the opponent passed when the turn comes straight back", () => {
        // given
        const room = makeRoom({ state: makeState({ last_move: { square: "d3", slot: 0, flipped: ["d4"] } }) });

        // when
        renderBoard(room, blackViewer);

        // then
        expect(screen.getByText("Opponent had no legal moves and passed. Your turn again.")).toBeInTheDocument();
    });

    it("marks where the last disc landed and which discs it turned over", () => {
        // given
        const room = makeRoom({
            turn_user_id: "u-white",
            state: makeState({ last_move: { square: "d3", slot: 0, flipped: ["d4"] } }),
        });

        // when
        renderBoard(room, blackViewer);

        // then
        expect(square("d3")).toHaveClass(styles.squareLastMove);
        expect(square("d4")).toHaveClass(styles.squareFlipped);
    });

    it("shows why the server refused the placement", async () => {
        // given
        const user = userEvent.setup();
        onMove.mockRejectedValue(new Error("square already taken"));
        renderBoard(makeRoom(), blackViewer);

        // when
        await user.click(square("d3"));

        // then
        expect(await screen.findByText("square already taken")).toBeInTheDocument();
    });

    it("only resigns once the player confirms", async () => {
        // given
        const user = userEvent.setup();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        renderBoard(makeRoom(), blackViewer);

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
        renderBoard(makeRoom(), blackViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        expect(onResign).toHaveBeenCalledOnce();
    });

    it("tells the winner how the finished game ended", () => {
        // given
        const room = makeRoom({ status: "finished", winner_user_id: "u-black", stats: makeStats() });

        // when
        renderBoard(room, blackViewer);

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("with the most discs")).toBeInTheDocument();
        expect(screen.getByText("Corners")).toBeInTheDocument();
        expect(screen.getByText("Total moves: 60")).toBeInTheDocument();
    });

    it("names the winner for a spectator watching the end", () => {
        // given
        const room = makeRoom({ status: "finished", winner_user_id: "u-white", stats: makeStats() });

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByText("Beatrice won")).toBeInTheDocument();
    });

    it("keeps live stats in front of a spectator while the game runs", () => {
        // given
        const room = makeRoom({ status: "active", stats: makeStats() });

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByText("Live stats")).toBeInTheDocument();
        expect(screen.getByText("Passes")).toBeInTheDocument();
    });
});
