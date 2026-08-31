import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatSession, ChatSessionStatus, UseChatSessionOptions } from "../../../hooks/chat/useChatSession";
import type { ChatMessage, UserProfile } from "../../../types/api";
import { RoomChatPanel } from "./RoomChatPanel";

const mocks = vi.hoisted(() => ({
    useChatSession: vi.fn(),
    useBlockedUserIds: vi.fn(),
    addMessage: vi.fn(),
    setMessages: vi.fn(),
    seedMessages: vi.fn(),
    loadUntilMessage: vi.fn(),
    resync: vi.fn(),
    onScroll: vi.fn(),
    toBottom: vi.fn(),
    toBottomInstant: vi.fn(),
    startEditing: vi.fn(),
    cancelEditing: vi.fn(),
    saveEdit: vi.fn(),
    removeMessage: vi.fn(),
    editLast: vi.fn(),
    noteTyping: vi.fn(),
    clearTyping: vi.fn(),
    resetTyping: vi.fn(),
}));

vi.mock("../../../hooks/chat/useChatSession", () => ({ useChatSession: mocks.useChatSession }));

vi.mock("../../../hooks/useBlockedUserIds", () => ({ useBlockedUserIds: mocks.useBlockedUserIds }));

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
    return makeChatMessage({
        id: "msg-1",
        room_id: "session-7",
        sender: { id: "sender-1", username: "beatrice", display_name: "Beatrice" },
        created_at: "2026-08-01T10:05:00Z",
        ...overrides,
    });
}

interface SessionStub {
    status?: ChatSessionStatus;
    messages?: ChatMessage[];
    hasMore?: boolean;
    loadingMore?: boolean;
}

function stubSession({ status = "live", messages = [], hasMore = false, loadingMore = false }: SessionStub = {}) {
    const session: ChatSession = {
        status,
        messages,
        history: {
            hasMore,
            loadingMore,
            setMessages: mocks.setMessages,
            addMessage: mocks.addMessage,
            seedMessages: mocks.seedMessages,
            loadUntilMessage: mocks.loadUntilMessage,
            resync: mocks.resync,
        },
        scroll: {
            containerRef: () => {},
            contentRef: () => {},
            endRef: { current: null },
            onScroll: mocks.onScroll,
            toBottom: mocks.toBottom,
            toBottomInstant: mocks.toBottomInstant,
        },
        editing: {
            messageId: null,
            start: mocks.startEditing,
            cancel: mocks.cancelEditing,
            save: mocks.saveEdit,
            remove: mocks.removeMessage,
            editLast: mocks.editLast,
        },
        typing: {
            userIds: [],
            note: mocks.noteTyping,
            clear: mocks.clearTyping,
            reset: mocks.resetTyping,
        },
    };

    mocks.useChatSession.mockReturnValue(session);
}

function renderPanel(
    props: Partial<React.ComponentProps<typeof RoomChatPanel>> = {},
    user: UserProfile | null = viewer,
) {
    return renderWithProviders(<RoomChatPanel roomId="session-7" title="Party chat" canSend {...props} />, { user });
}

function sessionOptions(): UseChatSessionOptions {
    return mocks.useChatSession.mock.calls[0][0] as UseChatSessionOptions;
}

beforeEach(() => {
    mocks.useBlockedUserIds.mockReturnValue(new Set<string>());
    stubSession();
});

describe("RoomChatPanel", () => {
    it("opens a session on whichever room it is pointed at", () => {
        // given a panel scoped to a watch party session's own room
        stubSession({ messages: [makeMessage()] });

        // when
        renderPanel();

        // then
        expect(sessionOptions()).toMatchObject({ roomId: "session-7", user: viewer, scrollMode: "instant" });
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("hands the session its message window", () => {
        // given

        // when
        renderPanel({ maxMessages: 50 });

        // then
        expect(sessionOptions().maxMessages).toBe(50);
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

    it("invites a scroll back while the session says there is more", () => {
        // given
        stubSession({ hasMore: true });

        // when
        renderPanel();

        // then
        expect(screen.getByText("Scroll up for more")).toBeInTheDocument();
    });

    it("says nothing about older messages when the session offers none", () => {
        // given
        stubSession({ hasMore: false });

        // when
        renderPanel();

        // then
        expect(screen.queryByText("Scroll up for more")).not.toBeInTheDocument();
    });

    it("stays inert until it has a room to show", () => {
        // given a party that has not been joined yet
        stubSession({ status: "idle" });

        // when
        renderPanel({ roomId: undefined, canSend: false, notice: "Joining chat..." });

        // then
        expect(screen.getByText("Joining chat...")).toBeInTheDocument();
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
        expect(screen.queryByText("This chat has ended.")).not.toBeInTheDocument();
    });

    it("says so when the room it was reading is gone", () => {
        // given a watch party whose room cascade deleted when the party ended
        stubSession({ status: "ended", messages: [makeMessage()] });

        // when
        renderPanel();

        // then the viewer keeps the transcript and is told why nothing more arrives
        expect(screen.getByText("This chat has ended.")).toBeInTheDocument();
        expect(screen.getByText("the golden truth")).toBeInTheDocument();
    });

    it("takes the composer away once the session has ended", () => {
        // given
        stubSession({ status: "ended" });

        // when
        renderPanel();

        // then a viewer cannot post into a room that no longer exists
        expect(screen.queryByTestId("composer")).not.toBeInTheDocument();
    });

    it("prefers the caller's own closing wording over the default", () => {
        // given a stream that has gone offline
        stubSession({ status: "ended" });

        // when
        renderPanel({
            roomId: undefined,
            canSend: false,
            closedNotice: "Chat is closed while the stream is offline.",
        });

        // then
        expect(screen.getByText("Chat is closed while the stream is offline.")).toBeInTheDocument();
        expect(screen.queryByText("This chat has ended.")).not.toBeInTheDocument();
    });
});
