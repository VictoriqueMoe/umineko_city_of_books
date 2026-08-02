import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer, GameScoreboardRow, UserProfile } from "../../types/api";
import { GameHubPage } from "./GameHubPage";

const { useGameScoreboard, useLiveGameRooms, navigate } = vi.hoisted(() => ({
    useGameScoreboard: vi.fn(),
    useLiveGameRooms: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/gameRoom", () => ({ useGameScoreboard, useLiveGameRooms }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const viewer = makeUser({ id: "me", username: "me", display_name: "Me" });

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
        watcher_count: 3,
        ...overrides,
    };
}

function makeScoreboardRow(overrides: Partial<GameScoreboardRow> = {}): GameScoreboardRow {
    return {
        user: { id: "a", username: "battler", display_name: "Battler" },
        wins: 2,
        losses: 1,
        draws: 0,
        games_played: 3,
        win_rate: 0.6667,
        ...overrides,
    };
}

interface StubOptions {
    rows?: GameScoreboardRow[];
    rooms?: GameRoom[];
    scoreboardLoading?: boolean;
    liveLoading?: boolean;
}

function stubHub(options: StubOptions = {}) {
    useGameScoreboard.mockReturnValue({
        data: options.rows ? { game_type: "chess", rows: options.rows } : null,
        loading: options.scoreboardLoading ?? false,
    });
    useLiveGameRooms.mockReturnValue({
        rooms: options.rooms ?? [],
        total: options.rooms?.length ?? 0,
        loading: options.liveLoading ?? false,
        error: "",
        refresh: vi.fn(),
    });
}

function renderHub(type = "chess", user: UserProfile | null = viewer) {
    return renderWithProviders(<GameHubPage />, { user, route: `/games/${type}`, path: "/games/:type" });
}

describe("GameHubPage", () => {
    it("says the game does not exist for an unknown type", () => {
        // given
        stubHub();

        // when
        renderHub("mahjong");

        // then
        expect(screen.getByText("Unknown game")).toBeInTheDocument();
        expect(screen.getByText("That game type does not exist.")).toBeInTheDocument();
    });

    it("sends the visitor back to the games list from an unknown type", async () => {
        // given
        stubHub();
        const user = userEvent.setup();
        renderHub("mahjong");

        // when
        await user.click(screen.getByRole("button", { name: "Back to Games" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/games");
    });

    it("asks for the scoreboard and the live list of the game being viewed", () => {
        // given
        stubHub();

        // when
        renderHub("othello");

        // then
        expect(useGameScoreboard).toHaveBeenCalledWith("othello");
        expect(useLiveGameRooms).toHaveBeenCalledWith("othello");
    });

    it("leaves both queries without a type when the game is unknown", () => {
        // given
        stubHub();

        // when
        renderHub("mahjong");

        // then
        expect(useGameScoreboard).toHaveBeenCalledWith(undefined);
        expect(useLiveGameRooms).toHaveBeenCalledWith(undefined);
    });

    it("introduces the game with its tagline and rules", () => {
        // given
        stubHub();

        // when
        renderHub("checkers");

        // then
        expect(screen.getByRole("heading", { name: "Checkers" })).toBeInTheDocument();
        expect(screen.getByText(/Classic American draughts/)).toBeInTheDocument();
        expect(screen.getByText("How to play checkers")).toBeInTheDocument();
    });

    it("invites a signed in member to start a new game", () => {
        // given
        stubHub();

        // when
        renderHub("chess");

        // then
        expect(screen.getByRole("link", { name: "Start a new chess game" })).toHaveAttribute(
            "href",
            "/games/chess/new",
        );
    });

    it("asks a signed out visitor to sign in before playing", () => {
        // given
        stubHub();

        // when
        renderHub("chess", null);

        // then
        expect(screen.getByRole("link", { name: "Sign in to play" })).toHaveAttribute("href", "/login");
        expect(screen.queryByRole("link", { name: /Start a new/ })).not.toBeInTheDocument();
    });

    it("waits on the live list while the live games are still loading", () => {
        // given
        stubHub({ liveLoading: true, rooms: [makeRoom()] });

        // when
        renderHub();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("Battler vs Beatrice")).not.toBeInTheDocument();
    });

    it("keeps the live games on screen while only the scoreboard is still loading", () => {
        // given
        stubHub({ scoreboardLoading: true, rooms: [makeRoom()] });

        // when
        renderHub();

        // then
        expect(screen.getByText("Battler vs Beatrice")).toBeInTheDocument();
    });

    it("waits on the scoreboard instead of calling it empty while it loads", () => {
        // given
        stubHub({ scoreboardLoading: true });

        // when
        renderHub();

        // then
        expect(screen.queryByText("No completed games yet. Be the first to finish a match.")).not.toBeInTheDocument();
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("says nothing is running when no game of this type is live", () => {
        // given
        stubHub({ rooms: [] });

        // when
        renderHub("othello");

        // then
        expect(screen.getByText("No othello games in progress right now.")).toBeInTheDocument();
    });

    it("lists the live games with their seats and watcher counts", () => {
        // given
        stubHub({ rooms: [makeRoom({ id: "room-2", watcher_count: 12 })] });

        // when
        renderHub();

        // then
        expect(screen.getByText("Battler vs Beatrice")).toBeInTheDocument();
        expect(screen.getByText("12 watching")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Battler vs Beatrice/ })).toHaveAttribute(
            "href",
            "/games/chess/room-2",
        );
    });

    it("shows at most five live games", () => {
        // given
        const rooms: GameRoom[] = [];
        for (let i = 0; i < 7; i++) {
            rooms.push(makeRoom({ id: `room-${i}` }));
        }
        stubHub({ rooms });

        // when
        renderHub();

        // then
        expect(screen.getAllByText("live")).toHaveLength(5);
    });

    it("marks an empty seat with a placeholder", () => {
        // given
        stubHub({ rooms: [makeRoom({ players: [makePlayer({ user_id: "a", slot: 0 })] })] });

        // when
        renderHub();

        // then
        expect(screen.getByText("Battler vs ?")).toBeInTheDocument();
    });

    it("encourages the first finished match when the scoreboard is empty", () => {
        // given
        stubHub({ rows: [] });

        // when
        renderHub();

        // then
        expect(screen.getByText("No completed games yet. Be the first to finish a match.")).toBeInTheDocument();
        expect(screen.queryByRole("table")).not.toBeInTheDocument();
    });

    it("ranks the scoreboard rows and shows each win rate as a percentage", () => {
        // given
        stubHub({
            rows: [
                makeScoreboardRow(),
                makeScoreboardRow({
                    user: { id: "b", username: "beatrice", display_name: "Beatrice" },
                    wins: 1,
                    losses: 3,
                    draws: 1,
                    games_played: 5,
                    win_rate: 0.2,
                }),
            ],
        });

        // when
        renderHub();

        // then
        const rows = screen.getAllByRole("row");
        expect(rows).toHaveLength(3);
        expect(rows[1]).toHaveTextContent("66.7%");
        expect(rows[2]).toHaveTextContent("20.0%");
    });

    it("links onward to the live and past games pages", () => {
        // given
        stubHub();

        // when
        renderHub();

        // then
        expect(screen.getByRole("link", { name: "Live games" })).toHaveAttribute("href", "/games/live");
        expect(screen.getByRole("link", { name: "Past games" })).toHaveAttribute("href", "/games/past");
    });
});
