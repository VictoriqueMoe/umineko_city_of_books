import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { GameRoom, GameRoomPlayer, UserProfile } from "../../types/api";
import { ChessGamePage } from "./ChessGamePage";

const {
    useGameRoom,
    useAcceptGameInvite,
    useDeclineGameInvite,
    useSubmitGameAction,
    useResignGame,
    useOfferDraw,
    useAcceptDraw,
    useDeclineDraw,
    navigate,
} = vi.hoisted(() => ({
    useGameRoom: vi.fn(),
    useAcceptGameInvite: vi.fn(),
    useDeclineGameInvite: vi.fn(),
    useSubmitGameAction: vi.fn(),
    useResignGame: vi.fn(),
    useOfferDraw: vi.fn(),
    useAcceptDraw: vi.fn(),
    useDeclineDraw: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/gameRoom", () => ({ useGameRoom }));
vi.mock("../../api/mutations/gameRoom", () => ({
    useAcceptDraw,
    useAcceptGameInvite,
    useDeclineDraw,
    useDeclineGameInvite,
    useOfferDraw,
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
    onMove: (move: { from: string; to: string; promotion?: string }) => Promise<void>;
    onResign: () => Promise<void>;
    onOfferDraw: () => Promise<void>;
    onAcceptDraw: () => Promise<void>;
    onDeclineDraw: () => Promise<void>;
}

interface ChatStubProps {
    roomId: string;
    variant: string;
    watcherCount: number;
}

vi.mock("../../components/games/chess/ChessBoardView", () => ({
    ChessBoardView: (props: BoardStubProps) => (
        <section aria-label="chess board">
            <p>{`${props.isSpectator ? "spectating" : "playing"} ${props.room.id} as ${props.viewer?.display_name ?? "nobody"}`}</p>
            <button onClick={() => props.onMove({ from: "e2", to: "e4" })}>push a pawn</button>
            <button onClick={() => props.onMove({ from: "a7", to: "a8", promotion: "q" })}>promote a pawn</button>
            <button onClick={() => props.onResign()}>resign</button>
            <button onClick={() => props.onOfferDraw()}>offer a draw</button>
            <button onClick={() => props.onAcceptDraw()}>accept the draw</button>
            <button onClick={() => props.onDeclineDraw()}>decline the draw</button>
        </section>
    ),
}));

vi.mock("../../components/games/chat/GameChat.tsx", () => ({
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
        game_type: "chess",
        status: "active",
        state: {},
        created_by: "host",
        created_at: "2026-07-01T10:00:00Z",
        updated_at: "2026-07-01T10:00:00Z",
        players: [
            makePlayer({ user_id: "host", slot: 0, display_name: "Battler" }),
            makePlayer({ user_id: "guest", slot: 1, display_name: "Beatrice" }),
        ],
        watcher_count: 6,
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
    const offerDraw = vi.fn(() => Promise.resolve({}));
    const acceptDraw = vi.fn(() => Promise.resolve({}));
    const declineDraw = vi.fn(() => Promise.resolve({}));
    useAcceptGameInvite.mockReturnValue({ mutateAsync: acceptInvite });
    useDeclineGameInvite.mockReturnValue({ mutateAsync: declineInvite });
    useSubmitGameAction.mockReturnValue({ mutateAsync: submitAction });
    useResignGame.mockReturnValue({ mutateAsync: resign });
    useOfferDraw.mockReturnValue({ mutateAsync: offerDraw });
    useAcceptDraw.mockReturnValue({ mutateAsync: acceptDraw });
    useDeclineDraw.mockReturnValue({ mutateAsync: declineDraw });

    return { refetch, acceptInvite, declineInvite, submitAction, resign, offerDraw, acceptDraw, declineDraw };
}

function renderGame(user: UserProfile | null = host) {
    return renderWithProviders(<ChessGamePage />, { user, route: "/games/chess/room-1", path: "/games/chess/:id" });
}

describe("ChessGamePage", () => {
    it("renders nothing without a room id in the address", () => {
        // given
        stubRoom();

        // when
        const { container } = renderWithProviders(<ChessGamePage />, {
            user: host,
            route: "/games/chess",
            path: "/games/chess",
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
        stubRoom({ error: "no such match" });

        // when
        renderGame();

        // then
        expect(screen.getByText("no such match")).toBeInTheDocument();
    });

    it("sends the viewer to the live games from the error state", async () => {
        // given
        stubRoom({ error: "no such match" });
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

    it("offers the invitee the choice to accept or decline", () => {
        // given
        stubRoom({ room: makeRoom({ status: "pending" }) });

        // when
        renderGame(guest);

        // then
        expect(screen.getByText(/Battler has invited you to a chess game/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Accept" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Decline" })).toBeInTheDocument();
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

    it("returns to the games list from a pending invite", async () => {
        // given
        stubRoom({ room: makeRoom({ status: "pending" }) });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "Back" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/games");
    });

    it("shows the board and the player chat once the game is running", () => {
        // given
        stubRoom({ room: makeRoom() });

        // when
        renderGame(host);

        // then
        expect(screen.getByRole("region", { name: "chess board" })).toBeInTheDocument();
        expect(screen.getByText("playing room-1 as Battler")).toBeInTheDocument();
        expect(screen.getByText("player chat for room-1 with 6 watching")).toBeInTheDocument();
    });

    it("treats an outsider as a spectator with the spectator chat", () => {
        // given
        stubRoom({ room: makeRoom() });

        // when
        renderGame(onlooker);

        // then
        expect(screen.getByText("spectating room-1 as Ronove")).toBeInTheDocument();
        expect(screen.getByText("spectator chat for room-1 with 6 watching")).toBeInTheDocument();
    });

    it("treats a signed out visitor as a spectator", () => {
        // given
        stubRoom({ room: makeRoom() });

        // when
        renderGame(null);

        // then
        expect(screen.getByText("spectating room-1 as nobody")).toBeInTheDocument();
    });

    it("binds the action mutation to the open room", () => {
        // given
        stubRoom({ room: makeRoom({ id: "room-2" }) });

        // when
        renderGame(host);

        // then
        expect(useSubmitGameAction).toHaveBeenLastCalledWith("room-2");
    });

    it("submits a move with an empty promotion when none was chosen", async () => {
        // given
        const { submitAction } = stubRoom({ room: makeRoom() });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "push a pawn" }));

        // then
        expect(submitAction).toHaveBeenCalledWith({ from: "e2", to: "e4", promotion: "" });
    });

    it("carries the chosen promotion piece through with the move", async () => {
        // given
        const { submitAction } = stubRoom({ room: makeRoom() });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "promote a pawn" }));

        // then
        expect(submitAction).toHaveBeenCalledWith({ from: "a7", to: "a8", promotion: "q" });
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

    it("offers, accepts and declines a draw on the open room", async () => {
        // given
        const { offerDraw, acceptDraw, declineDraw } = stubRoom({ room: makeRoom({ id: "room-4" }) });
        const user = userEvent.setup();
        renderGame(host);

        // when
        await user.click(screen.getByRole("button", { name: "offer a draw" }));
        await user.click(screen.getByRole("button", { name: "accept the draw" }));
        await user.click(screen.getByRole("button", { name: "decline the draw" }));

        // then
        expect(offerDraw).toHaveBeenCalledWith("room-4");
        expect(acceptDraw).toHaveBeenCalledWith("room-4");
        expect(declineDraw).toHaveBeenCalledWith("room-4");
    });

    it("still shows the finished board rather than the invite screen", () => {
        // given
        stubRoom({ room: makeRoom({ status: "finished", winner_user_id: "host" }) });

        // when
        renderGame(host);

        // then
        expect(screen.getByRole("region", { name: "chess board" })).toBeInTheDocument();
    });
});
