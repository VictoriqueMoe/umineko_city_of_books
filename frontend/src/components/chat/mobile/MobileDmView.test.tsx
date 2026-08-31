import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Room } from "livekit-client";
import { roomCapabilities } from "../../../domain/chat/roomPolicy";
import type { DmController } from "../../../hooks/useDmController";
import { makeDmController } from "../../../hooks/useDmController.fixture";
import { makeDmRoom, makePublicUser, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatRoom, User } from "../../../types/api";
import { MobileDmView, type DmThreadView } from "./MobileDmView";

const mocks = vi.hoisted(() => ({
    forceMuteVoiceParticipant: vi.fn<
        (roomId: string | null | undefined, userId: string, muted: boolean) => Promise<void>
    >(() => Promise.resolve()),
}));

vi.mock("../../../hooks/mutations/chat", async importOriginal => {
    const actual = await importOriginal<typeof import("../../../hooks/mutations/chat")>();

    return {
        ...actual,
        useForceMuteVoiceParticipant: (roomId: string | null | undefined) => ({
            mutate: (
                variables: { userId: string; muted: boolean },
                callbacks?: { onError?: (err: unknown) => void },
            ) => {
                mocks
                    .forceMuteVoiceParticipant(roomId, variables.userId, variables.muted)
                    .catch((err: unknown) => callbacks?.onError?.(err));
            },
        }),
    };
});

vi.mock("../MessageList/MessageList", () => ({
    MessageList: ({
        highlightedMessageId,
        viewerTimedOut,
    }: {
        highlightedMessageId?: string | null;
        viewerTimedOut?: boolean;
    }) => (
        <div
            data-testid="dm-messages"
            data-highlighted={highlightedMessageId ?? ""}
            data-timed-out={String(Boolean(viewerTimedOut))}
        >
            messages
        </div>
    ),
}));

vi.mock("../MessageSearchPanel/MessageSearchPanel", () => ({
    MessageSearchPanel: ({ roomId, onClose }: { roomId: string; onClose: () => void }) => (
        <div data-testid="search-panel" data-room={roomId}>
            <button type="button" onClick={onClose}>
                close search
            </button>
        </div>
    ),
}));

vi.mock("../Voice/VoiceBar", () => ({
    VoiceBar: ({
        onLeave,
        canModerate,
        onForceMute,
    }: {
        onLeave: () => void;
        canModerate?: boolean;
        onForceMute?: (identity: string, muted: boolean) => void;
    }) => (
        <div data-testid="voice-bar" data-can-moderate={String(canModerate)}>
            <button type="button" onClick={onLeave}>
                leave voice bar
            </button>
            <button type="button" onClick={() => onForceMute?.("battler", true)}>
                server mute battler
            </button>
        </div>
    ),
}));

vi.mock("../ChatComposer/ChatComposer", () => ({
    ChatComposer: ({
        roomId,
        draftRecipientId,
        extraActions,
        onTyping,
        onEditLast,
        mentionPool,
        timeoutUntil,
    }: {
        roomId: string | null;
        draftRecipientId: string | null;
        extraActions?: React.ReactNode;
        onTyping?: () => void;
        onEditLast?: () => void;
        mentionPool?: User[];
        timeoutUntil?: string;
    }) => (
        <div
            data-testid="composer"
            data-room-id={roomId ?? ""}
            data-draft-recipient={draftRecipientId ?? ""}
            data-has-typing={String(Boolean(onTyping))}
            data-has-edit-last={String(Boolean(onEditLast))}
            data-mention-pool={mentionPool === undefined ? "none" : mentionPool.map(u => u.username).join(",")}
            data-timeout-until={timeoutUntil ?? ""}
        >
            {extraActions}
        </div>
    ),
}));

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeOther(overrides: Partial<User> = {}): User {
    return { id: "u2", username: "battler", display_name: "Battler", ...overrides };
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({
        members: [makePublicUser(), makeOther()],
        created_at: "2026-07-01T00:00:00Z",
        ...overrides,
    });
}

const controllerDefaults = makeDmController();

