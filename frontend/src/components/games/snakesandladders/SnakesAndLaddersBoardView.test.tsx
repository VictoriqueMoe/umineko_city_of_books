import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { GameRoom, GameRoomPlayer, SnakesLaddersState, SnakesLaddersStats, User } from "../../../types/api";
import { SnakesAndLaddersBoardView } from "./SnakesAndLaddersBoardView";

interface StubBoardProps {
    positions: number[];
    tokens: Array<{ initial: string }>;
    lastTo?: number | null;
}

vi.mock("./SnakesLaddersBoard.tsx", () => ({
    SnakesLaddersBoard: ({ positions, tokens, lastTo }: StubBoardProps) => (
        <div
            data-testid="sl-board"
            data-positions={positions.join(",")}
            data-tokens={tokens.map(t => t.initial).join(",")}
            data-last-to={String(lastTo)}
        />
    ),
}));

const playerOne = makeUser({ id: "u-one", username: "battler", display_name: "Battler" });

const onRoll = vi.fn<() => Promise<void>>();
const onResign = vi.fn<() => Promise<void>>();

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "u-one";
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

function makeState(overrides: Partial<SnakesLaddersState> = {}): SnakesLaddersState {
    return {
        positions: [0, 0],
        turn: 0,
        rolls: 0,
        ladders_climbed: [0, 0],
        snakes_hit: [0, 0],
        ...overrides,
    };
}

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "snakes_and_ladders",
        status: "active",
        state: makeState(),
        turn_user_id: "u-one",
        created_by: "u-one",
        created_at: "2026-08-02T10:00:00.000Z",
        updated_at: "2026-08-02T10:00:00.000Z",
        players: [
            makePlayer({ user_id: "u-one", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "u-two", slot: 1, display_name: "Erika", username: "erika" }),
        ],
        watcher_count: 0,
        ...overrides,
    };
}

function makeStats(overrides: Partial<SnakesLaddersStats> = {}): SnakesLaddersStats {
    return {
        total_rolls: 24,
        rolls_p0: 12,
        rolls_p1: 12,
        ladders_p0: 3,
        ladders_p1: 1,
        snakes_p0: 1,
        snakes_p1: 4,
        final_p0: 100,
        final_p1: 62,
        result_reason: "win",
        duration_seconds: 180,
        ...overrides,
    };
}

function renderBoard(room: GameRoom, viewer: User | null, isSpectator = false) {
    return renderWithProviders(
        <SnakesAndLaddersBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onRoll={onRoll}
            onResign={onResign}
        />,
    );
}

