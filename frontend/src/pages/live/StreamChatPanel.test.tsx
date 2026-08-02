import { screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { ChatMessage, UserProfile, WSMessage } from "../../types/api";
import { StreamChatPanel } from "./StreamChatPanel";

const mocks = vi.hoisted(() => ({
    joinStreamChat: vi.fn(),
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

vi.mock("../../api/endpoints", () => ({ joinStreamChat: mocks.joinStreamChat }));

vi.mock("../../hooks/useMessageHistory", () => ({ useMessageHistory: mocks.useMessageHistory }));

vi.mock("../../hooks/useBlockedUserIds", () => ({ useBlockedUserIds: mocks.useBlockedUserIds }));

vi.mock("../../hooks/useChatMessageHandlers", () => ({
    useChatMessageHandlers: () => ({ handleEditMessage: mocks.handleEditMessage }),
}));

vi.mock("../../utils/chatStream", () => ({
    handleIncomingChatMessage: mocks.handleIncomingChatMessage,
    applySharedChatWSBranch: mocks.applySharedChatWSBranch,
}));

vi.mock("../../components/chat/MessageBubble/MessageBubble", () => ({
    MessageBubble: (props: { message: ChatMessage; isOwn: boolean; senderBlocked: boolean }) => (
        <div data-testid="bubble" data-own={String(props.isOwn)} data-blocked={String(props.senderBlocked)}>
            {props.message.body}
        </div>
    ),
}));

vi.mock("../../components/chat/ChatComposer/ChatComposer", () => ({
    ChatComposer: (props: { roomId: string | null }) => <div data-testid="composer">{props.roomId}</div>,
}));

vi.mock("../../components/Lightbox/Lightbox", () => ({
    Lightbox: (props: { src: string }) => <div data-testid="lightbox">{props.src}</div>,
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "msg-1",
        room_id: "stream-1",
        sender: { id: "sender-1", username: "beatrice", display_name: "Beatrice" },
        body: "Golden butterflies everywhere",
        is_system: false,
        created_at: "2026-02-01T12:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

interface HistoryOptions {
    messages?: ChatMessage[];
    hasMore?: boolean;
    loadingMore?: boolean;
}

function stubHistory(options: HistoryOptions = {}) {
    mocks.useMessageHistory.mockReturnValue({
        messages: options.messages ?? [],
        setMessages: mocks.setMessages,
        hasMore: options.hasMore ?? false,
        loadingMore: options.loadingMore ?? false,
        containerRef: { current: null },
        contentRef: { current: null },
        endRef: { current: null },
        scrollToBottomInstant: mocks.scrollToBottomInstant,
        handleScroll: mocks.handleScroll,
        addMessage: mocks.addMessage,
    });
}

function renderPanel(options: { user?: UserProfile | null; isLive?: boolean; streamId?: string } = {}) {
    const listeners: ((msg: WSMessage) => void)[] = [];
    const result = renderWithProviders(
        <StreamChatPanel streamId={options.streamId ?? "stream-1"} isLive={options.isLive ?? true} />,
        {
            user: options.user === undefined ? viewer : options.user,
            notification: {
                addWSListener: listener => {
                    listeners.push(listener);
                    return () => {};
                },
            },
        },
    );

    return { ...result, listeners };
}

beforeEach(() => {
    mocks.joinStreamChat.mockResolvedValue(undefined);
    mocks.useBlockedUserIds.mockReturnValue(new Set<string>());
    stubHistory();
});

describe("StreamChatPanel signed out", () => {
    it("asks a signed out visitor to log in before chatting", () => {
        // given
        const user = null;

        // when
        renderPanel({ user });

        // then
        expect(screen.getByRole("link", { name: "Log in" })).toHaveAttribute("href", "/login");
        expect(screen.getByText(/to join the chat/)).toBeInTheDocument();
    });

    it("never tries to join the chat on behalf of a signed out visitor", () => {
        // given
        const user = null;

        // when
        renderPanel({ user });

        // then
        expect(mocks.joinStreamChat).not.toHaveBeenCalled();
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
    });
});

describe("StreamChatPanel joining", () => {
    it("joins the chat of the stream being watched", async () => {
        // given
        const streamId = "stream-77";

        // when
        renderPanel({ streamId });

        // then
        await waitFor(() => {
            expect(mocks.joinStreamChat).toHaveBeenCalledWith("stream-77");
        });
    });

    it("says it is joining until the server has let it in", () => {
        // given
        mocks.joinStreamChat.mockReturnValue(new Promise<void>(() => {}));

        // when
        renderPanel();

        // then
        expect(screen.getByText("Joining chat...")).toBeInTheDocument();
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
    });

    it("offers the composer once the join has succeeded", async () => {
        // given
        mocks.joinStreamChat.mockResolvedValue(undefined);

        // when
        renderPanel({ streamId: "stream-5" });

        // then
        expect(await screen.findByTestId("composer")).toHaveTextContent("stream-5");
        expect(screen.queryByText("Joining chat...")).not.toBeInTheDocument();
    });

    it("admits when the chat could not be joined", async () => {
        // given
        mocks.joinStreamChat.mockRejectedValue(new Error("no room at the inn"));

        // when
        renderPanel();

        // then
        expect(await screen.findByText("Couldn't join the chat.")).toBeInTheDocument();
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
    });

    it("does not try to join while the stream is offline", () => {
        // given
        const isLive = false;

        // when
        renderPanel({ isLive });

        // then
        expect(mocks.joinStreamChat).not.toHaveBeenCalled();
        expect(screen.getByText("Chat is closed while the stream is offline.")).toBeInTheDocument();
    });

    it("only reads back history once it has joined a live chat", async () => {
        // given
        mocks.joinStreamChat.mockResolvedValue(undefined);

        // when
        renderPanel({ streamId: "stream-3" });

        // then
        expect(mocks.useMessageHistory).toHaveBeenCalledWith(undefined, 50);
        await waitFor(() => {
            expect(mocks.useMessageHistory).toHaveBeenLastCalledWith("stream-3", 50);
        });
    });
});

describe("StreamChatPanel messages", () => {
    it("draws every message it has in hand", async () => {
        // given
        stubHistory({
            messages: [makeMessage({ id: "m1", body: "first" }), makeMessage({ id: "m2", body: "second" })],
        });

        // when
        renderPanel();

        // then
        const bubbles = await screen.findAllByTestId("bubble");
        expect(bubbles.map(bubble => bubble.textContent)).toEqual(["first", "second"]);
    });

    it("marks the viewer's own message as theirs", () => {
        // given
        stubHistory({ messages: [makeMessage({ sender: { ...makeMessage().sender, id: viewer.id } })] });

        // when
        renderPanel();

        // then
        expect(screen.getByTestId("bubble")).toHaveAttribute("data-own", "true");
    });

    it("flags a message from somebody the viewer has blocked", () => {
        // given
        mocks.useBlockedUserIds.mockReturnValue(new Set(["sender-1"]));
        stubHistory({ messages: [makeMessage()] });

        // when
        renderPanel();

        // then
        expect(screen.getByTestId("bubble")).toHaveAttribute("data-blocked", "true");
    });

    it("invites the viewer to scroll up when there is older history", () => {
        // given
        stubHistory({ hasMore: true, loadingMore: false });

        // when
        renderPanel();

        // then
        expect(screen.getByText("Scroll up for more")).toBeInTheDocument();
    });

    it("says it is fetching while older history is on its way", () => {
        // given
        stubHistory({ hasMore: true, loadingMore: true });

        // when
        renderPanel();

        // then
        expect(screen.getByText("Loading older messages...")).toBeInTheDocument();
    });
});

describe("StreamChatPanel live updates", () => {
    it("hands an incoming chat message to the stream handler", async () => {
        // given
        const { listeners } = renderPanel({ streamId: "stream-4" });
        await screen.findByTestId("composer");
        const incoming = makeMessage({ room_id: "stream-4" });

        // when
        listeners[listeners.length - 1]({ type: "chat_message", data: incoming });

        // then
        expect(mocks.handleIncomingChatMessage).toHaveBeenCalledWith(
            incoming,
            "stream-4",
            mocks.setMessages,
            expect.any(Function),
        );
        expect(mocks.applySharedChatWSBranch).not.toHaveBeenCalled();
    });

    it("passes any other chat event to the shared branch", async () => {
        // given
        const { listeners } = renderPanel({ streamId: "stream-4" });
        await screen.findByTestId("composer");
        const event: WSMessage = { type: "chat_message_deleted", data: { room_id: "stream-4", id: "m1" } };

        // when
        listeners[listeners.length - 1](event);

        // then
        expect(mocks.applySharedChatWSBranch).toHaveBeenCalledWith(
            event,
            expect.objectContaining({ activeRoomId: "stream-4", setMessages: mocks.setMessages }),
        );
    });

    it("ignores live chat events while the stream is offline", () => {
        // given
        const { listeners } = renderPanel({ isLive: false });

        // when
        for (const listener of listeners) {
            listener({ type: "chat_message", data: makeMessage() });
        }

        // then
        expect(mocks.handleIncomingChatMessage).not.toHaveBeenCalled();
    });
});

describe("StreamChatPanel lightbox", () => {
    it("keeps the lightbox shut until an image is opened", async () => {
        // given
        stubHistory({ messages: [makeMessage()] });

        // when
        renderPanel();

        // then
        await screen.findByTestId("bubble");
        expect(screen.queryByTestId("lightbox")).not.toBeInTheDocument();
    });

    it("starts a fresh panel when the viewer switches stream", async () => {
        // given
        const { rerender } = renderPanel({ streamId: "stream-1" });
        await screen.findByTestId("composer");

        // when
        rerender(<StreamChatPanel streamId="stream-2" isLive />);

        // then
        await waitFor(() => {
            expect(mocks.joinStreamChat).toHaveBeenCalledWith("stream-2");
        });
    });
});
