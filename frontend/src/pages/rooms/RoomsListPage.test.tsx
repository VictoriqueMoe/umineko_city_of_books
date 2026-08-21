import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../../test-utils/render";
import type { ChatRoom, UserProfile, WSMessage } from "../../types/api";
import { RoomsListPage } from "./RoomsListPage";

const mocks = vi.hoisted(() => ({
    listMyChatRooms: vi.fn(),
    listPublicChatRooms: vi.fn(),
    getUserRooms: vi.fn(),
    joinMutateAsync: vi.fn(),
}));

vi.mock("../../api/endpoints", () => ({
    listMyChatRooms: mocks.listMyChatRooms,
    listPublicChatRooms: mocks.listPublicChatRooms,
    getUserRooms: mocks.getUserRooms,
    getChatRoomMembers: vi.fn(),
    getChatRoomPinnedMessages: vi.fn(),
    getChatUnreadCount: vi.fn(),
    getRoomMessages: vi.fn(),
    getRoomMessagesBefore: vi.fn(),
    listChatRoomBans: vi.fn(),
    listChatRoomBannedWords: vi.fn(),
    resolveDMRoom: vi.fn(),
}));

vi.mock("../../api/mutations/chat", () => ({
    useJoinChatRoom: () => ({ mutateAsync: mocks.joinMutateAsync }),
}));

vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: (props: { page: string }) => <div>{`rules for ${props.page}`}</div>,
}));

vi.mock("../../components/chat/CreateRoomModal/CreateRoomModal", () => ({
    CreateRoomModal: (props: { isOpen: boolean }) => (props.isOpen ? <div data-testid="create-room" /> : null),
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Tea Parlour",
        description: "",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 4,
        hot_score: 0,
        members: [],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

interface ListOptions {
    hosted?: ChatRoom[];
    hostedTotal?: number;
    joined?: ChatRoom[];
    joinedTotal?: number;
    discover?: ChatRoom[];
    discoverTotal?: number;
    system?: ChatRoom[];
}

function stubLists(options: ListOptions = {}) {
    mocks.listMyChatRooms.mockImplementation((params: { role?: string }) => {
        if (params.role === "host") {
            return Promise.resolve({
                rooms: options.hosted ?? [],
                total: options.hostedTotal ?? options.hosted?.length ?? 0,
            });
        }

        return Promise.resolve({
            rooms: options.joined ?? [],
            total: options.joinedTotal ?? options.joined?.length ?? 0,
        });
    });
    mocks.listPublicChatRooms.mockResolvedValue({
        rooms: options.discover ?? [],
        total: options.discoverTotal ?? options.discover?.length ?? 0,
    });
    mocks.getUserRooms.mockResolvedValue({ rooms: options.system ?? [] });
}

function renderList(options: { user?: UserProfile | null } = {}) {
    const listeners: ((msg: WSMessage) => void)[] = [];
    const queryClient = createTestQueryClient();
    const result = renderWithProviders(<RoomsListPage />, {
        user: options.user === undefined ? viewer : options.user,
        queryClient,
        notification: {
            addWSListener: listener => {
                listeners.push(listener);
                return () => {};
            },
        },
    });

    return { ...result, listeners, queryClient };
}

beforeEach(() => {
    stubLists();
    mocks.joinMutateAsync.mockResolvedValue(makeRoom({ id: "room-joined" }));
});

describe("RoomsListPage layout", () => {
    it("explains what chat rooms are and shows their rules", async () => {
        // given
        stubLists();

        // when
        renderList();

        // then
        expect(await screen.findByText("What are Chat Rooms?")).toBeInTheDocument();
        expect(screen.getByText("rules for chat_rooms")).toBeInTheDocument();
    });

    it("keeps the personal sections away from a signed out visitor", async () => {
        // given
        const user = null;

        // when
        renderList({ user });

        // then
        await screen.findByText("What are Chat Rooms?");
        expect(screen.queryByRole("heading", { name: /My Rooms/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: /Joined Rooms/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /New Room/ })).not.toBeInTheDocument();
    });

    it("gives a signed in member their own sections", async () => {
        // given
        stubLists();

        // when
        renderList();

        // then
        expect(await screen.findByRole("heading", { name: /My Rooms/ })).toBeInTheDocument();
        expect(screen.getByRole("heading", { name: /Joined Rooms/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /New Room/ })).toBeInTheDocument();
    });

    it("opens the create room dialog when asked", async () => {
        // given
        const user = userEvent.setup();
        renderList();
        const create = await screen.findByRole("button", { name: /New Room/ });

        // when
        await user.click(create);

        // then
        expect(screen.getByTestId("create-room")).toBeInTheDocument();
    });
});

describe("RoomsListPage empty states", () => {
    it("says the member has hosted nothing yet", async () => {
        // given
        stubLists({ hosted: [] });

        // when
        renderList();

        // then
        expect(await screen.findByText("You haven't created any rooms yet.")).toBeInTheDocument();
    });

    it("says the member has joined nothing yet", async () => {
        // given
        stubLists({ joined: [], system: [] });

        // when
        renderList();

        // then
        expect(
            await screen.findByText("You haven't joined any rooms yet. Browse below or create one."),
        ).toBeInTheDocument();
    });

    it("invites the first public room when there are none", async () => {
        // given
        stubLists({ discover: [] });

        // when
        renderList();

        // then
        expect(await screen.findByText("No public rooms yet. Create the first one!")).toBeInTheDocument();
    });

    it("blames the filters for an empty result once one is set", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [] });
        renderList();
        await screen.findByText("No public rooms yet. Create the first one!");

        // when
        await user.click(screen.getByRole("button", { name: "RP only" }));

        // then
        expect(await screen.findByText("No public rooms match your search.")).toBeInTheDocument();
        expect(screen.getByText("No rooms you host match the filters.")).toBeInTheDocument();
        expect(screen.getByText("No joined rooms match the filters.")).toBeInTheDocument();
    });
});