describe("SnakesAndLaddersBoardView", () => {
    beforeEach(() => {
        onRoll.mockResolvedValue(undefined);
        onResign.mockResolvedValue(undefined);
    });

    it("invites the player whose turn it is to roll", () => {
        // given
        const room = makeRoom({
            state: makeState({ rolls: 1, last: { slot: 1, roll: 3, from: 0, stepped: 3, to: 3 } }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByRole("button", { name: "Roll the die" })).toBeEnabled();
    });

    it("makes the waiting player watch instead", () => {
        // given
        const room = makeRoom({
            turn_user_id: "u-two",
            state: makeState({ rolls: 1, last: { slot: 0, roll: 3, from: 0, stepped: 3, to: 3 } }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByRole("button", { name: "Waiting for opponent..." })).toBeDisabled();
    });

    it("rolls the die for the player", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "Roll the die" }));

        // then
        expect(onRoll).toHaveBeenCalledOnce();
    });

    it("shows why the server refused the roll", async () => {
        // given
        const user = userEvent.setup();
        onRoll.mockRejectedValue(new Error("not your turn"));
        renderBoard(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "Roll the die" }));

        // then
        expect(await screen.findByText("not your turn")).toBeInTheDocument();
    });

    it("gives a spectator nothing to press", () => {
        // given
        const room = makeRoom({
            state: makeState({ rolls: 1, last: { slot: 0, roll: 3, from: 0, stepped: 3, to: 3 } }),
        });

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.queryByRole("button", { name: "Roll the die" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Waiting for opponent..." })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Resign" })).not.toBeInTheDocument();
    });

    it("shows the face of the die that was just rolled", () => {
        // given
        const room = makeRoom({
            state: makeState({ rolls: 1, last: { slot: 0, roll: 4, from: 0, stepped: 4, to: 4 } }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByRole("img", { name: "Die showing 4" })).toBeInTheDocument();
    });

    it("describes a plain move", () => {
        // given
        const room = makeRoom({
            state: makeState({
                positions: [10, 0],
                rolls: 1,
                last: { slot: 0, roll: 4, from: 6, stepped: 10, to: 10 },
            }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("Battler rolled 4 and moved to 10.")).toBeInTheDocument();
    });

    it("describes a ladder climb", () => {
        // given
        const room = makeRoom({
            state: makeState({ positions: [31, 0], rolls: 1, last: { slot: 0, roll: 3, from: 6, stepped: 9, to: 31 } }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("Battler rolled 3, found a ladder at 9 and climbed to 31.")).toBeInTheDocument();
    });

    it("describes a snake bite", () => {
        // given
        const room = makeRoom({
            state: makeState({
                positions: [0, 6],
                rolls: 1,
                last: { slot: 1, roll: 2, from: 14, stepped: 16, to: 6 },
            }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("Erika rolled 2, hit a snake at 16 and slid to 6.")).toBeInTheDocument();
    });

    it("describes overshooting the last square", () => {
        // given
        const room = makeRoom({
            state: makeState({
                positions: [98, 0],
                rolls: 1,
                last: { slot: 0, roll: 5, from: 98, stepped: 98, to: 98 },
            }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("Battler rolled 5 and overshot 100 - staying put.")).toBeInTheDocument();
    });

    it("names an empty seat rather than a missing player", () => {
        // given
        const room = makeRoom({
            players: [makePlayer({ user_id: "u-one", slot: 0, display_name: "Battler" })],
            state: makeState({
                positions: [0, 10],
                rolls: 1,
                last: { slot: 1, roll: 4, from: 6, stepped: 10, to: 10 },
            }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("Player 2 rolled 4 and moved to 10.")).toBeInTheDocument();
    });

    it("takes each token initial from the seated player", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-tokens", "B,E");
    });

    it("falls back to a question mark for an empty seat", () => {
        // given
        const room = makeRoom({ players: [makePlayer({ user_id: "u-one", slot: 0, display_name: "Battler" })] });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-tokens", "B,?");
    });

    it("shows the tokens where the server says they are on the first sight of a game", () => {
        // given
        const room = makeRoom({
            state: makeState({ positions: [31, 6], rolls: 4, last: { slot: 0, roll: 3, from: 6, stepped: 9, to: 31 } }),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-positions", "31,6");
    });

    it("walks the token through the roll before dropping it down the snake", () => {
        // given
        vi.useFakeTimers();
        const room = makeRoom({
            state: makeState({
                positions: [14, 0],
                rolls: 1,
                last: { slot: 0, roll: 2, from: 12, stepped: 14, to: 14 },
            }),
        });
        const { rerender } = renderBoard(room, playerOne);
        const rolled = makeRoom({
            state: makeState({ positions: [6, 0], rolls: 2, last: { slot: 0, roll: 2, from: 14, stepped: 16, to: 6 } }),
        });

        // when
        rerender(
            <SnakesAndLaddersBoardView
                room={rolled}
                viewer={playerOne}
                isSpectator={false}
                onRoll={onRoll}
                onResign={onResign}
            />,
        );

        // then
        act(() => {
            vi.advanceTimersByTime(0);
        });
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-positions", "14,0");
        act(() => {
            vi.advanceTimersByTime(600);
        });
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-positions", "16,0");
        act(() => {
            vi.advanceTimersByTime(850);
        });
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-positions", "6,0");
    });

    it("follows a correction to the positions that arrives without a new roll", () => {
        // given
        const last = { slot: 0, roll: 2, from: 12, stepped: 14, to: 14 };
        const room = makeRoom({ state: makeState({ positions: [14, 0], rolls: 1, last }) });
        const { rerender } = renderBoard(room, playerOne);
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-positions", "14,0");

        // when
        const corrected = makeRoom({ state: makeState({ positions: [3, 9], rolls: 1, last }) });
        rerender(
            <SnakesAndLaddersBoardView
                room={corrected}
                viewer={playerOne}
                isSpectator={false}
                onRoll={onRoll}
                onResign={onResign}
            />,
        );

        // then
        expect(screen.getByTestId("sl-board")).toHaveAttribute("data-positions", "3,9");
    });

    it("only resigns once the player confirms", async () => {
        // given
        const user = userEvent.setup();
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        renderBoard(makeRoom(), playerOne);

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
        renderBoard(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        await waitFor(() => {
            expect(onResign).toHaveBeenCalledOnce();
        });
    });

    it("packs the roll controls away once the game is over", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-one",
            state: makeState({
                positions: [100, 62],
                rolls: 24,
                last: { slot: 0, roll: 2, from: 98, stepped: 100, to: 100 },
            }),
            stats: makeStats(),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.queryByRole("button", { name: "Roll the die" })).not.toBeInTheDocument();
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("Ladders climbed")).toBeInTheDocument();
        expect(screen.getByText("Total rolls: 24")).toBeInTheDocument();
    });

    it("keeps live stats in front of a spectator while the game runs", () => {
        // given
        const room = makeRoom({ status: "active", stats: makeStats() });

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByText("Live stats")).toBeInTheDocument();
        expect(screen.getByText("Snakes hit")).toBeInTheDocument();
    });
});
