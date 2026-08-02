import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChessStats, GameRoom, GameRoomPlayer, User } from "../../../types/api";
import { ChessBoardView } from "./ChessBoardView";

interface StubBoardOptions {
    position: string;
    boardOrientation: string;
    allowDragging: boolean;
    squareStyles: Record<string, unknown>;
    onMouseOverSquare: (arg: { square: string; piece: unknown }) => void;
    onMouseOutSquare: () => void;
    onPieceDrop: (arg: { sourceSquare: string; targetSquare: string | null }) => boolean;
}

interface BoardSquare {
    type: string;
    color: string;
}

const chess = vi.hoisted(() => ({
    load: vi.fn(),
    loadPgn: vi.fn(),
    moves: vi.fn(),
    isCheck: vi.fn(),
    board: vi.fn(),
    turn: vi.fn(),
    history: vi.fn(),
}));

const harness = vi.hoisted(() => ({
    dropFrom: "e2",
    dropTo: "e4" as string | null,
    hover: "e2",
}));

vi.mock("chess.js", () => ({
    Chess: class {
        load = chess.load;
        loadPgn = chess.loadPgn;
        moves = chess.moves;
        isCheck = chess.isCheck;
        board = chess.board;
        turn = chess.turn;
        history = chess.history;
    },
}));

vi.mock("react-chessboard", () => ({
    Chessboard: ({ options }: { options: StubBoardOptions }) => (
        <div
            data-testid="chessboard"
            data-position={options.position}
            data-orientation={options.boardOrientation}
            data-dragging={String(options.allowDragging)}
            data-highlights={Object.keys(options.squareStyles).join(" ")}
        >
            <button
                type="button"
                onClick={() => options.onPieceDrop({ sourceSquare: harness.dropFrom, targetSquare: harness.dropTo })}
            >
                drop piece
            </button>
            <button
                type="button"
                onClick={() => options.onMouseOverSquare({ square: harness.hover, piece: { pieceType: "wP" } })}
            >
                hover square
            </button>
            <button type="button" onClick={() => options.onMouseOutSquare()}>
                leave square
            </button>
        </div>
    ),
}));

const START_FEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";
const MOVE_PLACEHOLDER = "Type a move (e.g. e2e4, e1g1, e7e8q)";

const whiteViewer = makeUser({ id: "u-white", username: "battler", display_name: "Battler" });
const blackViewer = makeUser({ id: "u-black", username: "beatrice", display_name: "Beatrice" });

const onMove = vi.fn<(move: { from: string; to: string; promotion?: string }) => Promise<void>>();
const onResign = vi.fn<() => Promise<void>>();
const onOfferDraw = vi.fn<() => Promise<void>>();
const onAcceptDraw = vi.fn<() => Promise<void>>();
const onDeclineDraw = vi.fn<() => Promise<void>>();

function emptyBoard(): Array<Array<BoardSquare | null>> {
    const rows: Array<Array<BoardSquare | null>> = [];
    for (let r = 0; r < 8; r++) {
        rows.push(new Array<BoardSquare | null>(8).fill(null));
    }
    return rows;
}

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "u-white";
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

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "chess",
        status: "active",
        state: { fen: START_FEN, pgn: "" },
        turn_user_id: "u-white",
        created_by: "u-white",
        created_at: "2026-08-02T10:00:00.000Z",
        updated_at: "2026-08-02T10:00:00.000Z",
        players: [
            makePlayer({ user_id: "u-white", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "u-black", slot: 1, display_name: "Beatrice", username: "beatrice" }),
        ],
        watcher_count: 0,
        ...overrides,
    };
}

function makeStats(overrides: Partial<ChessStats> = {}): ChessStats {
    return {
        total_ply: 42,
        white_moves: 21,
        black_moves: 21,
        white_captures: 5,
        black_captures: 3,
        white_checks: 2,
        black_checks: 1,
        result_reason: "checkmate",
        duration_seconds: 600,
        final_fen: START_FEN,
        ...overrides,
    };
}