describe("RoomsListPage room cards", () => {
    it("links a hosted room to its own page and counts its members", async () => {
        // given
        stubLists({ hosted: [makeRoom({ id: "room-9", name: "Rokkenjima", member_count: 6 })] });

        // when
        renderList();

        // then
        const link = await screen.findByRole("link", { name: /Rokkenjima/ });
        expect(link).toHaveAttribute("href", "/rooms/room-9");
        expect(screen.getByText(/6 members/)).toBeInTheDocument();
    });

    it("badges the room the member hosts", async () => {
        // given
        stubLists({ hosted: [makeRoom({ viewer_role: "host" })] });

        // when
        renderList();

        // then
        expect(await screen.findByText("Host")).toBeInTheDocument();
    });

    it("badges a roleplay room, a private room and an archived one", async () => {
        // given
        stubLists({
            hosted: [makeRoom({ is_rp: true, is_public: false, archived_at: "2026-01-02T00:00:00Z" })],
        });

        // when
        renderList();

        // then
        expect(await screen.findByText("RP")).toBeInTheDocument();
        expect(screen.getByText("Private")).toBeInTheDocument();
        expect(screen.getByText("Archived")).toBeInTheDocument();
    });

    it("marks a busy room as hot", async () => {
        // given
        stubLists({ hosted: [makeRoom({ hot_score: 51 })] });

        // when
        renderList();

        // then
        expect(await screen.findByText("Hot")).toBeInTheDocument();
    });

    it("never calls an archived room hot", async () => {
        // given
        stubLists({ hosted: [makeRoom({ hot_score: 99, archived_at: "2026-01-02T00:00:00Z" })] });

        // when
        renderList();

        // then
        expect(await screen.findByText("Archived")).toBeInTheDocument();
        expect(screen.queryByText("Hot")).not.toBeInTheDocument();
    });

    it("shows how many people are in voice", async () => {
        // given
        stubLists({ hosted: [makeRoom({ voice_count: 3 })] });

        // when
        renderList();

        // then
        expect(await screen.findByTitle("Voice chat active")).toHaveTextContent("3");
    });

    it("marks a room the member joined silently or muted", async () => {
        // given
        stubLists({ joined: [makeRoom({ viewer_ghost: true, viewer_muted: true })] });

        // when
        renderList();

        // then
        expect(await screen.findByTitle("You joined silently as a ghost")).toBeInTheDocument();
        expect(screen.getByTitle("Notifications muted")).toBeInTheDocument();
    });

    it("pins the system rooms above the joined ones", async () => {
        // given
        stubLists({ system: [makeRoom({ id: "room-sys", name: "Staff Lounge", is_system: true })] });

        // when
        renderList();

        // then
        expect(await screen.findByRole("link", { name: /Staff Lounge/ })).toHaveAttribute("href", "/rooms/room-sys");
        expect(screen.getByText("Pinned")).toBeInTheDocument();
        expect(screen.getByText("System")).toBeInTheDocument();
    });

    it("keeps system rooms out of the hosted and joined lists", async () => {
        // given
        stubLists({
            hosted: [makeRoom({ id: "room-sys", name: "Staff Lounge", is_system: true })],
            system: [],
        });

        // when
        renderList();

        // then
        expect(await screen.findByText("You haven't created any rooms yet.")).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: /Staff Lounge/ })).not.toBeInTheDocument();
    });

    it("separates roleplay rooms from ordinary chat rooms", async () => {
        // given
        stubLists({
            hosted: [makeRoom({ id: "r1", name: "In Character", is_rp: true }), makeRoom({ id: "r2", name: "OOC" })],
        });

        // when
        renderList();

        // then
        expect(await screen.findByText("Roleplay")).toBeInTheDocument();
        expect(screen.getByText("Chat")).toBeInTheDocument();
    });

    it("leaves the group labels off when every room is the same kind", async () => {
        // given
        stubLists({ hosted: [makeRoom({ id: "r1" }), makeRoom({ id: "r2", name: "Second" })] });

        // when
        renderList();

        // then
        await screen.findByRole("link", { name: /Second/ });
        expect(screen.queryByText("Roleplay")).not.toBeInTheDocument();
    });
});

