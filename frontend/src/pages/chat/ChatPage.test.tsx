import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Room } from "livekit-client";
import { roomCapabilities } from "../../domain/chat/roomPolicy";
import type { DmThreadView } from "../../components/chat/mobile/MobileDmView";
import type { DmController } from "../../hooks/useDmController";
import { makeDmController } from "../../hooks/useDmController.fixture";
import { makeChatMessage, makeDmRoom, makeRoomMember, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { emitRealtimeEvent } from "../../test-utils/ws";
import type { ChatMessage, ChatRoom, ChatRoomMember, User, UserProfile } from "../../types/api";
import { ChatPage } from "./ChatPage";

const mocks = vi.hoisted(() => ({
    useDmController: vi.fn(),
    useIsMobile: vi.fn(),
    forceMute: vi.fn(),
    useChatRoomMembers: vi.fn(),
}));

vi.mock("../../hooks/useDmController", async importOriginal => {
    const actual = await importOriginal<typeof import("../../hooks/useDmController")>();
    return { ...actual, useDmController: mocks.useDmController };
});

vi.mock("../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

vi.mock("../../hooks/mutations/chat", async importOriginal => {
    const actual = await importOriginal<typeof import("../../hooks/mutations/chat")>();
    return {
        ...actual,
        useForceMuteVoiceParticipant: (roomId: string | null | undefined) => ({
            mutate: (variables: { userId: string; muted: boolean }, options?: { onError?: (err: unknown) => void }) =>
                mocks.forceMute(roomId, variables, options),
        }),
    };
});

vi.mock("../../hooks/queries/chat", async importOriginal => {
    const actual = await importOriginal<typeof import("../../hooks/queries/chat")>();
    return { ...actual, useChatRoomMembers: mocks.useChatRoomMembers };
});

vi.mock("../../components/chat/mobile/MobileDmView", () => ({
    MobileDmView: (props: { controller: DmController; thread: DmThreadView }) => (
        <div data-testid="mobile-dm-view" data-highlighted={props.thread.anchor.highlightedMsgId ?? ""}>
            {props.controller.messages.map(message => (
                <div key={message.id} id={`chat-msg-${message.id}`} />
            ))}
        </div>
    ),
}));

vi.mock("../../components/chat/ChatComposer/ChatComposer", () => ({
    ChatComposer: (props: {
        roomId: string | null;
        draftRecipientId: string | null;
        mentionPool?: User[];
        timeoutUntil?: string;
        extraActions?: React.ReactNode;
    }) => (
        <div
            data-testid="composer"
            data-room={String(props.roomId)}
            data-draft={String(props.draftRecipientId)}
            data-mention-pool={
                props.mentionPool === undefined ? "none" : props.mentionPool.map(u => u.username).join(",")
            }
            data-timeout-until={props.timeoutUntil ?? ""}
        >
            {props.extraActions}
        </div>
    ),
}));

vi.mock("../../components/chat/MessageList/MessageList", () => ({
    MessageList: (props: {
        messages: ChatMessage[];
        highlightedMessageId?: string | null;
        viewerTimedOut?: boolean;
    }) => (
        <div
            data-testid="dm-messages"
            data-highlighted={props.highlightedMessageId ?? ""}
            data-timed-out={String(Boolean(props.viewerTimedOut))}
        >
            {props.messages.map(message => (
                <div key={message.id} id={`chat-msg-${message.id}`}>
                    {message.body}
                </div>
            ))}
        </div>
    ),
}));

vi.mock("../../components/chat/MessageSearchPanel/MessageSearchPanel", () => ({
    MessageSearchPanel: (props: {
        roomId: string;
        onClose: () => void;
        onJump: (messageId: string, createdAt?: string) => void;
    }) => (
        <div data-testid="search-panel" data-room={props.roomId}>
            <button type="button" onClick={() => props.onJump("m2", "2026-01-02T00:00:00Z")}>
                jump to a hit
            </button>
            <button type="button" onClick={props.onClose}>
                close search
            </button>
        </div>
    ),
}));

vi.mock("../../components/chat/Voice/VoiceBar", () => ({
    VoiceBar: (props: { canModerate: boolean; onForceMute: (id: string, muted: boolean) => void }) => (
        <button type="button" data-moderator={String(props.canModerate)} onClick={() => props.onForceMute("u9", true)}>
            voice bar
        </button>
    ),
}));

vi.mock("../../components/chat/Voice/VoiceButton", () => ({
    VoiceButton: (props: { enabled: boolean }) => <div data-testid="voice-button">{String(props.enabled)}</div>,
}));

vi.mock("../../components/Lightbox/Lightbox", () => ({
    Lightbox: (props: { src: string }) => <div data-testid="lightbox">{props.src}</div>,
}));

const scrollIntoView = vi.spyOn(Element.prototype, "scrollIntoView");

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });
const beatrice: User = { id: "user-b", username: "beatrice", display_name: "Beatrice" };
const ange: User = { id: "user-a", username: "ange", display_name: "Ange" };

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({ members: [viewer, beatrice], ...overrides });
}

