import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer } from "../../types/api";
import { PastGamesPage } from "./PastGamesPage";

const { useFinishedGameRooms } = vi.hoisted(() => ({ useFinishedGameRooms: vi.fn() }));

vi.mock("../../api/queries/gameRoom", () => ({ useFinishedGameRooms }));

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "player-0";
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
        status: "finished",
        state: {},
        created_by: "a",
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T12:00:00Z",
        finished_at: "2026-07-01T12:00:00Z",
        players: [
            makePlayer({ user_id: "a", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "b", slot: 1, display_name: "Beatrice" }),
        ],
        watcher_count: 0,
        ...overrides,
    };
}

interface StubOptions {
    rooms?: GameRoom[];
    total?: number;
    loading?: boolean;
}

function stubPast(options: StubOptions = {}) {
    useFinishedGameRooms.mockReturnValue({
        rooms: options.rooms ?? [],
        total: options.total ?? options.rooms?.length ?? 0,
        loading: options.loading ?? false,
    });
}

function manyRooms(count: number): GameRoom[] {
    const rooms: GameRoom[] = [];
    for (let i = 0; i < count; i++) {
        rooms.push(makeRoom({ id: `room-${i}`, winner_user_id: "a" }));
    }
    return rooms;
}

describe("PastGamesPage", () => {
    it("waits while the finished rooms are loading", () => {
        // given
        stubPast({ loading: true, rooms: [makeRoom()] });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("Battler vs Beatrice")).not.toBeInTheDocument();
    });

    it("says there is nothing to browse when no game has finished", () => {
        // given
        stubPast({ rooms: [] });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByText("No finished games yet.")).toBeInTheDocument();
    });

    it("asks for the first page of every game type", () => {
        // given
        stubPast();

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(useFinishedGameRooms).toHaveBeenCalledWith(undefined, 20, 0);
    });

    it("names the winner of a game the first seat took", () => {
        // given
        stubPast({ rooms: [makeRoom({ winner_user_id: "a" })] });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByText(/Battler won/)).toBeInTheDocument();
    });

    it("names the winner of a game the second seat took", () => {
        // given
        stubPast({ rooms: [makeRoom({ winner_user_id: "b" })] });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByText(/Beatrice won/)).toBeInTheDocument();
    });

    it("calls a game with no winner a draw", () => {
        // given
        stubPast({ rooms: [makeRoom()] });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByText(/Draw/)).toBeInTheDocument();
    });

    it("links each finished game to its board", () => {
        // given
        stubPast({ rooms: [makeRoom({ id: "room-42", game_type: "checkers" })] });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByRole("link", { name: /Battler vs Beatrice/ })).toHaveAttribute(
            "href",
            "/games/checkers/room-42",
        );
        expect(screen.getByText("finished")).toBeInTheDocument();
    });

    it("counts the visible slice of the whole archive", () => {
        // given
        stubPast({ rooms: manyRooms(20), total: 45 });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByText("1-20 of 45")).toBeInTheDocument();
    });

    it("cannot go back from the first page", () => {
        // given
        stubPast({ rooms: manyRooms(20), total: 45 });

        // when
        renderWithProviders(<PastGamesPage />);

        // then
        expect(screen.getByRole("button", { name: "Previous" })).toBeDisabled();
        expect(screen.getByRole("button", { name: "Next" })).toBeEnabled();
    });

    it("asks for the next slice when the reader pages forward", async () => {
        // given
        stubPast({ rooms: manyRooms(20), total: 45 });
        const user = userEvent.setup();
        renderWithProviders(<PastGamesPage />);

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(useFinishedGameRooms).toHaveBeenLastCalledWith(undefined, 20, 20);
        expect(screen.getByText("21-40 of 45")).toBeInTheDocument();
    });

    it("asks for the previous slice when the reader pages back", async () => {
        // given
        stubPast({ rooms: manyRooms(20), total: 45 });
        const user = userEvent.setup();
        renderWithProviders(<PastGamesPage />);
        await user.click(screen.getByRole("button", { name: "Next" }));

        // when
        await user.click(screen.getByRole("button", { name: "Previous" }));

        // then
        expect(useFinishedGameRooms).toHaveBeenLastCalledWith(undefined, 20, 0);
        expect(screen.getByText("1-20 of 45")).toBeInTheDocument();
    });

    it("cannot go forward once the last slice is showing", async () => {
        // given
        stubPast({ rooms: manyRooms(20), total: 35 });
        const user = userEvent.setup();
        renderWithProviders(<PastGamesPage />);

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(screen.getByRole("button", { name: "Next" })).toBeDisabled();
        expect(screen.getByText("21-35 of 35")).toBeInTheDocument();
    });
});