describe("RoomsListPage discovery", () => {
    it("counts the public rooms in the section heading", async () => {
        // given
        stubLists({ discover: [makeRoom({ id: "pub-1" })], discoverTotal: 12 });

        // when
        renderList();

        // then
        expect(await screen.findByRole("heading", { name: "Discover Public Rooms (12)" })).toBeInTheDocument();
    });

    it("hides the join buttons from a signed out visitor", async () => {
        // given
        stubLists({ discover: [makeRoom({ id: "pub-1" })] });

        // when
        renderList({ user: null });

        // then
        await screen.findByText("Tea Parlour");
        expect(screen.queryByRole("button", { name: "Join Room" })).not.toBeInTheDocument();
    });

    it("joins a public room when the member asks to", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [makeRoom({ id: "pub-1" })] });
        renderList();
        const join = await screen.findByRole("button", { name: "Join Room" });

        // when
        await user.click(join);

        // then
        expect(mocks.joinMutateAsync).toHaveBeenCalledWith({ roomId: "pub-1", ghost: false });
    });

    it("keeps the ghost join for site staff only", async () => {
        // given
        stubLists({ discover: [makeRoom({ id: "pub-1" })] });

        // when
        renderList({ user: makeUser({ id: "viewer-1", role: undefined }) });

        // then
        await screen.findByRole("button", { name: "Join Room" });
        expect(screen.queryByRole("button", { name: /Ghost/ })).not.toBeInTheDocument();
    });

    it("lets site staff slip into a room silently", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [makeRoom({ id: "pub-1" })] });
        renderList({ user: makeUser({ id: "viewer-1", role: "moderator" }) });
        const ghost = await screen.findByRole("button", { name: /Ghost/ });

        // when
        await user.click(ghost);

        // then
        expect(mocks.joinMutateAsync).toHaveBeenCalledWith({ roomId: "pub-1", ghost: true });
    });

    it("explains why a join was refused and keeps the page steady", async () => {
        // given
        const user = userEvent.setup();
        mocks.joinMutateAsync.mockRejectedValue(new Error("banned from this room"));
        stubLists({ discover: [makeRoom({ id: "pub-1" })] });
        renderList();
        const join = await screen.findByRole("button", { name: "Join Room" });

        // when
        await user.click(join);

        // then
        expect(await screen.findByText("banned from this room")).toBeInTheDocument();
        await waitFor(() => {
            expect(screen.getByRole("button", { name: "Join Room" })).toBeEnabled();
        });
    });

    it("always calls a discovered room public", async () => {
        // given
        stubLists({ discover: [makeRoom({ id: "pub-1", is_public: false })] });

        // when
        renderList({ user: null });

        // then
        expect(await screen.findByText("Public")).toBeInTheDocument();
        expect(screen.queryByText("Private")).not.toBeInTheDocument();
    });
});