interface ControllerOptions {
    user?: UserProfile | null;
    loading?: boolean;
    rooms?: ChatRoom[];
    activeRoom?: ChatRoom | null;
    activeRoomId?: string | null;
    draftRecipient?: User | null;
    messages?: ChatMessage[];
    roomMembers?: ChatRoomMember[];
    route?: string;
    typingNames?: string[];
    voiceStatus?: DmController["voice"]["status"];
    voiceRoom?: DmController["voice"]["room"];
    voiceEnabled?: boolean;
    lightboxSrc?: string | null;
    showNewDm?: boolean;
    dmSearch?: string;
    dmResults?: User[];
    dmMutuals?: User[];
    dmError?: string;
    dmCreating?: boolean;
    toast?: string | null;
}

function stubController(options: ControllerOptions = {}) {
    const handlers = {
        setDraftRecipient: vi.fn(),
        setReplyingTo: vi.fn(),
        setLightboxSrc: vi.fn(),
        setShowNewDm: vi.fn(),
        setDmSearch: vi.fn(),
        handleRoomSelect: vi.fn(),
        handleMobileBack: vi.fn(),
        handleSentMessage: vi.fn(),
        handleSelectUser: vi.fn(),
        handleEditLast: vi.fn(),
        handleDeleteChat: vi.fn(),
        notifyTyping: vi.fn(),
        showToast: vi.fn(),
        voiceLeave: vi.fn(),
        voiceJoin: vi.fn(),
        loadUntilMessage: vi.fn(() => Promise.resolve(true)),
    };

    const base = makeDmController();
    const viewerUser = options.user === undefined ? viewer : options.user;
    const activeRoom = options.activeRoom ?? null;

    mocks.useChatRoomMembers.mockReturnValue({
        members: options.roomMembers ?? [],
        loading: false,
        refresh: vi.fn(),
    });

    const controller = makeDmController({
        user: viewerUser,
        capabilities: roomCapabilities(activeRoom, viewerUser),
        loading: options.loading ?? false,
        rooms: options.rooms ?? [],
        activeRoomId: options.activeRoomId ?? activeRoom?.id ?? null,
        activeRoom: activeRoom ?? undefined,
        messages: options.messages ?? [],
        loadUntilMessage: handlers.loadUntilMessage,
        draftRecipient: options.draftRecipient ?? null,
        setDraftRecipient: handlers.setDraftRecipient,
        typingNames: options.typingNames ?? [],
        voice: {
            ...base.voice,
            status: options.voiceStatus ?? "idle",
            room: options.voiceRoom ?? null,
            join: handlers.voiceJoin,
            leave: handlers.voiceLeave,
        },
        voiceEnabled: options.voiceEnabled ?? true,
        setReplyingTo: handlers.setReplyingTo,
        lightboxSrc: options.lightboxSrc ?? null,
        setLightboxSrc: handlers.setLightboxSrc,
        showNewDm: options.showNewDm ?? false,
        setShowNewDm: handlers.setShowNewDm,
        dmSearch: options.dmSearch ?? "",
        setDmSearch: handlers.setDmSearch,
        dmResults: options.dmResults ?? [],
        dmMutuals: options.dmMutuals ?? [],
        dmError: options.dmError ?? "",
        dmCreating: options.dmCreating ?? false,
        toast: options.toast ?? null,
        showToast: handlers.showToast,
        handleRoomSelect: handlers.handleRoomSelect,
        handleMobileBack: handlers.handleMobileBack,
        handleSentMessage: handlers.handleSentMessage,
        handleSelectUser: handlers.handleSelectUser,
        handleEditLast: handlers.handleEditLast,
        handleDeleteChat: handlers.handleDeleteChat,
        notifyTyping: handlers.notifyTyping,
    });

    mocks.useDmController.mockReturnValue(controller);

    return handlers;
}

