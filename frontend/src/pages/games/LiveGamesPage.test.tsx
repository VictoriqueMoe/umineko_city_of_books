import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { createTestQueryClient, renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer } from "../../types/api";
import { LiveGamesPage } from "./LiveGamesPage";

const { useLiveGameRooms } = vi.hoisted(() => ({ useLiveGameRooms: vi.fn() }));

vi.mock("../../api/queries/gameRoom", () => ({ useLiveGameRooms }));

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
        status: "active",
        state: {},
        created_by: "a",
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        players: [
            makePlayer({ user_id: "a", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "b", slot: 1, display_name: "Beatrice" }),
        ],
        watcher_count: 4,
        ...overrides,
    };
}

interface StubOptions {
    rooms?: GameRoom[];
    loading?: boolean;
    error?: string;
}

function stubLive(options: StubOptions = {}) {
    useLiveGameRooms.mockReturnValue({
        rooms: options.rooms ?? [],
        total: options.rooms?.length ?? 0,
        loading: options.loading ?? false,
        error: options.error ?? "",
        refresh: vi.fn(),
    });
}

describe("LiveGamesPage", () => {
    it("waits while the live rooms are loading", () => {
        // given
        stubLive({ loading: true, rooms: [makeRoom()] });

        // when
        renderWithProviders(<LiveGamesPage />);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("Battler vs Beatrice")).not.toBeInTheDocument();
    });

    it("says nothing is running when no game is live", () => {
        // given
        stubLive({ rooms: [] });

        // when
        renderWithProviders(<LiveGamesPage />);

        // then
        expect(screen.getByText("No games in progress right now.")).toBeInTheDocument();
    });

    it("reports the reason the live rooms could not be loaded", () => {
        // given
        stubLive({ error: "the golden land is closed" });

        // when
        renderWithProviders(<LiveGamesPage />);

        // then
        expect(screen.getByText("the golden land is closed")).toBeInTheDocument();
    });

    it("lists each live room with its seats, game type and watchers", () => {
        // given
        stubLive({ rooms: [makeRoom({ id: "room-5", game_type: "othello", watcher_count: 9 })] });

        // when
        renderWithProviders(<LiveGamesPage />);

        // then
        expect(screen.getByText("Battler vs Beatrice")).toBeInTheDocument();
        expect(screen.getByText(/othello/)).toBeInTheDocument();
        expect(screen.getByText(/9 watching/)).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Battler vs Beatrice/ })).toHaveAttribute(
            "href",
            "/games/othello/room-5",
        );
    });

    it("marks a seat nobody has taken with a placeholder", () => {
        // given
        stubLive({ rooms: [makeRoom({ players: [makePlayer({ user_id: "b", slot: 1, display_name: "Beatrice" })] })] });

        // when
        renderWithProviders(<LiveGamesPage />);

        // then
        expect(screen.getByText("? vs Beatrice")).toBeInTheDocument();
    });

    it("refreshes the live list when the page mounts", () => {
        // given
        stubLive();
        const queryClient = createTestQueryClient();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        renderWithProviders(<LiveGamesPage />, { queryClient });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["game-rooms", "live"] });
    });

    it("keeps every live room, not just the first few", () => {
        // given
        const rooms: GameRoom[] = [];
        for (let i = 0; i < 8; i++) {
            rooms.push(makeRoom({ id: `room-${i}` }));
        }
        stubLive({ rooms });

        // when
        renderWithProviders(<LiveGamesPage />);

        // then
        expect(screen.getAllByText("live")).toHaveLength(8);
    });
});