function makeVoice(overrides: Partial<DmController["voice"]> = {}): DmController["voice"] {
    return { ...controllerDefaults.voice, ...overrides };
}

function makeController(overrides: Partial<DmController> = {}): DmController {
    const user = overrides.user === undefined ? viewer : overrides.user;

    return makeDmController({ user, capabilities: roomCapabilities(overrides.activeRoom ?? null, user), ...overrides });
}

function makeThread(overrides: Partial<DmThreadView> = {}): DmThreadView {
    return {
        anchor: { highlightedMsgId: null, handleJumpToMessage: vi.fn(() => Promise.resolve()) },
        mentionPool: [],
        viewerTimedOut: false,
        searchOpen: false,
        setSearchOpen: vi.fn(),
        ...overrides,
    };
}

function renderView(overrides: Partial<DmController> = {}, threadOverrides: Partial<DmThreadView> = {}) {
    const controller = makeController(overrides);
    const thread = makeThread(threadOverrides);
    const result = renderWithProviders(<MobileDmView controller={controller} thread={thread} />);

    return { ...result, controller, thread };
}

function roomView(overrides: Partial<DmController> = {}, threadOverrides: Partial<DmThreadView> = {}) {
    return renderView(
        {
            mobileView: "room",
            activeRoom: makeRoom(),
            activeRoomId: "room-1",
            ...overrides,
        },
        threadOverrides,
    );
}

beforeEach(() => {
    mocks.forceMuteVoiceParticipant.mockResolvedValue(undefined);
});

