import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeGamePlayer, makeGameRoom, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer, UserProfile } from "../../types/api";
import { PongGamePage } from "./PongGamePage";

const { useGameRoom, useAcceptGameInvite, useDeclineGameInvite, useResignGame, navigate } = vi.hoisted(() => ({
    useGameRoom: vi.fn(),
    useAcceptGameInvite: vi.fn(),
    useDeclineGameInvite: vi.fn(),
    useResignGame: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../hooks/queries/gameRoom", () => ({ useGameRoom }));
vi.mock("../../hooks/mutations/gameRoom", () => ({
    useAcceptGameInvite,
    useDeclineGameInvite,
    useResignGame,
}));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

interface BoardStubProps {
    room: GameRoom;
    viewer: UserProfile | null;
    isSpectator: boolean;
    onResign: () => Promise<void>;
}

interface ChatStubProps {
    roomId: string;
    variant: string;
    watcherCount: number;
}

vi.mock("../../components/games/pong/PongBoardView", () => ({
    PongBoardView: (props: BoardStubProps) => (
        <section aria-label="pong board">
            <p>{`${props.isSpectator ? "spectating" : "playing"} ${props.room.id} as ${props.viewer?.display_name ?? "nobody"}`}</p>
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
    return makeGamePlayer({ user_id: "host", ...overrides });
}

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return makeGameRoom(
        {},
        {
            game_type: "pong",
            created_by: "host",
            players: [
                makePlayer({ user_id: "host", slot: 0, display_name: "Battler" }),
                makePlayer({ user_id: "guest", slot: 1, display_name: "Beatrice" }),
            ],
            watcher_count: 1,
            ...overrides,
        },
    );
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
    const resign = vi.fn(() => Promise.resolve({}));
    useAcceptGameInvite.mockReturnValue({ mutateAsync: acceptInvite });
    useDeclineGameInvite.mockReturnValue({ mutateAsync: declineInvite });
    useResignGame.mockReturnValue({ mutateAsync: resign });

    return { refetch, acceptInvite, declineInvite, resign };
}

function renderGame(user: UserProfile | null = host) {
    return renderWithProviders(<PongGamePage />, {
        user,
        route: "/games/pong/room-1",
        path: "/games/pong/:id",
    });
}

describe("PongGamePage", () => {
    it("renders nothing without a room id in the address", () => {
        // given
        stubRoom();

        // when
        const { container } = renderWithProviders(<PongGamePage />, {
            user: host,
            route: "/games/pong",
            path: "/games/pong",
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
        stubRoom({ error: "the court is gone" });

        // when
        renderGame();

        // then
        expect(screen.getByText("the court is gone")).toBeInTheDocument();
    });

    it("sends the viewer to the live games from the error state", async () => {
        // given
        stubRoom({ error: "the court is gone" });
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

    it("holds a declined invite back from the board, which never got a state", () => {
        // given
        stubRoom({ room: makeRoom({ status: "declined" }) });

        // when
        renderGame();

        // then
        expect(screen.getByText("This match never started. The invite was declined or cancelled.")).toBeInTheDocument();
        expect(screen.queryByLabelText("pong board")).not.toBeInTheDocument();
    });

    it("warns the invitee that the ball is live from the countdown", () => {
        // given
        stubRoom({ room: makeRoom({ status: "pending" }) });

        // when
        renderGame(guest);

        // then
        expect(screen.getByText(/has invited you to a pong match/)).toHaveTextContent(
            "Battler has invited you to a pong match. Accept to start; the ball is live from the countdown, so keep a hand on the mouse.",
        );
    });

    it("tells the host they are waiting on their opponent", () => {
        // given
        stubRoom({ room: makeRoom({ status: "pending" }) });

        // when
        renderGame(host);

        // then
        expect(screen.getByText(/Waiting for/)).toHaveTextContent("Waiting for Beatrice to accept.");
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

    it("shows the court and the player chat once the match is running", () => {
        // given
        stubRoom({ room: makeRoom() });

        // when
        renderGame(host);

        // then
        expect(screen.getByRole("region", { name: "pong board" })).toBeInTheDocument();
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