function renderChat(options: ControllerOptions = {}) {
    const handlers = stubController(options);
    const result = renderWithProviders(<ChatPage />, {
        user: options.user === undefined ? viewer : options.user,
        route: options.route ?? "/chat",
    });

    return { ...result, ...handlers };
}

beforeEach(() => {
    mocks.useIsMobile.mockReturnValue(false);
    mocks.forceMute.mockReset();
    mocks.useChatRoomMembers.mockReturnValue({ members: [], loading: false, refresh: vi.fn() });
});

describe("ChatPage gates", () => {
    it("shows nothing at all to a signed out visitor", () => {
        // given
        const user = null;

        // when
        const { container } = renderChat({ user });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("waits while the conversations are being loaded", () => {
        // given
        const loading = true;

        // when
        renderChat({ loading });

        // then
        expect(screen.getByText("Loading chat...")).toBeInTheDocument();
        expect(screen.queryByText("Messages")).not.toBeInTheDocument();
    });

    it("hands the page to the mobile view on a small screen", () => {
        // given
        mocks.useIsMobile.mockReturnValue(true);

        // when
        renderChat();

        // then
        expect(screen.getByTestId("mobile-dm-view")).toBeInTheDocument();
        expect(screen.queryByText("Messages")).not.toBeInTheDocument();
    });
});

describe("ChatPage conversation list", () => {
    it("says there are no conversations yet when the list is empty", () => {
        // given
        const rooms: ChatRoom[] = [];

        // when
        renderChat({ rooms });

        // then
        expect(screen.getByText("No conversations yet")).toBeInTheDocument();
    });

    it("shows the other person's name for a direct message", () => {
        // given
        const rooms = [makeRoom()];

        // when
        renderChat({ rooms });

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("names a group chat after the room itself", () => {
        // given
        const rooms = [makeRoom({ id: "room-g", type: "group", name: "Rokkenjima" })];

        // when
        renderChat({ rooms });

        // then
        expect(screen.getByText("Rokkenjima")).toBeInTheDocument();
    });

    it("marks a conversation with unread messages", () => {
        // given
        const rooms = [makeRoom({ unread: true })];

        // when
        renderChat({ rooms });

        // then
        expect(screen.getByLabelText("unread")).toBeInTheDocument();
    });

    it("opens a conversation when it is picked", async () => {
        // given
        const user = userEvent.setup();
        const { handleRoomSelect } = renderChat({ rooms: [makeRoom({ id: "room-7" })] });

        // when
        await user.click(screen.getByText("Beatrice"));

        // then
        expect(handleRoomSelect).toHaveBeenCalledWith("room-7");
    });
});

describe("ChatPage message area", () => {
    it("asks the viewer to pick a conversation when none is open", () => {
        // given
        const activeRoom = null;

        // when
        renderChat({ activeRoom });

        // then
        expect(screen.getByText("Select a conversation")).toBeInTheDocument();
        expect(screen.queryByTestId("dm-messages")).not.toBeInTheDocument();
    });

    it("opens a blank thread for a brand new conversation", () => {
        // given
        const draftRecipient = beatrice;

        // when
        renderChat({ draftRecipient });

        // then
        expect(screen.getByText(/Send your first message to/)).toHaveTextContent(
            "Send your first message to Beatrice.",
        );
        expect(screen.getByTestId("composer")).toHaveAttribute("data-draft", "user-b");
    });

    it("abandons a brand new conversation when cancelled", async () => {
        // given
        const user = userEvent.setup();
        const { setDraftRecipient } = renderChat({ draftRecipient: beatrice });

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(setDraftRecipient).toHaveBeenCalledWith(null);
    });

    it("shows the messages of the open conversation", () => {
        // given
        const activeRoom = makeRoom({ id: "room-3" });

        // when
        renderChat({ activeRoom });

        // then
        expect(screen.getByTestId("dm-messages")).toBeInTheDocument();
        expect(screen.getByTestId("composer")).toHaveAttribute("data-room", "room-3");
    });

    it("offers to delete the open conversation", async () => {
        // given
        const user = userEvent.setup();
        const { handleDeleteChat } = renderChat({ activeRoom: makeRoom() });

        // when
        await user.click(screen.getByRole("button", { name: "Delete Chat" }));

        // then
        expect(handleDeleteChat).toHaveBeenCalledTimes(1);
    });

    it("says who is typing in the open conversation", () => {
        // given
        const typingNames = ["Beatrice"];

        // when
        renderChat({ activeRoom: makeRoom(), typingNames });

        // then
        expect(screen.getByText(/is typing/)).toHaveTextContent("Beatrice is typing...");
    });

    it("goes back to the conversation list when asked", async () => {
        // given
        const user = userEvent.setup();
        const { handleMobileBack } = renderChat({ activeRoom: makeRoom() });

        // when
        await user.click(screen.getByRole("button", { name: "Back to conversations" }));

        // then
        expect(handleMobileBack).toHaveBeenCalledTimes(1);
    });

    it("passes the voice availability through to the composer", () => {
        // given
        const voiceEnabled = false;

        // when
        renderChat({ activeRoom: makeRoom(), voiceEnabled });

        // then
        expect(screen.getByTestId("voice-button")).toHaveTextContent("false");
    });
});

describe("ChatPage voice", () => {
    it("keeps the voice bar away until the call is connected", () => {
        // given
        const voiceStatus = "connecting";

        // when
        renderChat({ activeRoom: makeRoom(), voiceStatus, voiceRoom: new Room() });

        // then
        expect(screen.queryByRole("button", { name: "voice bar" })).not.toBeInTheDocument();
    });

    it("shows the voice bar once the call is connected", () => {
        // given
        const voiceStatus = "connected";

        // when
        renderChat({ activeRoom: makeRoom(), voiceStatus, voiceRoom: new Room() });

        // then
        expect(screen.getByRole("button", { name: "voice bar" })).toBeInTheDocument();
    });

    it("withholds moderation from an ordinary member", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: undefined });

        // when
        renderChat({ user, activeRoom: makeRoom(), voiceStatus: "connected", voiceRoom: new Room() });

        // then
        expect(screen.getByRole("button", { name: "voice bar" })).toHaveAttribute("data-moderator", "false");
    });

    it("grants moderation to site staff", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: "moderator" });

        // when
        renderChat({ user, activeRoom: makeRoom(), voiceStatus: "connected", voiceRoom: new Room() });

        // then
        expect(screen.getByRole("button", { name: "voice bar" })).toHaveAttribute("data-moderator", "true");
    });

    it("force mutes a participant in the open room", async () => {
        // given
        const user = userEvent.setup();
        renderChat({
            user: makeUser({ id: "viewer-1", role: "moderator" }),
            activeRoom: makeRoom({ id: "room-5" }),
            activeRoomId: "room-5",
            voiceStatus: "connected",
            voiceRoom: new Room(),
        });

        // when
        await user.click(screen.getByRole("button", { name: "voice bar" }));

        // then
        expect(mocks.forceMute).toHaveBeenCalledWith("room-5", { userId: "u9", muted: true }, expect.anything());
    });

    it("tells the moderator when a server mute did not take", async () => {
        // given
        mocks.forceMute.mockImplementation(
            (
                _roomId: string | null | undefined,
                _variables: { userId: string; muted: boolean },
                options?: { onError?: (err: unknown) => void },
            ) => options?.onError?.(new Error("LiveKit said no")),
        );
        const user = userEvent.setup();
        const { showToast } = renderChat({
            user: makeUser({ id: "viewer-1", role: "moderator" }),
            activeRoom: makeRoom({ id: "room-5" }),
            activeRoomId: "room-5",
            voiceStatus: "connected",
            voiceRoom: new Room(),
        });

        // when
        await user.click(screen.getByRole("button", { name: "voice bar" }));

        // then
        await waitFor(() => {
            expect(showToast).toHaveBeenCalledWith("LiveKit said no");
        });
    });
});

