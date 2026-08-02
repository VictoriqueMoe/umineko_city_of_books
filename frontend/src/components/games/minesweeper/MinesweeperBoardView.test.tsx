import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { GameRoom, GameRoomPlayer, MinesweeperState, MinesweeperStats, User } from "../../../types/api";
import { MinesweeperBoardView } from "./MinesweeperBoardView";

interface StubBoardProps {
    slot: number;
    interactive: boolean;
    cellSize?: number;
    flagMode?: boolean;
    pendingClick?: { x: number; y: number } | null;
    onReveal?: (x: number, y: number) => void;
    onFlag?: (x: number, y: number) => void;
}

interface StubIntroProps {
    onDone: () => void;
}

const mobile = vi.hoisted(() => ({ value: false }));

vi.mock("../../../hooks/useIsMobile", () => ({
    useIsMobile: () => mobile.value,
}));

vi.mock("./MinesweeperLightningCanvas", () => ({
    MinesweeperLightningCanvas: () => <div data-testid="lightning" />,
}));

vi.mock("./MinesweeperVsIntro", () => ({
    MinesweeperVsIntro: ({ onDone }: StubIntroProps) => (
        <div data-testid="vs-intro">
            <button type="button" onClick={onDone}>
                finish intro
            </button>
        </div>
    ),
}));

vi.mock("./MinesweeperBoard", () => ({
    MinesweeperBoard: ({ slot, interactive, cellSize, flagMode, pendingClick, onReveal, onFlag }: StubBoardProps) => (
        <div
            data-testid={`board-${slot}`}
            data-interactive={String(interactive)}
            data-cell-size={String(cellSize)}
            data-flag-mode={String(flagMode)}
            data-pending={pendingClick ? `${pendingClick.x},${pendingClick.y}` : ""}
        >
            <button type="button" onClick={() => onReveal?.(3, 4)}>
                reveal on {slot}
            </button>
            <button type="button" onClick={() => onFlag?.(5, 6)}>
                flag on {slot}
            </button>
        </div>
    ),
}));

const WIDTH = 9;
const HEIGHT = 9;
const CELLS = WIDTH * HEIGHT;

const playerOne = makeUser({ id: "u-one", username: "battler", display_name: "Battler" });

const onAction = vi.fn<(action: Record<string, unknown>) => Promise<void>>();
const onResign = vi.fn<() => Promise<void>>();

function marks(...set: number[]): boolean[] {
    const out = new Array<boolean>(CELLS).fill(false);
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
        mine_count: 10,
        characters: ["bernkastel", "erika"],
        revealed: [marks(), marks()],
        flagged: [marks(0, 1), marks(2)],
        revealed_count: [12, 5],
        values: [new Array<number>(CELLS).fill(0), new Array<number>(CELLS).fill(0)],
        mines: marks(4),
        mines_placed: true,
        pending_clicks: [null, null],
        ...overrides,
    };
}

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

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "minesweeper",
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

function makeStats(overrides: Partial<MinesweeperStats> = {}): MinesweeperStats {
    return {
        duration_seconds: 240,
        revealed_p0: 71,
        revealed_p1: 40,
        flags_p0: 8,
        flags_p1: 3,
        reason: "mine_hit",
        ...overrides,
    };
}

function renderView(room: GameRoom, viewer: User | null, isSpectator = false) {
    return renderWithProviders(
        <MinesweeperBoardView
            room={room}
            viewer={viewer}
            isSpectator={isSpectator}
            onAction={onAction}
            onResign={onResign}
        />,
    );
}

async function renderPlaying(room: GameRoom, viewer: User | null, isSpectator = false) {
    const user = userEvent.setup();
    const result = renderView(room, viewer, isSpectator);
    await user.click(screen.getByRole("button", { name: "finish intro" }));
    return { user, ...result };
}

