import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DmController } from "../../hooks/useDmController";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ChatRoom, User, UserProfile } from "../../types/api";
import { ChatPage } from "./ChatPage";

const mocks = vi.hoisted(() => ({
    useDmController: vi.fn(),
    useIsMobile: vi.fn(),
    forceMuteVoiceParticipant: vi.fn(),
}));

vi.mock("../../hooks/useDmController", async importOriginal => {
    const actual = await importOriginal<typeof import("../../hooks/useDmController")>();
    return { ...actual, useDmController: mocks.useDmController };
});

vi.mock("../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

vi.mock("../../api/endpoints", () => ({ forceMuteVoiceParticipant: mocks.forceMuteVoiceParticipant }));

vi.mock("../../components/chat/mobile/MobileDmView", () => ({
    MobileDmView: () => <div data-testid="mobile-dm-view" />,
}));

vi.mock("../../components/chat/ChatComposer/ChatComposer", () => ({
    ChatComposer: (props: {
        roomId: string | null;
        draftRecipientId: string | null;
        extraActions?: React.ReactNode;
    }) => (
        <div data-testid="composer" data-room={String(props.roomId)} data-draft={String(props.draftRecipientId)}>
            {props.extraActions}
        </div>
    ),
}));

vi.mock("../../components/chat/MessageList/DmMessageList", () => ({
    DmMessageList: () => <div data-testid="dm-messages" />,
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

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });
const beatrice: User = { id: "user-b", username: "beatrice", display_name: "Beatrice" };
const ange: User = { id: "user-a", username: "ange", display_name: "Ange" };

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "",
        description: "",
        type: "dm",
        is_public: false,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [viewer, beatrice],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

interface ControllerOptions {
    user?: UserProfile | null;
    loading?: boolean;
    rooms?: ChatRoom[];
    activeRoom?: ChatRoom | null;
    activeRoomId?: string | null;
    draftRecipient?: User | null;
    typingNames?: string[];
    voiceStatus?: string;
    voiceRoom?: object | null;
    voiceEnabled?: boolean;
    lightboxSrc?: string | null;
    showNewDm?: boolean;
    dmSearch?: string;
    dmResults?: User[];
    dmMutuals?: User[];
    dmError?: string;
    dmCreating?: boolean;
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
        voiceLeave: vi.fn(),
        voiceJoin: vi.fn(),
    };

    const controller = {
        user: options.user === undefined ? viewer : options.user,
        loading: options.loading ?? false,
        mobileView: "list",
        rooms: options.rooms ?? [],
        activeRoomId: options.activeRoomId ?? options.activeRoom?.id ?? null,
        activeRoom: options.activeRoom ?? null,
        draftRecipient: options.draftRecipient ?? null,
        setDraftRecipient: handlers.setDraftRecipient,
        messagesEndRef: { current: null },
        typingNames: options.typingNames ?? [],
        voice: {
            status: options.voiceStatus ?? "idle",
            room: options.voiceRoom ?? null,
            join: handlers.voiceJoin,
            leave: handlers.voiceLeave,
            presenceCount: 0,
        },
        voiceEnabled: options.voiceEnabled ?? true,
        replyingTo: null,
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
        handleRoomSelect: handlers.handleRoomSelect,
        handleMobileBack: handlers.handleMobileBack,
        handleSentMessage: handlers.handleSentMessage,
        handleSelectUser: handlers.handleSelectUser,
        handleEditLast: handlers.handleEditLast,
        handleDeleteChat: handlers.handleDeleteChat,
        notifyTyping: handlers.notifyTyping,
    };

    mocks.useDmController.mockReturnValue(controller as unknown as DmController);

    return handlers;
}

function renderChat(options: ControllerOptions = {}) {
    const handlers = stubController(options);
    const result = renderWithProviders(<ChatPage />, { user: options.user === undefined ? viewer : options.user });

    return { ...result, ...handlers };
}

beforeEach(() => {
    mocks.useIsMobile.mockReturnValue(false);
    mocks.forceMuteVoiceParticipant.mockResolvedValue(undefined);
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
        expect(screen.getByText("Send your first message to Beatrice.")).toBeInTheDocument();
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
        expect(screen.getByText("Beatrice is typing...")).toBeInTheDocument();
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
        renderChat({ activeRoom: makeRoom(), voiceStatus, voiceRoom: {} });

        // then
        expect(screen.queryByRole("button", { name: "voice bar" })).not.toBeInTheDocument();
    });

    it("shows the voice bar once the call is connected", () => {
        // given
        const voiceStatus = "connected";

        // when
        renderChat({ activeRoom: makeRoom(), voiceStatus, voiceRoom: {} });

        // then
        expect(screen.getByRole("button", { name: "voice bar" })).toBeInTheDocument();
    });

    it("withholds moderation from an ordinary member", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: undefined });

        // when
        renderChat({ user, activeRoom: makeRoom(), voiceStatus: "connected", voiceRoom: {} });

        // then
        expect(screen.getByRole("button", { name: "voice bar" })).toHaveAttribute("data-moderator", "false");
    });

    it("grants moderation to site staff", () => {
        // given
        const user = makeUser({ id: "viewer-1", role: "moderator" });

        // when
        renderChat({ user, activeRoom: makeRoom(), voiceStatus: "connected", voiceRoom: {} });

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
            voiceRoom: {},
        });

        // when
        await user.click(screen.getByRole("button", { name: "voice bar" }));

        // then
        expect(mocks.forceMuteVoiceParticipant).toHaveBeenCalledWith("room-5", "u9", true);
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