function renderBoard(room: GameRoom, viewer: User | null, isSpectator = false) {
    return renderWithProviders(
        <ChessBoardView
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

describe("ChessBoardView", () => {
    beforeEach(() => {
        chess.moves.mockReturnValue([]);
        chess.isCheck.mockReturnValue(false);
        chess.board.mockReturnValue(emptyBoard());
        chess.turn.mockReturnValue("w");
        chess.history.mockReturnValue([]);
        harness.dropFrom = "e2";
        harness.dropTo = "e4";
        harness.hover = "e2";
        onMove.mockResolvedValue(undefined);
        onResign.mockResolvedValue(undefined);
        onOfferDraw.mockResolvedValue(undefined);
        onAcceptDraw.mockResolvedValue(undefined);
        onDeclineDraw.mockResolvedValue(undefined);
    });

    it("draws the position the server sent", () => {
        // given
        const room = makeRoom({ state: { fen: "8/8/8/8/8/8/8/K6k w - - 0 1", pgn: "" } });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(screen.getByTestId("chessboard")).toHaveAttribute("data-position", "8/8/8/8/8/8/8/K6k w - - 0 1");
    });

    it("falls back to the opening position when the room has no board yet", () => {
        // given
        const room = makeRoom({ state: {} });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(screen.getByTestId("chessboard")).toHaveAttribute("data-position", START_FEN);
    });

    it("turns the board around for the player sitting in the black seat", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, blackViewer);

        // then
        expect(screen.getByTestId("chessboard")).toHaveAttribute("data-orientation", "black");
    });

    it("shows a spectator the board from the white side", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByTestId("chessboard")).toHaveAttribute("data-orientation", "white");
    });

    it("lets the player drag only while it is their turn", () => {
        // given
        const room = makeRoom({ turn_user_id: "u-white" });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(screen.getByTestId("chessboard")).toHaveAttribute("data-dragging", "true");
    });

    it("locks dragging while the opponent is thinking", () => {
        // given
        const room = makeRoom({ turn_user_id: "u-black" });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(screen.getByTestId("chessboard")).toHaveAttribute("data-dragging", "false");
    });

    it("locks dragging for a spectator", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByTestId("chessboard")).toHaveAttribute("data-dragging", "false");
    });

    it("sends the legal move that matches the square a piece was dropped on", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([
            { from: "e2", to: "e3" },
            { from: "e2", to: "e4" },
        ]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "drop piece" }));

        // then
        await waitFor(() => {
            expect(onMove).toHaveBeenCalledWith({ from: "e2", to: "e4", promotion: undefined });
        });
    });

    it("refuses a drop on a square the piece cannot reach", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([{ from: "e2", to: "e3" }]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "drop piece" }));

        // then
        expect(onMove).not.toHaveBeenCalled();
    });

    it("refuses a drop while it is the opponent's turn", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([{ from: "e2", to: "e4" }]);
        renderBoard(makeRoom({ turn_user_id: "u-black" }), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "drop piece" }));

        // then
        expect(onMove).not.toHaveBeenCalled();
    });

    it("promotes to a queen when a dropped pawn could become several pieces", async () => {
        // given
        const user = userEvent.setup();
        harness.dropFrom = "e7";
        harness.dropTo = "e8";
        chess.moves.mockReturnValue([
            { from: "e7", to: "e8", promotion: "n" },
            { from: "e7", to: "e8", promotion: "q" },
        ]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "drop piece" }));

        // then
        await waitFor(() => {
            expect(onMove).toHaveBeenCalledWith({ from: "e7", to: "e8", promotion: "q" });
        });
    });

    it("shows why the server rejected a dropped move", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([{ from: "e2", to: "e4" }]);
        onMove.mockRejectedValue(new Error("not your turn"));
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "drop piece" }));

        // then
        expect(await screen.findByText("not your turn")).toBeInTheDocument();
    });

    it("sends the typed coordinate move and empties the box", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([{ from: "e2", to: "e4" }]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.type(screen.getByPlaceholderText(MOVE_PLACEHOLDER), "E2E4");
        await user.click(screen.getByRole("button", { name: "Play" }));

        // then
        await waitFor(() => {
            expect(onMove).toHaveBeenCalledWith({ from: "e2", to: "e4", promotion: undefined });
        });
        expect(screen.getByPlaceholderText(MOVE_PLACEHOLDER)).toHaveValue("");
    });

    it("complains when the typed coordinate is not a square pair", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.type(screen.getByPlaceholderText(MOVE_PLACEHOLDER), "e2e9");
        await user.click(screen.getByRole("button", { name: "Play" }));

        // then
        expect(screen.getByText("Invalid coordinate. Use e.g. e2e4 or e7e8q.")).toBeInTheDocument();
        expect(onMove).not.toHaveBeenCalled();
    });

    it("names the illegal coordinate move it will not play", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([{ from: "e2", to: "e4" }]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.type(screen.getByPlaceholderText(MOVE_PLACEHOLDER), "e2e5");
        await user.click(screen.getByRole("button", { name: "Play" }));

        // then
        expect(screen.getByText("Illegal move: e2e5")).toBeInTheDocument();
        expect(onMove).not.toHaveBeenCalled();
    });

    it("asks for a promotion letter when the typed move promotes", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([
            { from: "e7", to: "e8", promotion: "q" },
            { from: "e7", to: "e8", promotion: "n" },
        ]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.type(screen.getByPlaceholderText(MOVE_PLACEHOLDER), "e7e8");
        await user.click(screen.getByRole("button", { name: "Play" }));

        // then
        expect(screen.getByText("Promotion required: append q, r, b, or n.")).toBeInTheDocument();
        expect(onMove).not.toHaveBeenCalled();
    });

    it("rejects a promotion piece that is not on offer", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([{ from: "e7", to: "e8", promotion: "q" }]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.type(screen.getByPlaceholderText(MOVE_PLACEHOLDER), "e7e8r");
        await user.click(screen.getByRole("button", { name: "Play" }));

        // then
        expect(screen.getByText("No promotion to r available.")).toBeInTheDocument();
        expect(onMove).not.toHaveBeenCalled();
    });

    it("rejects a promotion letter on an ordinary move", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([{ from: "e2", to: "e4" }]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.type(screen.getByPlaceholderText(MOVE_PLACEHOLDER), "e2e4q");
        await user.click(screen.getByRole("button", { name: "Play" }));

        // then
        expect(screen.getByText("That move is not a promotion.")).toBeInTheDocument();
        expect(onMove).not.toHaveBeenCalled();
    });

    it("keeps the coordinate box shut while the opponent is to move", () => {
        // given
        const room = makeRoom({ turn_user_id: "u-black" });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(screen.getByPlaceholderText(MOVE_PLACEHOLDER)).toBeDisabled();
        expect(screen.getByRole("button", { name: "Play" })).toBeDisabled();
    });

    it("gives a spectator no way to move or resign", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.queryByPlaceholderText(MOVE_PLACEHOLDER)).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Resign" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Offer draw" })).not.toBeInTheDocument();
    });

    it("offers accept and decline when the opponent asks for a draw", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom({ draw_offer_from_user_id: "u-black" }), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Accept draw" }));

        // then
        expect(onAcceptDraw).toHaveBeenCalledOnce();
    });

    it("declines the opponent's draw offer", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom({ draw_offer_from_user_id: "u-black" }), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Decline" }));

        // then
        expect(onDeclineDraw).toHaveBeenCalledOnce();
    });

    it("waits quietly on its own draw offer", () => {
        // given
        const room = makeRoom({ draw_offer_from_user_id: "u-white" });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(
            screen.getByText("Draw offered. Waiting for your opponent to respond. It is withdrawn if you make a move."),
        ).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Offer draw" })).not.toBeInTheDocument();
    });

    it("offers a draw when the player asks for one", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Offer draw" }));

        // then
        expect(onOfferDraw).toHaveBeenCalledOnce();
    });

    it("only resigns once the player confirms", async () => {
        // given
        const user = userEvent.setup();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        renderBoard(makeRoom(), whiteViewer);

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
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        expect(onResign).toHaveBeenCalledOnce();
    });

    it("warns the player when their king is in check and marks the square", () => {
        // given
        const board = emptyBoard();
        board[7][4] = { type: "k", color: "w" };
        chess.isCheck.mockReturnValue(true);
        chess.board.mockReturnValue(board);

        // when
        renderBoard(makeRoom(), whiteViewer);

        // then
        expect(screen.getByText("Check!")).toBeInTheDocument();
        expect(screen.getByTestId("chessboard").getAttribute("data-highlights")).toContain("e1");
    });

    it("marks the squares the last move came from and went to", () => {
        // given
        chess.history.mockReturnValue([
            { from: "e2", to: "e4" },
            { from: "e7", to: "e5" },
        ]);

        // when
        renderBoard(makeRoom({ state: { fen: START_FEN, pgn: "1. e4 e5" } }), whiteViewer);

        // then
        expect(screen.getByTestId("chessboard").getAttribute("data-highlights")).toBe("e7 e5");
    });

    it("marks where a hovered piece could go and forgets once the pointer leaves", async () => {
        // given
        const user = userEvent.setup();
        chess.moves.mockReturnValue([
            { from: "e2", to: "e4" },
            { from: "e2", to: "d3", captured: "p" },
        ]);
        renderBoard(makeRoom(), whiteViewer);

        // when
        await user.click(screen.getByRole("button", { name: "hover square" }));

        // then
        expect(screen.getByTestId("chessboard").getAttribute("data-highlights")).toBe("e4 d3");
        await user.click(screen.getByRole("button", { name: "leave square" }));
        expect(screen.getByTestId("chessboard").getAttribute("data-highlights")).toBe("");
    });

    it("tells the winner how the finished game ended", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-white",
            stats: makeStats({ result_reason: "checkmate" }),
        });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("by checkmate")).toBeInTheDocument();
    });

    it("counts the moves each side made once the game is over", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-black",
            stats: makeStats({ white_moves: 12, black_moves: 13, total_ply: 25 }),
        });

        // when
        renderBoard(room, whiteViewer);

        // then
        expect(screen.getByText("You lost")).toBeInTheDocument();
        expect(screen.getByText("Moves")).toBeInTheDocument();
        expect(screen.getByText("Total ply: 25")).toBeInTheDocument();
    });

    it("keeps live stats in front of a spectator while the game runs", () => {
        // given
        const room = makeRoom({ status: "active", stats: makeStats() });

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByText("Live stats")).toBeInTheDocument();
        expect(screen.getByText("Captures")).toBeInTheDocument();
    });
});