describe("MobileDmView", () => {
    it("renders nothing until the viewer is signed in", () => {
        // given
        const user = null;

        // when
        const { container } = renderView({ user });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("tells the viewer they have no conversations yet", () => {
        // given
        const rooms: ChatRoom[] = [];

        // when
        renderView({ rooms });

        // then
        expect(screen.getByText("Messages")).toBeInTheDocument();
        expect(screen.getByText("No conversations yet")).toBeInTheDocument();
    });

    it("lists each conversation by the other person in it", () => {
        // given
        const rooms = [makeRoom({ id: "room-1" })];

        // when
        renderView({ rooms });

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.queryByText("No conversations yet")).not.toBeInTheDocument();
    });

    it("falls back to the room name for a conversation with nobody else in it", () => {
        // given
        const rooms = [makeRoom({ id: "room-1", type: "group", name: "The Study" })];

        // when
        renderView({ rooms });

        // then
        expect(screen.getByText("The Study")).toBeInTheDocument();
    });

    it("marks a conversation that has something unread", () => {
        // given
        const rooms = [makeRoom({ id: "room-1", unread: true }), makeRoom({ id: "room-2", unread: false })];

        // when
        renderView({ rooms });

        // then
        expect(screen.getAllByLabelText("unread")).toHaveLength(1);
    });

    it("opens the conversation the viewer tapped", async () => {
        // given
        const handleRoomSelect = vi.fn();
        const user = userEvent.setup();
        renderView({ handleRoomSelect, rooms: [makeRoom({ id: "room-1" })] });

        // when
        await user.click(screen.getByText("Battler"));

        // then
        expect(handleRoomSelect).toHaveBeenCalledWith("room-1");
    });

    it("opens the new message dialog from the list header", async () => {
        // given
        const setShowNewDm = vi.fn();
        const user = userEvent.setup();
        renderView({ setShowNewDm });

        // when
        await user.click(screen.getByRole("button", { name: "New DM" }));

        // then
        expect(setShowNewDm).toHaveBeenCalledWith(true);
    });

    it("suggests mutual followers while the search box is empty", () => {
        // given
        const dmMutuals = [makeOther({ id: "u3", display_name: "Ronove" })];

        // when
        renderView({ showNewDm: true, dmSearch: "", dmMutuals, dmResults: [makeOther({ display_name: "Ignored" })] });

        // then
        expect(screen.getByText("Ronove")).toBeInTheDocument();
        expect(screen.queryByText("Ignored")).not.toBeInTheDocument();
    });

    it("shows the search results once the viewer has typed something", () => {
        // given
        const dmSearch = "ron";

        // when
        renderView({
            showNewDm: true,
            dmSearch,
            dmResults: [makeOther({ id: "u3", display_name: "Ronove" })],
            dmMutuals: [makeOther({ id: "u4", display_name: "Ignored" })],
        });

        // then
        expect(screen.getByText("Ronove")).toBeInTheDocument();
        expect(screen.queryByText("Ignored")).not.toBeInTheDocument();
    });

    it("records what the viewer typed into the search box", async () => {
        // given
        const setDmSearch = vi.fn();
        const user = userEvent.setup();
        renderView({ showNewDm: true, setDmSearch });

        // when
        await user.type(screen.getByPlaceholderText("Search users..."), "r");

        // then
        expect(setDmSearch).toHaveBeenCalledWith("r");
    });

    it("reports why a conversation could not be opened", () => {
        // given
        const dmError = "That person does not accept messages";

        // when
        renderView({ showNewDm: true, dmError });

        // then
        expect(screen.getByText("That person does not accept messages")).toBeInTheDocument();
    });

    it("starts a conversation with the person the viewer picked", async () => {
        // given
        const handleSelectUser = vi.fn();
        const target = makeOther({ id: "u3", display_name: "Ronove" });
        const user = userEvent.setup();
        renderView({ showNewDm: true, dmMutuals: [target], handleSelectUser });

        // when
        await user.click(screen.getByText("Ronove"));

        // then
        expect(handleSelectUser).toHaveBeenCalledWith(target);
    });

    it("blocks a second pick while a conversation is being created", () => {
        // given
        const dmCreating = true;

        // when
        renderView({ showNewDm: true, dmCreating, dmMutuals: [makeOther({ id: "u3", display_name: "Ronove" })] });

        // then
        expect(screen.getByText("Ronove").closest("button")).toBeDisabled();
    });

    it("keeps the new message dialog closed until it is asked for", () => {
        // given
        const showNewDm = false;

        // when
        renderView({ showNewDm, dmMutuals: [makeOther({ id: "u3", display_name: "Ronove" })] });

        // then
        expect(screen.queryByText("New Direct Message")).not.toBeInTheDocument();
    });

    it("goes back to the conversation list from an open conversation", async () => {
        // given
        const handleMobileBack = vi.fn();
        const user = userEvent.setup();
        roomView({ handleMobileBack });

        // when
        await user.click(screen.getByLabelText("Back to conversations"));

        // then
        expect(handleMobileBack).toHaveBeenCalledTimes(1);
    });

    it("offers to delete an existing conversation", async () => {
        // given
        const handleDeleteChat = vi.fn();
        const user = userEvent.setup();
        roomView({ handleDeleteChat });

        // when
        await user.click(screen.getByLabelText("Delete chat"));

        // then
        expect(handleDeleteChat).toHaveBeenCalledTimes(1);
    });

    it("mutes an open conversation from the top bar", async () => {
        // given
        const handleToggleMute = vi.fn();
        const user = userEvent.setup();
        roomView({ handleToggleMute });

        // when
        await user.click(screen.getByLabelText("Mute notifications"));

        // then
        expect(handleToggleMute).toHaveBeenCalledTimes(1);
    });

    it("offers to unmute a conversation the viewer already silenced", () => {
        // given
        const activeRoom = makeRoom({ viewer_muted: true });

        // when
        roomView({ activeRoom });

        // then
        expect(screen.getByLabelText("Unmute notifications")).toBeInTheDocument();
        expect(screen.queryByLabelText("Mute notifications")).not.toBeInTheDocument();
    });

    it("blocks a second mute while the first is still in flight", () => {
        // given
        const mutePending = true;

        // when
        roomView({ mutePending });

        // then
        expect(screen.getByLabelText("Mute notifications")).toBeDisabled();
    });

    it("keeps the mute control away from a conversation that does not exist yet", () => {
        // given
        const draftRecipient = makeOther();

        // when
        renderView({ mobileView: "room", draftRecipient });

        // then
        expect(screen.queryByLabelText("Mute notifications")).not.toBeInTheDocument();
    });

    it("offers to abandon a draft instead of deleting it", async () => {
        // given
        const setDraftRecipient = vi.fn();
        const user = userEvent.setup();
        renderView({
            mobileView: "room",
            draftRecipient: makeOther(),
            setDraftRecipient,
        });

        // when
        await user.click(screen.getByLabelText("Cancel"));

        // then
        expect(setDraftRecipient).toHaveBeenCalledWith(null);
        expect(screen.queryByLabelText("Delete chat")).not.toBeInTheDocument();
    });

    it("prompts the viewer to send the first message of a draft conversation", () => {
        // given
        const draftRecipient = makeOther({ display_name: "Battler" });

        // when
        renderView({ mobileView: "room", draftRecipient });

        // then
        expect(screen.getByText(/Send your first message to/)).toHaveTextContent("Send your first message to Battler.");
        expect(screen.queryByTestId("dm-messages")).not.toBeInTheDocument();
    });

    it("shows the message history once a conversation exists", () => {
        // given
        const activeRoom = makeRoom();

        // when
        roomView({ activeRoom });

        // then
        expect(screen.getByTestId("dm-messages")).toBeInTheDocument();
        expect(screen.queryByText(/Send your first message/)).not.toBeInTheDocument();
    });

    it("keeps the voice bar away while nobody has joined the call", () => {
        // given
        const voice = makeVoice({ status: "idle", room: null });

        // when
        roomView({ voice });

        // then
        expect(screen.queryByTestId("voice-bar")).not.toBeInTheDocument();
    });

    it("shows the voice bar once the viewer is connected to the call", () => {
        // given
        const voice = makeVoice({ status: "connected", room: new Room() });

        // when
        roomView({ voice });

        // then
        expect(screen.getByTestId("voice-bar")).toBeInTheDocument();
    });

    it("treats an ordinary member as unable to moderate the call", () => {
        // given
        const voice = makeVoice({ status: "connected", room: new Room() });

        // when
        roomView({ voice });

        // then
        expect(screen.getByTestId("voice-bar")).toHaveAttribute("data-can-moderate", "false");
    });

    it("lets site staff moderate the call", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice", role: "admin" });

        // when
        roomView({ user, voice: makeVoice({ status: "connected", room: new Room() }) });

        // then
        expect(screen.getByTestId("voice-bar")).toHaveAttribute("data-can-moderate", "true");
    });

    it("sends a server mute to the room the conversation belongs to", async () => {
        // given
        const user = userEvent.setup();
        roomView({ voice: makeVoice({ status: "connected", room: new Room() }) });

        // when
        await user.click(screen.getByRole("button", { name: "server mute battler" }));

        // then
        expect(mocks.forceMuteVoiceParticipant).toHaveBeenCalledWith("room-1", "battler", true);
    });

    it("tells the moderator when a server mute did not take", async () => {
        // given
        mocks.forceMuteVoiceParticipant.mockRejectedValue(new Error("LiveKit said no"));
        const showToast = vi.fn();
        const user = userEvent.setup();
        roomView({ showToast, voice: makeVoice({ status: "connected", room: new Room() }) });

        // when
        await user.click(screen.getByRole("button", { name: "server mute battler" }));

        // then
        await waitFor(() => {
            expect(showToast).toHaveBeenCalledWith("LiveKit said no");
        });
    });

    it("only shows who is typing inside a real conversation", () => {
        // given
        const typingNames = ["Battler"];

        // when
        roomView({ typingNames });

        // then
        expect(screen.getByText(/is typing/)).toHaveTextContent("Battler is typing...");
    });

    it("hides the typing indicator while the conversation is still a draft", () => {
        // given
        const typingNames = ["Battler"];

        // when
        renderView({ mobileView: "room", draftRecipient: makeOther(), typingNames });

        // then
        expect(screen.queryByText(/is typing/)).not.toBeInTheDocument();
    });

    it("points the composer at the open conversation", () => {
        // given
        const activeRoomId = "room-1";

        // when
        roomView({ activeRoomId });

        // then
        const composer = screen.getByTestId("composer");
        expect(composer).toHaveAttribute("data-room-id", "room-1");
        expect(composer).toHaveAttribute("data-draft-recipient", "");
        expect(composer).toHaveAttribute("data-has-typing", "true");
    });

    it("points the composer at the draft recipient when there is no room yet", () => {
        // given
        const draftRecipient = makeOther({ id: "u2" });

        // when
        renderView({ mobileView: "room", draftRecipient });

        // then
        const composer = screen.getByTestId("composer");
        expect(composer).toHaveAttribute("data-room-id", "");
        expect(composer).toHaveAttribute("data-draft-recipient", "u2");
        expect(composer).toHaveAttribute("data-has-typing", "false");
    });

    it("offers the voice control only once the conversation exists", () => {
        // given
        const voiceEnabled = true;

        // when
        roomView({ voiceEnabled });

        // then
        expect(screen.getByTitle("Join voice")).toBeInTheDocument();
    });

    it("hides the voice control from a draft conversation", () => {
        // given
        const draftRecipient = makeOther();

        // when
        renderView({ mobileView: "room", draftRecipient, voiceEnabled: true });

        // then
        expect(screen.queryByTitle("Join voice")).not.toBeInTheDocument();
    });

    it("shows the lightbox for the image the viewer opened", () => {
        // given
        const lightboxSrc = "https://cdn.example/photo.png";

        // when
        roomView({ lightboxSrc });

        // then
        expect(screen.getByRole("dialog")).toBeInTheDocument();
    });

    it("opens the message search from the top bar", async () => {
        // given
        const user = userEvent.setup();
        const { thread } = roomView();

        // when
        await user.click(screen.getByLabelText("Search messages"));

        // then
        expect(thread.setSearchOpen).toHaveBeenCalledWith(true);
    });

    it("keeps the search panel out of the way until it is opened", () => {
        // given
        const searchOpen = false;

        // when
        roomView({}, { searchOpen });

        // then
        expect(screen.queryByTestId("search-panel")).not.toBeInTheDocument();
    });

    it("searches the conversation that is open", () => {
        // given
        const searchOpen = true;

        // when
        roomView({}, { searchOpen });

        // then
        expect(screen.getByTestId("search-panel")).toHaveAttribute("data-room", "room-1");
    });

    it("offers no search from a conversation that does not exist yet", () => {
        // given
        const draftRecipient = makeOther();

        // when
        renderView({ mobileView: "room", draftRecipient });

        // then
        expect(screen.queryByLabelText("Search messages")).not.toBeInTheDocument();
    });

    it("gives the composer the conversation's members and the viewer's timeout", () => {
        // given
        const mentionPool = [makeOther({ id: "u3", username: "ronove" })];
        const viewerTimeoutUntil = "2026-08-02T12:00:00Z";

        // when
        roomView({}, { mentionPool, viewerTimeoutUntil });

        // then
        const composer = screen.getByTestId("composer");
        expect(composer).toHaveAttribute("data-mention-pool", "ronove");
        expect(composer).toHaveAttribute("data-timeout-until", "2026-08-02T12:00:00Z");
    });

    it("leaves a draft conversation on the site-wide mention search", () => {
        // given
        const draftRecipient = makeOther();

        // when
        renderView(
            { mobileView: "room", draftRecipient },
            { mentionPool: [makeOther({ id: "u3", username: "ronove" })], viewerTimeoutUntil: "2026-08-02T12:00:00Z" },
        );

        // then
        const composer = screen.getByTestId("composer");
        expect(composer).toHaveAttribute("data-mention-pool", "none");
        expect(composer).toHaveAttribute("data-timeout-until", "");
    });

    it("highlights the message the viewer was sent to", () => {
        // given
        const anchor = { highlightedMsgId: "m5", handleJumpToMessage: vi.fn(() => Promise.resolve()) };

        // when
        roomView({}, { anchor });

        // then
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-highlighted", "m5");
    });

    it("locks the messages while the viewer is timed out", () => {
        // given
        const viewerTimedOut = true;

        // when
        roomView({}, { viewerTimedOut });

        // then
        expect(screen.getByTestId("dm-messages")).toHaveAttribute("data-timed-out", "true");
    });
});