describe("ChatPage new direct message", () => {
    it("keeps the new message dialog shut until it is asked for", () => {
        // given
        const showNewDm = false;

        // when
        renderChat({ showNewDm });

        // then
        expect(screen.queryByText("New Direct Message")).not.toBeInTheDocument();
    });

    it("opens the new message dialog when asked", async () => {
        // given
        const user = userEvent.setup();
        const { setShowNewDm } = renderChat();

        // when
        await user.click(screen.getByRole("button", { name: "New DM" }));

        // then
        expect(setShowNewDm).toHaveBeenCalledWith(true);
    });

    it("suggests mutual followers before anything has been searched", () => {
        // given
        const dmMutuals = [beatrice, ange];

        // when
        renderChat({ showNewDm: true, dmMutuals });

        // then
        expect(screen.getByText("Mutual followers")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("Ange")).toBeInTheDocument();
    });

    it("asks the viewer to search when they have no mutual followers", () => {
        // given
        const dmMutuals: User[] = [];

        // when
        renderChat({ showNewDm: true, dmMutuals });

        // then
        expect(screen.getByText("Search for a user to start a conversation")).toBeInTheDocument();
        expect(screen.queryByText("Mutual followers")).not.toBeInTheDocument();
    });

    it("says nobody matched a search that found nothing", () => {
        // given
        const dmSearch = "kinzo";

        // when
        renderChat({ showNewDm: true, dmSearch, dmResults: [], dmMutuals: [beatrice] });

        // then
        expect(screen.getByText("No users found")).toBeInTheDocument();
        expect(screen.queryByText("Mutual followers")).not.toBeInTheDocument();
    });

    it("starts a conversation with a searched user", async () => {
        // given
        const user = userEvent.setup();
        const { handleSelectUser } = renderChat({ showNewDm: true, dmSearch: "bea", dmResults: [beatrice] });

        // when
        await user.click(screen.getByText("Beatrice"));

        // then
        expect(handleSelectUser).toHaveBeenCalledWith(beatrice);
    });

    it("stops the viewer picking twice while a conversation is being created", () => {
        // given
        const dmCreating = true;

        // when
        renderChat({ showNewDm: true, dmSearch: "bea", dmResults: [beatrice], dmCreating });

        // then
        expect(screen.getByText("Beatrice").closest("button")).toBeDisabled();
    });

    it("reports why a conversation could not be started", () => {
        // given
        const dmError = "That witch has closed her letterbox.";

        // when
        renderChat({ showNewDm: true, dmError });

        // then
        expect(screen.getByText("That witch has closed her letterbox.")).toBeInTheDocument();
    });
});