describe("MinesweeperBoardView", () => {
    beforeEach(() => {
        mobile.value = false;
        onAction.mockResolvedValue(undefined);
        onResign.mockResolvedValue(undefined);
    });

    it("waits politely while the room has no game state", () => {
        // given
        const room = { ...makeRoom(), state: undefined as unknown as GameRoom["state"] };

        // when
        renderView(room, playerOne);

        // then
        expect(screen.getByText("Loading game...")).toBeInTheDocument();
    });

    it("holds the council until both witches are picked", () => {
        // given
        const room = makeRoom({ state: makeState({ phase: "char_select", characters: ["", ""] }) });

        // when
        renderView(room, playerOne);

        // then
        expect(screen.getByText("Choose a witch to fight as.")).toBeInTheDocument();
        expect(screen.queryByTestId("board-0")).not.toBeInTheDocument();
    });

    it("sends the witch the player picked in the council", async () => {
        // given
        const user = userEvent.setup();
        const room = makeRoom({ state: makeState({ phase: "char_select", characters: ["", ""] }) });
        renderView(room, playerOne);

        // when
        await user.click(screen.getByRole("button", { name: /Erika Furudo/ }));

        // then
        expect(onAction).toHaveBeenCalledWith({ type: "select_character", character: "erika" });
    });

    it("plays the duel introduction before the boards appear", () => {
        // given
        const room = makeRoom();

        // when
        renderView(room, playerOne);

        // then
        expect(screen.getByTestId("vs-intro")).toBeInTheDocument();
        expect(screen.queryByTestId("board-0")).not.toBeInTheDocument();
    });

    it("puts both boards up once the introduction is done", async () => {
        // given
        const room = makeRoom();

        // when
        await renderPlaying(room, playerOne);

        // then
        expect(screen.getByTestId("board-0")).toHaveAttribute("data-interactive", "true");
        expect(screen.getByTestId("board-1")).toHaveAttribute("data-interactive", "false");
    });

    it("puts the player's own board on the left whichever seat they took", async () => {
        // given
        const room = makeRoom({ turn_user_id: "u-two" });
        const viewer = makeUser({ id: "u-two", username: "erika", display_name: "Erika" });

        // when
        await renderPlaying(room, viewer);

        // then
        expect(screen.getByTestId("board-1")).toHaveAttribute("data-interactive", "true");
        expect(screen.getByTestId("board-0")).toHaveAttribute("data-interactive", "false");
    });

    it("sends the square the player opened", async () => {
        // given
        const { user } = await renderPlaying(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "reveal on 0" }));

        // then
        expect(onAction).toHaveBeenCalledWith({ type: "reveal", x: 3, y: 4 });
    });

    it("sends the square the player flagged", async () => {
        // given
        const { user } = await renderPlaying(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "flag on 0" }));

        // then
        expect(onAction).toHaveBeenCalledWith({ type: "flag", x: 5, y: 6 });
    });

    it("holds the first click on screen until the mines have been laid", async () => {
        // given
        const room = makeRoom({ state: makeState({ mines_placed: false }) });
        const { user } = await renderPlaying(room, playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "reveal on 0" }));

        // then
        await waitFor(() => {
            expect(screen.getByTestId("board-0")).toHaveAttribute("data-pending", "3,4");
        });
    });

    it("lets go of the pending click once the mines are laid", async () => {
        // given
        const room = makeRoom({ state: makeState({ mines_placed: false }) });
        const { user, rerender } = await renderPlaying(room, playerOne);
        await user.click(screen.getByRole("button", { name: "reveal on 0" }));

        // when
        rerender(
            <MinesweeperBoardView
                room={makeRoom({ state: makeState({ mines_placed: true }) })}
                viewer={playerOne}
                isSpectator={false}
                onAction={onAction}
                onResign={onResign}
            />,
        );

        // then
        await waitFor(() => {
            expect(screen.getByTestId("board-0")).toHaveAttribute("data-pending", "");
        });
    });

    it("shows why the server refused the action", async () => {
        // given
        onAction.mockRejectedValue(new Error("not your board"));
        const { user } = await renderPlaying(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "reveal on 0" }));

        // then
        expect(await screen.findByText("not your board")).toBeInTheDocument();
    });

    it("counts the safe squares each side has cleared", async () => {
        // given
        const room = makeRoom();

        // when
        await renderPlaying(room, playerOne);

        // then
        expect(screen.getByText("12 / 71")).toBeInTheDocument();
        expect(screen.getByText("5 / 71")).toBeInTheDocument();
    });

    it("counts the flags the player has planted", async () => {
        // given
        const room = makeRoom();

        // when
        await renderPlaying(room, playerOne);

        // then
        expect(screen.getByText("2 / 10")).toBeInTheDocument();
    });

    it("gives a spectator both boards, both flag counts and no controls", async () => {
        // given
        const room = makeRoom();

        // when
        await renderPlaying(room, null, true);

        // then
        expect(screen.getByTestId("board-0")).toHaveAttribute("data-interactive", "false");
        expect(screen.getByTestId("board-0")).toHaveAttribute("data-pending", "");
        expect(screen.getByText("2 / 10")).toBeInTheDocument();
        expect(screen.getByText("1 / 10")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Resign" })).not.toBeInTheDocument();
    });

    it("names both seats for a spectator", async () => {
        // given
        const room = makeRoom();

        // when
        await renderPlaying(room, null, true);

        // then
        expect(screen.getAllByText("Battler").length).toBeGreaterThan(0);
        expect(screen.getAllByText("Erika").length).toBeGreaterThan(0);
    });

    it("offers a phone player a flag mode toggle", async () => {
        // given
        mobile.value = true;
        const { user } = await renderPlaying(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "⚑ Flag" }));

        // then
        expect(screen.getByTestId("board-0")).toHaveAttribute("data-flag-mode", "true");
        expect(screen.getByRole("button", { name: "⚑ Flag" })).toHaveAttribute("aria-pressed", "true");
    });

    it("keeps the flag toggle away from a desktop player", async () => {
        // given
        mobile.value = false;

        // when
        await renderPlaying(makeRoom(), playerOne);

        // then
        expect(screen.queryByRole("button", { name: "⚑ Flag" })).not.toBeInTheDocument();
    });

    it("only resigns once the player confirms", async () => {
        // given
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const { user } = await renderPlaying(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Resign this game?");
        expect(onResign).not.toHaveBeenCalled();
    });

    it("resigns when the player confirms", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const { user } = await renderPlaying(makeRoom(), playerOne);

        // when
        await user.click(screen.getByRole("button", { name: "Resign" }));

        // then
        await waitFor(() => {
            expect(onResign).toHaveBeenCalledOnce();
        });
    });

    it("tells the loser who blew themselves up", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-one",
            state: makeState({ phase: "finished", winner_slot: 0, reason: "mine_hit" }),
            stats: makeStats(),
        });

        // when
        renderView(room, playerOne);

        // then
        expect(screen.getByText("You won")).toBeInTheDocument();
        expect(screen.getByText("after Erika hit a mine")).toBeInTheDocument();
    });

    it("counts what each side managed once the game is over", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-two",
            state: makeState({ phase: "finished", winner_slot: 1, reason: "mine_hit" }),
            stats: makeStats(),
        });

        // when
        renderView(room, playerOne);

        // then
        expect(screen.getByText("You lost")).toBeInTheDocument();
        expect(screen.getByText("Cells revealed")).toBeInTheDocument();
        expect(screen.getByText("Flags placed")).toBeInTheDocument();
        expect(screen.getByText("Duration: 240")).toBeInTheDocument();
    });

    it("packs the resign control away once the game is over", () => {
        // given
        const room = makeRoom({
            status: "finished",
            winner_user_id: "u-one",
            state: makeState({ phase: "finished", winner_slot: 0, reason: "completed" }),
            stats: makeStats({ reason: "completed" }),
        });

        // when
        renderView(room, playerOne);

        // then
        expect(screen.queryByRole("button", { name: "Resign" })).not.toBeInTheDocument();
        expect(screen.getByText("by clearing the board")).toBeInTheDocument();
    });
});
