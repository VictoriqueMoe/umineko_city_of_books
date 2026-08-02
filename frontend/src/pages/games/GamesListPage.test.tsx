import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer } from "../../types/api";
import { GamesListPage } from "./GamesListPage";

const { useMyGameRooms, useDeclineGameInvite, useCancelGameInvite, navigate } = vi.hoisted(() => ({
    useMyGameRooms: vi.fn(),
    useDeclineGameInvite: vi.fn(),
    useCancelGameInvite: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/gameRoom", () => ({ useMyGameRooms }));
vi.mock("../../api/mutations/gameRoom", () => ({ useDeclineGameInvite, useCancelGameInvite }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const viewer = makeUser({ id: "me", username: "me", display_name: "Me" });

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "me";
    return {
        user_id: id,
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        role: "player",
        slot: 0,
        joined: true,
        connected: true,
        user: { id, username: "beatrice", display_name: "Beatrice" },
        ...overrides,
    };
}

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "chess",
        status: "pending",
        state: {},
        created_by: "beatrice-id",
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        players: [
            makePlayer({ user_id: "me", slot: 0, display_name: "Me" }),
            makePlayer({ user_id: "beatrice-id", slot: 1, display_name: "Beatrice" }),
        ],
        watcher_count: 0,
        ...overrides,
    };
}

interface StubOptions {
    rooms?: GameRoom[];
    loading?: boolean;
    error?: string;
    decline?: () => Promise<unknown>;
    cancel?: () => Promise<unknown>;
}

function stubGames(options: StubOptions = {}) {
    const refresh = vi.fn(() => Promise.resolve());
    useMyGameRooms.mockReturnValue({
        rooms: options.rooms ?? [],
        total: options.rooms?.length ?? 0,
        loading: options.loading ?? false,
        error: options.error ?? "",
        refresh,
    });
    const declineAsync = vi.fn(options.decline ?? (() => Promise.resolve({})));
    const cancelAsync = vi.fn(options.cancel ?? (() => Promise.resolve({})));
    useDeclineGameInvite.mockReturnValue({ mutateAsync: declineAsync });
    useCancelGameInvite.mockReturnValue({ mutateAsync: cancelAsync });

    return { refresh, declineAsync, cancelAsync };
}

function renderPage() {
    return renderWithProviders(<GamesListPage />, { user: viewer, route: "/games" });
}

describe("GamesListPage", () => {
    it("explains that the games feature is still in beta", () => {
        // given
        stubGames();

        // when
        renderPage();

        // then
        expect(screen.getByText("Games are in beta")).toBeInTheDocument();
    });

    it("offers a tile for every playable game", () => {
        // given
        stubGames();

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: /^Chess/ })).toHaveAttribute("href", "/games/chess");
        expect(screen.getByRole("link", { name: /^Checkers/ })).toHaveAttribute("href", "/games/checkers");
        expect(screen.getByRole("link", { name: /^Othello/ })).toHaveAttribute("href", "/games/othello");
        expect(screen.getByRole("link", { name: /^Minesweeper/ })).toHaveAttribute("href", "/games/minesweeper");
        expect(screen.getByRole("link", { name: /^Snakes & Ladders/ })).toHaveAttribute(
            "href",
            "/games/snakes_and_ladders",
        );
    });

    it("waits on the invites section while the rooms are still loading", () => {
        // given
        stubGames({ loading: true });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("No pending invites.")).not.toBeInTheDocument();
    });

    it("shows an empty state for every section when the player has no games", () => {
        // given
        stubGames({ rooms: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No pending invites.")).toBeInTheDocument();
        expect(screen.getByText("None.")).toBeInTheDocument();
        expect(screen.getByText("None in progress.")).toBeInTheDocument();
        expect(screen.getByText("None yet.")).toBeInTheDocument();
    });

    it("reports the reason the rooms could not be loaded", () => {
        // given
        stubGames({ error: "the servants are asleep" });

        // when
        renderPage();

        // then
        expect(screen.getByText("the servants are asleep")).toBeInTheDocument();
    });

    it("lists an invite somebody else sent under invites for you", () => {
        // given
        stubGames({ rooms: [makeRoom()] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Beatrice invited you to Chess")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "View and accept" })).toBeInTheDocument();
    });

    it("lists an invite the player sent under waiting on opponent", () => {
        // given
        stubGames({ rooms: [makeRoom({ created_by: "me", game_type: "othello" })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Othello vs Beatrice")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
        expect(screen.getByText("No pending invites.")).toBeInTheDocument();
    });

    it("opens the room when the player goes to accept an invite", async () => {
        // given
        stubGames({ rooms: [makeRoom({ id: "room-9", game_type: "checkers" })] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "View and accept" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/games/checkers/room-9");
    });

    it("declines an invite and refreshes the list", async () => {
        // given
        const { declineAsync, refresh } = stubGames({ rooms: [makeRoom({ id: "room-3" })] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Decline" }));

        // then
        expect(declineAsync).toHaveBeenCalledWith("room-3");
        await waitFor(() => {
            expect(refresh).toHaveBeenCalledOnce();
        });
    });

    it("leaves the list alone and says why when declining fails", async () => {
        // given
        const { refresh } = stubGames({
            rooms: [makeRoom({ id: "room-3" })],
            decline: () => Promise.reject(new Error("that invite has already gone")),
        });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Decline" }));

        // then
        expect(await screen.findByText("that invite has already gone")).toBeInTheDocument();
        expect(refresh).not.toHaveBeenCalled();
    });

    it("says why cancelling an invite failed", async () => {
        // given
        stubGames({
            rooms: [makeRoom({ id: "room-7", created_by: "me" })],
            cancel: () => Promise.reject(new Error("that invite has already gone")),
        });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(await screen.findByText("that invite has already gone")).toBeInTheDocument();
    });

    it("asks before cancelling an invite the player sent", async () => {
        // given
        const { cancelAsync } = stubGames({ rooms: [makeRoom({ created_by: "me" })] });
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(confirm).toHaveBeenCalledWith("Cancel this invite?");
        expect(cancelAsync).not.toHaveBeenCalled();
    });

    it("cancels the invite once the player confirms", async () => {
        // given
        const { cancelAsync, refresh } = stubGames({ rooms: [makeRoom({ id: "room-7", created_by: "me" })] });
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(cancelAsync).toHaveBeenCalledWith("room-7");
        await waitFor(() => {
            expect(refresh).toHaveBeenCalledOnce();
        });
    });

    it("marks an active game where it is the player's move", () => {
        // given
        stubGames({ rooms: [makeRoom({ id: "room-4", status: "active", turn_user_id: "me" })] });

        // when
        renderPage();

        // then
        expect(screen.getByText(/^Your turn/)).toBeInTheDocument();
        expect(screen.getByText("your turn")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Chess vs Beatrice/ })).toHaveAttribute("href", "/games/chess/room-4");
    });

    it("marks an active game where the opponent is to move", () => {
        // given
        stubGames({ rooms: [makeRoom({ status: "active", turn_user_id: "beatrice-id" })] });

        // when
        renderPage();

        // then
        expect(screen.getByText(/^Their turn/)).toBeInTheDocument();
        expect(screen.getByText("active")).toBeInTheDocument();
    });

    it("calls a finished game with no winner a draw", () => {
        // given
        stubGames({ rooms: [makeRoom({ status: "finished", finished_at: "2026-07-02T10:00:00Z" })] });

        // when
        renderPage();

        // then
        expect(screen.getByText(/^Draw/)).toBeInTheDocument();
    });

    it("calls a finished game the player won a win", () => {
        // given
        stubGames({ rooms: [makeRoom({ status: "finished", winner_user_id: "me" })] });

        // when
        renderPage();

        // then
        expect(screen.getByText(/^Won/)).toBeInTheDocument();
    });

    it("calls a finished game the opponent won a loss", () => {
        // given
        stubGames({ rooms: [makeRoom({ status: "finished", winner_user_id: "beatrice-id" })] });

        // when
        renderPage();

        // then
        expect(screen.getByText(/^Lost/)).toBeInTheDocument();
    });

    it("files an abandoned game under past games with its status as the outcome", () => {
        // given
        stubGames({ rooms: [makeRoom({ status: "abandoned" })] });

        // when
        renderPage();

        // then
        expect(screen.getAllByText(/^abandoned/)).toHaveLength(2);
        expect(screen.getByText("None in progress.")).toBeInTheDocument();
    });

    it("names an opponent who has left the room as unknown", () => {
        // given
        stubGames({
            rooms: [makeRoom({ status: "active", players: [makePlayer({ user_id: "me", display_name: "Me" })] })],
        });

        // when
        renderPage();

        // then
        expect(screen.getByText("Chess vs Unknown")).toBeInTheDocument();
    });

    it("links through to the live games page", () => {
        // given
        stubGames();

        // when
        renderPage();

        // then
        expect(screen.getByRole("link", { name: "Live Games" })).toHaveAttribute("href", "/games/live");
    });
});