describe("ChatPage message anchoring", () => {
    it("follows a notification deep link to the message it names", async () => {
        // given
        const messages = [makeChatMessage({ id: "m1", body: "first" }), makeChatMessage({ id: "m2", body: "second" })];

        // when
        renderChat({ activeRoom: makeRoom({ id: "room-1" }), messages, route: "/chat/room-1#msg-m2" });

        // then
        await waitFor(() => {
            expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-highlighted", "m2");
        });
        expect(scrollIntoView).toHaveBeenCalled();
    });

    it("reaches back through older history for a deep link that is not on screen", async () => {
        // given
        const messages = [makeChatMessage({ id: "m1" })];

        // when
        const { loadUntilMessage } = renderChat({
            activeRoom: makeRoom({ id: "room-1" }),
            messages,
            route: "/chat/room-1#msg-m9",
        });

        // then
        await waitFor(() => {
            expect(loadUntilMessage).toHaveBeenCalledWith("m9", undefined);
        });
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-highlighted", "");
    });

    it("carries a deep link through to the mobile conversation", async () => {
        // given
        mocks.useIsMobile.mockReturnValue(true);
        const messages = [makeChatMessage({ id: "m2" })];

        // when
        renderChat({ activeRoom: makeRoom({ id: "room-1" }), messages, route: "/chat/room-1#msg-m2" });

        // then
        await waitFor(() => {
            expect(screen.getByTestId("mobile-dm-view")).toHaveAttribute("data-highlighted", "m2");
        });
    });

    it("leaves the conversation unanchored when the address names no message", () => {
        // given
        const messages = [makeChatMessage({ id: "m1" })];

        // when
        const { loadUntilMessage } = renderChat({ activeRoom: makeRoom({ id: "room-1" }), messages });

        // then
        expect(loadUntilMessage).not.toHaveBeenCalled();
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-highlighted", "");
    });
});

describe("ChatPage message search", () => {
    it("keeps the search panel shut until the viewer asks for it", () => {
        // given
        const activeRoom = makeRoom();

        // when
        renderChat({ activeRoom });

        // then
        expect(screen.queryByTestId("search-panel")).not.toBeInTheDocument();
    });

    it("searches the conversation that is open", async () => {
        // given
        const user = userEvent.setup();
        renderChat({ activeRoom: makeRoom({ id: "room-4" }) });

        // when
        await user.click(screen.getByRole("button", { name: "Search messages" }));

        // then
        expect(screen.getByTestId("search-panel")).toHaveAttribute("data-room", "room-4");
    });

    it("puts the search away again", async () => {
        // given
        const user = userEvent.setup();
        renderChat({ activeRoom: makeRoom() });
        await user.click(screen.getByRole("button", { name: "Search messages" }));

        // when
        await user.click(screen.getByRole("button", { name: "close search" }));

        // then
        expect(screen.queryByTestId("search-panel")).not.toBeInTheDocument();
    });

    it("scrolls to the message the viewer picked out of the results", async () => {
        // given
        const user = userEvent.setup();
        const messages = [makeChatMessage({ id: "m1" }), makeChatMessage({ id: "m2" })];
        renderChat({ activeRoom: makeRoom(), messages });
        await user.click(screen.getByRole("button", { name: "Search messages" }));

        // when
        await user.click(screen.getByRole("button", { name: "jump to a hit" }));

        // then
        await waitFor(() => {
            expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-highlighted", "m2");
        });
    });

    it("offers no search from a conversation that does not exist yet", () => {
        // given
        const draftRecipient = beatrice;

        // when
        renderChat({ draftRecipient });

        // then
        expect(screen.queryByRole("button", { name: "Search messages" })).not.toBeInTheDocument();
    });
});

