import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, UserProfile, WSMessage } from "../../../types/api";
import { RoomChatPanel } from "./RoomChatPanel";

const mocks = vi.hoisted(() => ({
    useMessageHistory: vi.fn(),
    useBlockedUserIds: vi.fn(),
    handleEditMessage: vi.fn(),
    handleIncomingChatMessage: vi.fn(),
    applySharedChatWSBranch: vi.fn(),
    addMessage: vi.fn(),
    scrollToBottomInstant: vi.fn(),
    handleScroll: vi.fn(),
    setMessages: vi.fn(),
}));

vi.mock("../../../hooks/useMessageHistory", () => ({ useMessageHistory: mocks.useMessageHistory }));

vi.mock("../../../hooks/useBlockedUserIds", () => ({ useBlockedUserIds: mocks.useBlockedUserIds }));

vi.mock("../../../hooks/useChatMessageHandlers", () => ({
    useChatMessageHandlers: () => ({ handleEditMessage: mocks.handleEditMessage }),
}));

vi.mock("../../../utils/chatStream", () => ({
    handleIncomingChatMessage: mocks.handleIncomingChatMessage,
    applySharedChatWSBranch: mocks.applySharedChatWSBranch,
}));

vi.mock("../MessageBubble/MessageBubble", () => ({
    MessageBubble: (props: { message: ChatMessage }) => <div data-testid="bubble">{props.message.body}</div>,
}));

vi.mock("../ChatComposer/ChatComposer", () => ({
    ChatComposer: (props: { roomId: string | null }) => <div data-testid="composer">{props.roomId}</div>,
}));

vi.mock("../../Lightbox/Lightbox", () => ({
    Lightbox: (props: { src: string }) => <div data-testid="lightbox">{props.src}</div>,
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "msg-1",
        room_id: "session-7",
        sender: { id: "sender-1", username: "beatrice", display_name: "Beatrice" },
        body: "the golden truth",
        is_system: false,
        created_at: "2026-08-01T10:05:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

function stubHistory(messages: ChatMessage[] = []) {
    mocks.useMessageHistory.mockReturnValue({
        messages,
        setMessages: mocks.setMessages,
        hasMore: false,
        loadingMore: false,
        containerRef: { current: null },
        contentRef: { current: null },
        endRef: { current: null },
        scrollToBottomInstant: mocks.scrollToBottomInstant,
        handleScroll: mocks.handleScroll,
        addMessage: mocks.addMessage,
    });
}

function renderPanel(
    props: Partial<React.ComponentProps<typeof RoomChatPanel>> = {},
    user: UserProfile | null = viewer,
) {
    const listeners: ((msg: WSMessage) => void)[] = [];
    const result = renderWithProviders(<RoomChatPanel roomId="session-7" title="Party chat" canSend {...props} />, {
        user,
        notification: {
            addWSListener: listener => {
                listeners.push(listener);
                return () => {};
            },
        },
    });

    return { ...result, listeners };
}

beforeEach(() => {
    mocks.useBlockedUserIds.mockReturnValue(new Set<string>());
    stubHistory();
});

describe("RoomChatPanel", () => {
    it("reads the history of whichever room it is pointed at", () => {
        // given a panel scoped to a watch party session's own room
        stubHistory([makeMessage()]);

        // when
        renderPanel();

        // then
        expect(mocks.useMessageHistory).toHaveBeenCalledWith("session-7", undefined);
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("lets a participant compose into that same room", () => {
        // given

        // when
        renderPanel();

        // then
        expect(screen.getByTestId("composer")).toHaveTextContent("session-7");
    });

    it("withholds the composer while the room is not sendable", () => {
        // given a room the viewer may read but not post to

        // when
        renderPanel({ canSend: false });

        // then
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
    });

    it("uses the caller's wording to invite a signed out visitor in", () => {
        // given

        // when
        renderPanel({ loginPrompt: "to join the party chat." }, null);

        // then
        expect(screen.getByRole("link", { name: "Log in" })).toHaveAttribute("href", "/login");
        expect(screen.getByText(/to join the party chat/)).toBeInTheDocument();
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
    });

    it("routes an incoming message through the shared handler for its own room", () => {
        // given
        const { listeners } = renderPanel();
        const incoming = makeMessage({ id: "msg-live" });

        // when
        listeners[0]({ type: "chat_message", data: incoming } as WSMessage);

        // then
        expect(mocks.handleIncomingChatMessage).toHaveBeenCalledWith(
            incoming,
            "session-7",
            mocks.setMessages,
            expect.any(Function),
        );
    });

    it("hands every other chat event to the shared branch", () => {
        // given
        const { listeners } = renderPanel();
        const event = { type: "chat_message_deleted", data: { room_id: "session-7", id: "msg-1" } } as WSMessage;

        // when
        listeners[0](event);

        // then
        expect(mocks.applySharedChatWSBranch).toHaveBeenCalledWith(
            event,
            expect.objectContaining({ activeRoomId: "session-7" }),
        );
    });

    it("stays inert until it has a room to show", () => {
        // given a party that has not been joined yet

        // when
        const { listeners } = renderPanel({ roomId: undefined, canSend: false, notice: "Joining chat..." });

        // then
        expect(listeners).toHaveLength(0);
        expect(screen.getByText("Joining chat...")).toBeInTheDocument();
    });
});