describe("RoomsListPage filters", () => {
    it("filters by a tag when one is clicked on a discovered room", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [makeRoom({ id: "pub-1", tags: ["horror"] })] });
        renderList({ user: null });
        const tag = await screen.findByRole("button", { name: "#horror" });

        // when
        await user.click(tag);

        // then
        expect(await screen.findByRole("button", { name: "#horror x" })).toBeInTheDocument();
    });

    it("clears the tag filter when the chip is clicked", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [makeRoom({ id: "pub-1", tags: ["horror"] })] });
        renderList({ user: null });
        await user.click(await screen.findByRole("button", { name: "#horror" }));
        const chip = await screen.findByRole("button", { name: "#horror x" });

        // when
        await user.click(chip);

        // then
        await waitFor(() => {
            expect(screen.queryByRole("button", { name: "#horror x" })).not.toBeInTheDocument();
        });
    });

    it("asks the server for archived rooms once that filter is on", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [] });
        renderList({ user: null });
        await screen.findByText("No public rooms yet. Create the first one!");

        // when
        await user.click(screen.getByRole("button", { name: "Include archived" }));

        // then
        await waitFor(() => {
            expect(mocks.listPublicChatRooms).toHaveBeenLastCalledWith(
                expect.objectContaining({ includeArchived: true }),
            );
        });
    });

    it("asks the server for roleplay rooms only once that filter is on", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [] });
        renderList({ user: null });
        await screen.findByText("No public rooms yet. Create the first one!");

        // when
        await user.click(screen.getByRole("button", { name: "RP only" }));

        // then
        await waitFor(() => {
            expect(mocks.listPublicChatRooms).toHaveBeenLastCalledWith(expect.objectContaining({ rp: true }));
        });
    });

    it("waits for the typing to settle before searching", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [] });
        renderList({ user: null });
        await screen.findByText("No public rooms yet. Create the first one!");

        // when
        await user.type(screen.getByPlaceholderText("Search rooms..."), "tea");

        // then
        await waitFor(() => {
            expect(mocks.listPublicChatRooms).toHaveBeenLastCalledWith(expect.objectContaining({ search: "tea" }));
        });
    });
});

describe("RoomsListPage paging", () => {
    it("offers to load more public rooms when there are more to come", async () => {
        // given
        stubLists({ discover: [makeRoom({ id: "pub-1" })], discoverTotal: 40 });

        // when
        renderList({ user: null });

        // then
        expect(await screen.findByRole("button", { name: "Load more" })).toBeInTheDocument();
    });

    it("asks for a bigger page when load more is clicked", async () => {
        // given
        const user = userEvent.setup();
        stubLists({ discover: [makeRoom({ id: "pub-1" })], discoverTotal: 40 });
        renderList({ user: null });
        const loadMore = await screen.findByRole("button", { name: "Load more" });

        // when
        await user.click(loadMore);

        // then
        await waitFor(() => {
            expect(mocks.listPublicChatRooms).toHaveBeenLastCalledWith(expect.objectContaining({ limit: 40 }));
        });
    });

    it("hides load more once everything has arrived", async () => {
        // given
        stubLists({ discover: [makeRoom({ id: "pub-1" })], discoverTotal: 1 });

        // when
        renderList({ user: null });

        // then
        await screen.findByText("Tea Parlour");
        expect(screen.queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
    });
});

describe("RoomsListPage live updates", () => {
    it("refreshes the joined and pinned rooms on an invitation", async () => {
        // given
        const { listeners, queryClient } = renderList();
        await screen.findByRole("heading", { name: /My Rooms/ });
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "chat_room_invited", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["chat", "rooms-list", "joined"] });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.userRooms() });
    });

    it("refreshes every list when the member is kicked", async () => {
        // given
        const { listeners, queryClient } = renderList();
        await screen.findByRole("heading", { name: /My Rooms/ });
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "chat_kicked", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["chat", "rooms-list"] });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.userRooms() });
    });

    it("reads the pinned rooms from the shared user rooms entry", async () => {
        // given
        stubLists({ system: [makeRoom({ id: "room-sys", name: "Staff Lounge", is_system: true })] });

        // when
        const { queryClient } = renderList();

        // then
        await screen.findByRole("link", { name: /Staff Lounge/ });
        expect(queryClient.getQueryData(queryKeys.chat.userRooms())).toBeDefined();
    });

    it("refreshes the lists when voice presence changes", async () => {
        // given
        const { listeners, queryClient } = renderList();
        await screen.findByRole("heading", { name: /My Rooms/ });
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "voice_presence", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["chat", "rooms-list"] });
    });

    it("ignores chat events it does not care about", async () => {
        // given
        const { listeners, queryClient } = renderList();
        await screen.findByRole("heading", { name: /My Rooms/ });
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "chat_message", data: {} });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});