describe("ChatPage composer", () => {
    it("offers the conversation's own members to the mention autocomplete", () => {
        // given
        const activeRoom = makeRoom();

        // when
        renderChat({ activeRoom });

        // then
        expect(screen.getByTestId("composer")).toHaveAttribute("data-mention-pool", "battler,beatrice");
    });

    it("leaves a brand new conversation on the site-wide mention search", () => {
        // given
        const draftRecipient = beatrice;

        // when
        renderChat({ draftRecipient });

        // then
        expect(screen.getByTestId("composer")).toHaveAttribute("data-mention-pool", "none");
    });

    it("hands the composer the viewer's own timeout and locks the messages", () => {
        // given
        const roomMembers = [makeRoomMember({ user: viewer, timeout_until: "2099-01-01T00:00:00Z" })];

        // when
        renderChat({ activeRoom: makeRoom(), roomMembers });

        // then
        expect(screen.getByTestId("composer")).toHaveAttribute("data-timeout-until", "2099-01-01T00:00:00Z");
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-timed-out", "true");
    });

    it("ignores a timeout that belongs to the other person", () => {
        // given
        const roomMembers = [makeRoomMember({ user: beatrice, timeout_until: "2099-01-01T00:00:00Z" })];

        // when
        renderChat({ activeRoom: makeRoom(), roomMembers });

        // then
        expect(screen.getByTestId("composer")).toHaveAttribute("data-timeout-until", "");
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-timed-out", "false");
    });

    it("takes up a timeout imposed while the conversation is open", async () => {
        // given
        const roomMembers = [makeRoomMember({ user: viewer })];
        renderChat({ activeRoom: makeRoom({ id: "room-1" }), roomMembers });
        expect(screen.getByTestId("composer")).toHaveAttribute("data-timeout-until", "");

        // when
        emitRealtimeEvent({
            type: "chat_member_updated",
            data: { room_id: "room-1", user_id: viewer.id, timeout_until: "2099-01-01T00:00:00Z" },
        });

        // then
        await waitFor(() => {
            expect(screen.getByTestId("composer")).toHaveAttribute("data-timeout-until", "2099-01-01T00:00:00Z");
        });
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-timed-out", "true");
    });

    it("ignores a timeout imposed in some other conversation", async () => {
        // given
        const roomMembers = [makeRoomMember({ user: viewer })];
        renderChat({ activeRoom: makeRoom({ id: "room-1" }), roomMembers });

        // when
        emitRealtimeEvent({
            type: "chat_member_updated",
            data: { room_id: "room-2", user_id: viewer.id, timeout_until: "2099-01-01T00:00:00Z" },
        });

        // then
        await waitFor(() => {
            expect(screen.getByTestId("composer")).toHaveAttribute("data-timeout-until", "");
        });
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-timed-out", "false");
    });

    it("unlocks the messages once the viewer's timeout has run out", () => {
        // given
        const roomMembers = [makeRoomMember({ user: viewer, timeout_until: "2020-01-01T00:00:00Z" })];

        // when
        renderChat({ activeRoom: makeRoom(), roomMembers });

        // then
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-timed-out", "false");
    });
});

describe("ChatPage lightbox", () => {
    it("keeps the lightbox shut until an image is opened", () => {
        // given
        const lightboxSrc = null;

        // when
        renderChat({ lightboxSrc });

        // then
        expect(screen.queryByTestId("lightbox")).not.toBeInTheDocument();
    });

    it("shows the image the viewer opened", () => {
        // given
        const lightboxSrc = "/media/witch.png";

        // when
        renderChat({ lightboxSrc });

        // then
        expect(screen.getByTestId("lightbox")).toHaveTextContent("/media/witch.png");
    });
});
