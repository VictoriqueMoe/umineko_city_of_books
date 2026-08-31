import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { emitRealtimeEvent, type RealtimeTestEvent } from "../../test-utils/ws";
import type { ChatMessage, UserProfile } from "../../types/api";
import { StreamChatPanel } from "./StreamChatPanel";

const mocks = vi.hoisted(() => ({
    joinStreamChat: vi.fn(),
    getStreamViewerToken: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
    uploadStreamThumbnail: vi.fn(),
    useMessageHistory: vi.fn(),
    useBlockedUserIds: vi.fn(),
    handleEditMessage: vi.fn(),
    markReadDebounced: vi.fn(),
    addMessage: vi.fn(),
    scrollToBottomInstant: vi.fn(),
    handleScroll: vi.fn(),
    setMessages: vi.fn(),
}));

vi.mock("../../api/endpoints/stream", () => ({
    joinStreamChat: mocks.joinStreamChat,
    getStreamViewerToken: mocks.getStreamViewerToken,
    resetStreamCredentials: mocks.resetStreamCredentials,
    startStream: mocks.startStream,
    stopStream: mocks.stopStream,
    updateStreamTitle: mocks.updateStreamTitle,
    uploadStreamThumbnail: mocks.uploadStreamThumbnail,
}));

vi.mock("../../hooks/useMessageHistory", () => ({ useMessageHistory: mocks.useMessageHistory }));

vi.mock("../../hooks/useBlockedUserIds", () => ({ useBlockedUserIds: mocks.useBlockedUserIds }));

vi.mock("../../hooks/useChatMessageHandlers", () => ({
    useChatMessageHandlers: () => ({ handleEditMessage: mocks.handleEditMessage }),
}));

