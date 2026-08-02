import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer, UserProfile } from "../../types/api";
import { MinesweeperGamePage } from "./MinesweeperGamePage";

const { useGameRoom, useAcceptGameInvite, useDeclineGameInvite, useSubmitGameAction, useResignGame, navigate } =
    vi.hoisted(() => ({
        useGameRoom: vi.fn(),
        useAcceptGameInvite: vi.fn(),
        useDeclineGameInvite: vi.fn(),
        useSubmitGameAction: vi.fn(),
        useResignGame: vi.fn(),
        navigate: vi.fn(),
    }));

vi.mock("../../api/queries/gameRoom", () => ({ useGameRoom }));
vi.mock("../../api/mutations/gameRoom", () => ({
    useAcceptGameInvite,
    useDeclineGameInvite,
    useResignGame,
    useSubmitGameAction,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

interface BoardStubProps {
    room: GameRoom;
    viewer: UserProfile | null;
    isSpectator: boolean;
    onAction: (payload: Record<string, unknown>) => Promise<void>;
    onResign: () => Promise<void>;
}

interface ChatStubProps {
    roomId: string;
    variant: string;
    watcherCount: number;
}

vi.mock("../../components/games/minesweeper/MinesweeperBoardView", () => ({
    MinesweeperBoardView: (props: BoardStubProps) => (
        <section aria-label="minesweeper board">
            <p>{`${props.isSpectator ? "spectating" : "playing"} ${props.room.id} as ${props.viewer?.display_name ?? "nobody"}`}</p>
            <button onClick={() => props.onAction({ type: "reveal", row: 2, col: 3 })}>reveal a cell</button>
            <button onClick={() => props.onAction({ type: "flag", row: 0, col: 0 })}>flag a cell</button>
            <button onClick={() => props.onAction({ type: "pick_character", character: "bernkastel" })}>
                pick a character
            </button>
            <button onClick={() => props.onResign()}>resign</button>
        </section>
    ),
}));

vi.mock("../../components/games/chat/GameChat", () => ({
    GameChat: (props: ChatStubProps) => (
        <section aria-label="game chat">{`${props.variant} chat for ${props.roomId} with ${props.watcherCount} watching`}</section>
    ),
}));

const host = makeUser({ id: "host", username: "battler", display_name: "Battler" });
const guest = makeUser({ id: "guest", username: "beatrice", display_name: "Beatrice" });
const onlooker = makeUser({ id: "onlooker", username: "ronove", display_name: "Ronove" });

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "host";
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
        state: {},
        created_by: "host",
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        players: [
            makePlayer({ user_id: "host", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "guest", slot: 1, display_name: "Beatrice" }),
        ],
        watcher_count: 1,
        ...overrides,
    };
}

interface StubOptions {
    room?: GameRoom | null;
    loading?: boolean;
    error?: string;
    accept?: () => Promise<unknown>;
    decline?: () => Promise<unknown>;
}

function stubRoom(options: StubOptions = {}) {
    const refetch = vi.fn(() => Promise.resolve());
    useGameRoom.mockReturnValue({
        room: options.room ?? null,
        loading: options.loading ?? false,
        error: options.error ?? "",
        refetch,
        wsConnected: true,
    });
    const acceptInvite = vi.fn(options.accept ?? (() => Promise.resolve({})));
    const declineInvite = vi.fn(options.decline ?? (() => Promise.resolve({})));
    const submitAction = vi.fn(() => Promise.resolve({}));
    const resign = vi.fn(() => Promise.resolve({}));
    useAcceptGameInvite.mockReturnValue({ mutateAsync: acceptInvite });
    useDeclineGameInvite.mockReturnValue({ mutateAsync: declineInvite });
    useSubmitGameAction.mockReturnValue({ mutateAsync: submitAction });
    useResignGame.mockReturnValue({ mutateAsync: resign });

    return { refetch, acceptInvite, declineInvite, submitAction, resign };
}

function renderGame(user: UserProfile | null = host) {
    return renderWithProviders(<MinesweeperGamePage />, {
        user,
        route: "/games/minesweeper/room-1",
        path: "/games/minesweeper/:id",
    });
}

describe("MinesweeperGamePage", () => {
    it("renders nothing without a room id in the address", () => {
        // given
        stubRoom();

        // when
        const { container } = renderWithProviders(<MinesweeperGamePage />, {
            user: host,
            route: "/games/minesweeper",
            path: "/games/minesweeper",
        });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("waits while the room is being fetched", () => {
        // given
        stubRoom({ loading: true });

        // when
        renderGame();

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
    });

    it("reports why the room could not be opened", () => {
        // given
        stubRoom({ error: "the minefield is gone" });

        // when
        renderGame();

        // then
        expect(screen.getByText("the minefield is gone")).toBeInTheDocument();
    });

    it("sends the viewer to the live games from the error state", async () => {
        // given
        stubRoom({ error: "the minefield is gone" });
        const user = userEvent.setup();
        renderGame();

        // when
        await user.click(screen.getByRole("button", { name: "Back" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/games/live");
    });

    it("renders nothing when the room came back empty", () => {
        // given
        stubRoom({ room: null });

        // when
        const { container } = renderGame();

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("keeps a pending invite private from anyone who is not playing", () => {
        // given
        stubRoom({ room: makeRoom({ status: "pending" }) });

        // when
        renderGame(onlooker);

        // then
        expect(screen.getByText(/invites are private/)).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Accept" })).not.toBeInTheDocument();
    });

    it("explains the simultaneous race to the invitee", () => {
        // given
        stubRoom({ room: makeRoom({ status: "pending" }) });

        // when
        renderGame(guest);

        // then
        expect(screen.getByText(/Battler has invited you to a minesweeper match/)).toBeInTheDocument();
        expect(screen.getByText(/race to clear the board/)).toBeInTheDocument();
    });

    it("tells the host they are waiting on their opponent", () => {
        // given
        stubRoom({ room: makeRoom({ status: "pending" }) });

        // when
        renderGame(host);

        // then
        expect(screen.getByText("Waiting for Beatrice to accept.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Accept" })).not.toBeInTheDocument();
    });

    it("accepts the invite and refetches the room", async () => {
        // given
        const { acceptInvite, refetch } = stubRoom({ room: makeRoom({ id: "room-8", status: "pending" }) });
        const user = userEvent.setup();
        renderGame(guest);

        // when
        await user.click(screen.getByRole("button", { name: "Accept" }));

        // then
        expect(acceptInvite).toHaveBeenCalledWith("room-8");
        expect(refetch).toHaveBeenCalledOnce();
    });

    it("shows why an invite could not be accepted", async () => {
        // given
        stubRoom({
            room: makeRoom({ status: "pending" }),
            accept: () => Promise.reject(new Error("the invite expired")),
        });
        const user = userEvent.setup();
        renderGame(guest);

        // when
        await user.click(screen.getByRole("button", { name: "Accept" }));

        // then
        expect(await screen.findByText("the invite expired")).toBeInTheDocument();
    });

    it("declines the invite and returns to the games list", async () => {
        // given
        const { declineInvite } = stubRoom({ room: makeRoom({ id: "room-8", status: "pending" }) });
        const user = userEvent.setup();
        renderGame(guest);

        // when
        await user.click(screen.getByRole("button", { name: "Decline" }));

        // then
        expect(declineInvite).toHaveBeenCalledWith("room-8");
        expect(navigate).toHaveBeenCalledWith("/games");
    });

    it("keeps the invitee in place and says why the decline failed", async () => {
        // given
        stubRoom({
            room: makeRoom({ id: "room-8", status: "pending" }),
            decline: () => Promise.reject(new Error("that invite has already gone")),
        });
        const user = userEvent.setup();
        renderGame(guest);

        // when
        await user.click(screen.getByRole("button", { name: "Decline" }));

        // then
        expect(await screen.findByText("that invite has already gone")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalledWith("/games");
    });

    it("shows the board and the player chat once the game is running", () => {
        // given
        stubRoom({ room: makeRoom() });

        // when
        renderGame(host);

        // then
        expect(screen.getByRole("region", { name: "minesweeper board" })).toBeInTheDocument();
        expect(screen.getByText("playing room-1 as Battler")).toBeInTheDocument();
        expect(screen.getByText("player chat for room-1 with 1 watching")).toBeInTheDocument();
    });

    it("treats an outsider as a spectator with the spectator chat", () => {
        // given
        stubRoom({ room: makeRoom() });

        // when
        renderGame(onlooker);

        // then
        expect(screen.getByText("spectating room-1 as Ronove")).toBeInTheDocument();
        expect(screen.getByText("spectator chat for room-1 with 1 watching")).toBeInTheDocument();
    });

    it("binds the action mutation to the open room", () => {
        // given
        stubRoom({ room: makeRoom({ id: "room-2" }) });

        // when
        renderGame(host);

        // then
        expect(useSubmitGameAction).toHaveBeenLastCalledWith("room-2");
    });

    it("forwards a reveal untouched to the action mutation", async () => {
        // given
        const { submitAction } = stubRoom({ room: makeRoom() });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "reveal a cell" }));

        // then
        expect(submitAction).toHaveBeenCalledWith({ type: "reveal", row: 2, col: 3 });
    });

    it("forwards a flag untouched to the action mutation", async () => {
        // given
        const { submitAction } = stubRoom({ room: makeRoom() });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "flag a cell" }));

        // then
        expect(submitAction).toHaveBeenCalledWith({ type: "flag", row: 0, col: 0 });
    });

    it("forwards a character choice untouched to the action mutation", async () => {
        // given
        const { submitAction } = stubRoom({ room: makeRoom() });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "pick a character" }));

        // then
        expect(submitAction).toHaveBeenCalledWith({ type: "pick_character", character: "bernkastel" });
    });

    it("resigns the open room", async () => {
        // given
        const { resign } = stubRoom({ room: makeRoom({ id: "room-3" }) });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "resign" }));

        // then
        expect(resign).toHaveBeenCalledWith("room-3");
    });
});
