import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeGamePlayer, makeGameRoom, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { GameRoom, GameRoomPlayer, PongState, PongStats, User } from "../../../types/api";
import { PongBoardView } from "./PongBoardView";

interface StubCourtProps {
    mySlot: number | null;
    isSpectator: boolean;
}

vi.mock("./PongCourt", () => ({
    PongCourt: ({ mySlot, isSpectator }: StubCourtProps) => (
        <div data-testid="pong-court" data-slot={String(mySlot)} data-spectator={String(isSpectator)} />
    ),
}));

const playerOne = makeUser({ id: "u-one", username: "battler", display_name: "Battler" });

const onResign = vi.fn<() => Promise<void>>();

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    return makeGamePlayer({ user_id: "u-one", ...overrides });
}

function makeState(overrides: Partial<PongState> = {}): PongState {
    return {
        phase: "rally",
        width: 1200,
        height: 800,
        ball_radius: 10,
        paddle_width: 18,
        paddle_height: 130,
        paddle_inset: 40,
        points_to_win: 7,
        ball_x: 600,
        ball_y: 400,
        ball_vx: 900,
        ball_vy: 120,
        ball_speed: 900,
        paddle_y: [400, 400],
        scores: [3, 2],
        hits: [8, 7],
        longest_rally: [5, 4],
        rally_hits: 2,
        top_speed: 1350,
        serve_to_slot: 1,
        serve_vx: -870,
        serve_vy: 230,
        phase_remain_ms: 0,
        ticks: 900,
        ...overrides,
    };
}

function makeRoom(overrides: Partial<GameRoom<PongState, PongStats>> = {}): GameRoom<PongState, PongStats> {
    return makeGameRoom(makeState(), {
        game_type: "pong",
        created_by: "u-one",
        created_at: "2026-08-26T10:00:00.000Z",
        updated_at: "2026-08-26T10:00:00.000Z",
        players: [
            makePlayer({ user_id: "u-one", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "u-two", slot: 1, display_name: "Erika", username: "erika" }),
        ],
        ...overrides,
    });
}

function makeStats(overrides: Partial<PongStats> = {}): PongStats {
    return {
        points_p0: 7,
        points_p1: 4,
        hits_p0: 31,
        hits_p1: 27,
        longest_rally_p0: 9,
        longest_rally_p1: 6,
        top_speed: 1811,
        result_reason: "win",
        duration_seconds: 240,
        ...overrides,
    };
}

function renderBoard(room: GameRoom<PongState, PongStats>, viewer: User | null, isSpectator = false) {
    return renderWithProviders(
        <PongBoardView room={room} viewer={viewer} isSpectator={isSpectator} onResign={onResign} />,
    );
}

describe("PongBoardView", () => {
    beforeEach(() => {
        onResign.mockResolvedValue(undefined);
    });

    it("puts the running score in front of the players", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("Score 3 - 2, first to 7")).toBeInTheDocument();
    });

    it("hands the court the viewer's own slot", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByTestId("pong-court")).toHaveAttribute("data-slot", "0");
    });

    it("gives a spectator no slot to steer", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByTestId("pong-court")).toHaveAttribute("data-slot", "null");
        expect(screen.getByText("Watching a live match")).toBeInTheDocument();
    });

    it("counts out the match once it is over", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-one",
            state: makeState({ phase: "finished", scores: [7, 4], winner_slot: 0, reason: "win" }),
            stats: makeStats(),
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("on points")).toBeInTheDocument();
        expect(screen.getByText("Points")).toBeInTheDocument();
        expect(screen.getByText("Paddle hits")).toBeInTheDocument();
        expect(screen.getByText("Longest rally")).toBeInTheDocument();
        expect(screen.getByText("Top speed: 1811")).toBeInTheDocument();
    });

    it("keeps live stats in front of a spectator while the match runs", () => {
        // given
        const room = makeRoom({ status: "active", stats: makeStats() });

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.getByText("Live stats")).toBeInTheDocument();
        expect(screen.getByText("Longest rally")).toBeInTheDocument();
    });

    it("shows no stats grid for a stats object belonging to another game", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-one",
            stats: { total_rolls: 24, final_p0: 100 } as unknown as PongStats,
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.queryByText("Paddle hits")).not.toBeInTheDocument();
    });

    it("warns both sides while a disconnected player runs down the clock", () => {
        // given
        const room = makeRoom({
            players: [
                makePlayer({ user_id: "u-one", slot: 0, display_name: "Battler" }),
                makePlayer({
                    user_id: "u-two",
                    slot: 1,
                    display_name: "Erika",
                    username: "erika",
                    connected: false,
                    disconnected_at: new Date(Date.now() - 10_000).toISOString(),
                }),
            ],
        });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.getByText(/disconnected - forfeits in/)).toHaveTextContent("Erika disconnected");
    });

    it("gives a spectator nothing to resign", () => {
        // given
        const room = makeRoom();

        // when
        renderBoard(room, null, true);

        // then
        expect(screen.queryByRole("button", { name: "Resign" })).not.toBeInTheDocument();
    });

    it("packs the resign button away once the match is over", () => {
        // given
        const room = makeRoom({ status: "finished", winner_user_id: "u-two", stats: makeStats() });

        // when
        renderBoard(room, playerOne);

        // then
        expect(screen.queryByRole("button", { name: "Resign" })).not.toBeInTheDocument();
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

    it("says why the server refused the resignation", async () => {
        // given
        const user = userEvent.setup();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        onResign.mockRejectedValue(new Error("room is not active"));
        renderBoard(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        expect(await screen.findByText("room is not active")).toBeInTheDocument();
    });

    it("remembers that the player turned the sound off", async () => {
        // given
        const user = userEvent.setup();
        renderBoard(makeRoom(), playerOne);
        expect(screen.getByRole("button", { name: "Sound on" })).toHaveAttribute("aria-pressed", "true");

        // when
        await user.click(screen.getByRole("button", { name: "Sound on" }));

        // then
        expect(screen.getByRole("button", { name: "Sound off" })).toHaveAttribute("aria-pressed", "false");
        expect(window.localStorage.getItem("pong_muted")).toBe("1");
    });
});