vi.mock("../../hooks/chat/useDebouncedMarkChatRoomRead", () => ({
    useDebouncedMarkChatRoomRead: () => mocks.markReadDebounced,
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
    return makeChatMessage({
        id: "msg-1",
        room_id: "stream-1",
        sender: { id: "sender-1", username: "beatrice", display_name: "Beatrice" },
        body: "Golden butterflies everywhere",
        created_at: "2026-02-01T12:00:00Z",
        ...overrides,
    });
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

function renderPanel(
    options: {
        user?: UserProfile | null;
        isLive?: boolean;
        streamId?: string;
        onPopOut?: () => void;
    } = {},
) {
    return renderWithProviders(
        <StreamChatPanel
            streamId={options.streamId ?? "stream-1"}
            isLive={options.isLive ?? true}
            onPopOut={options.onPopOut}
        />,
        { user: options.user === undefined ? viewer : options.user },
    );
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

    it("invites the viewer to scroll up when there is older history", async () => {
        // given
        stubHistory({ hasMore: true, loadingMore: false });

        // when
        renderPanel();

        // then
        await screen.findByTestId("composer");
        expect(screen.getByText("Scroll up for more")).toBeInTheDocument();
    });

    it("says it is fetching while older history is on its way", async () => {
        // given
        stubHistory({ hasMore: true, loadingMore: true });

        // when
        renderPanel();

        // then
        await screen.findByTestId("composer");
        expect(screen.getByText("Loading older messages...")).toBeInTheDocument();
    });
});

describe("StreamChatPanel live updates", () => {
    function lastPatch(): (current: ChatMessage[]) => ChatMessage[] {
        const calls = mocks.setMessages.mock.calls;

        return calls[calls.length - 1][0] as (current: ChatMessage[]) => ChatMessage[];
    }

    it("takes an incoming chat message for the stream it joined", async () => {
        // given
        renderPanel({ streamId: "stream-4" });
        await screen.findByTestId("composer");
        const incoming = makeMessage({ id: "m-live", room_id: "stream-4" });

        // when
        emitRealtimeEvent({ type: "chat_message", data: incoming });

        // then
        expect(lastPatch()([])).toEqual([incoming]);
        expect(mocks.markReadDebounced).toHaveBeenCalledExactlyOnceWith("stream-4");
    });

    it("leaves a chat message for another room alone", async () => {
        // given
        renderPanel({ streamId: "stream-4" });
        await screen.findByTestId("composer");

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage({ room_id: "stream-9" }) });

        // then
        expect(mocks.setMessages).not.toHaveBeenCalled();
        expect(mocks.markReadDebounced).not.toHaveBeenCalled();
    });

    it("drops a message deleted in the stream's own room", async () => {
        // given
        renderPanel({ streamId: "stream-4" });
        await screen.findByTestId("composer");
        const event: RealtimeTestEvent = {
            type: "chat_message_deleted",
            data: { room_id: "stream-4", message_id: "m1" },
        };

        // when
        emitRealtimeEvent(event);

        // then
        expect(lastPatch()([makeMessage({ id: "m1", room_id: "stream-4" })])).toEqual([]);
    });

    it("ignores live chat events while the stream is offline", () => {
        // given
        renderPanel({ isLive: false });

        // when
        emitRealtimeEvent({ type: "chat_message", data: makeMessage() });

        // then
        expect(mocks.setMessages).not.toHaveBeenCalled();
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

describe("StreamChatPanel switching stream", () => {
    it("drops back to joining until the new stream has let it in", async () => {
        // given
        const { rerender } = renderPanel({ streamId: "stream-1" });
        expect(await screen.findByTestId("composer")).toHaveTextContent("stream-1");
        let letIn = () => {};
        mocks.joinStreamChat.mockReturnValue(
            new Promise<void>(resolve => {
                letIn = resolve;
            }),
        );

        // when
        rerender(<StreamChatPanel streamId="stream-2" isLive />);

        // then
        expect(screen.getByText("Joining chat...")).toBeInTheDocument();
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
        expect(mocks.useMessageHistory).toHaveBeenLastCalledWith(undefined, 50);
        await act(async () => {
            letIn();
        });
        expect(await screen.findByTestId("composer")).toHaveTextContent("stream-2");
    });

    it("forgets a failed join once the viewer moves to another stream", async () => {
        // given
        mocks.joinStreamChat.mockRejectedValueOnce(new Error("no room at the inn"));
        const { rerender } = renderPanel({ streamId: "stream-1" });
        await screen.findByText("Couldn't join the chat.");
        mocks.joinStreamChat.mockResolvedValue(undefined);

        // when
        rerender(<StreamChatPanel streamId="stream-2" isLive />);

        // then
        expect(screen.queryByText("Couldn't join the chat.")).not.toBeInTheDocument();
        expect(await screen.findByTestId("composer")).toHaveTextContent("stream-2");
    });
});

describe("StreamChatPanel pop out control", () => {
    const popOutLabel = "Open chat in its own window";

    it("offers no pop out control when the host page does not support one", async () => {
        // given
        const noHandler = undefined;

        // when
        renderPanel({ onPopOut: noHandler });

        // then
        await screen.findByTestId("composer");
        expect(screen.queryByRole("button", { name: popOutLabel })).not.toBeInTheDocument();
    });

    it("offers a pop out control when the host page supports one", async () => {
        // given
        const onPopOut = vi.fn();

        // when
        renderPanel({ onPopOut });

        // then
        expect(await screen.findByRole("button", { name: popOutLabel })).toBeInTheDocument();
    });

    it("asks the host page to pop the chat out when the control is used", async () => {
        // given
        const onPopOut = vi.fn();
        renderPanel({ onPopOut });

        // when
        await userEvent.click(await screen.findByRole("button", { name: popOutLabel }));

        // then
        expect(onPopOut).toHaveBeenCalledTimes(1);
    });

    it("keeps the pop out control away from a signed out visitor", () => {
        // given
        const onPopOut = vi.fn();

        // when
        renderPanel({ user: null, onPopOut });

        // then
        expect(screen.queryByRole("button", { name: popOutLabel })).not.toBeInTheDocument();
    });
});
