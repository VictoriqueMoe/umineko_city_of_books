import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PendingSend, ThreadEntry } from "../../../domain/chat/optimisticSend";
import { makeChatMessage, makePendingSend, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatSession, ChatSessionStatus, UseChatSessionOptions } from "../../../hooks/chat/useChatSession";
import type { ChatSender } from "../../../hooks/chat/useOptimisticSends";
import type { ChatMessage, UserProfile } from "../../../types/api";
import type { ComposerTarget } from "../ChatComposer/ChatComposer";
import { RoomChatPanel } from "./RoomChatPanel";

const mocks = vi.hoisted(() => ({
    useChatSession: vi.fn(),
    useBlockedUserIds: vi.fn(),
    send: vi.fn(),
    retrySend: vi.fn(),
    discardSend: vi.fn(),
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
    MessageBubble: (props: {
        message: ChatMessage;
        delivery?: Pick<PendingSend, "nonce" | "delivery" | "error">;
        onRetrySend?: (nonce: string) => void;
        onDiscardSend?: (nonce: string) => void;
    }) => (
        <div data-testid="bubble" data-delivery={props.delivery?.delivery ?? "delivered"}>
            {props.message.body}
            {props.delivery && (
                <button type="button" onClick={() => props.onRetrySend?.(props.delivery?.nonce ?? "")}>
                    retry
                </button>
            )}
        </div>
    ),
}));

vi.mock("../ChatComposer/ChatComposer", () => ({
    ChatComposer: (props: { target: ComposerTarget | null }) => (
        <div data-testid="composer">{props.target?.kind === "room" ? props.target.sender.roomId : ""}</div>
    ),
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
    pending?: PendingSend[];
    sender?: ChatSender | null;
    hasMore?: boolean;
    loadingMore?: boolean;
}

function stubSession({
    status = "live",
    messages = [],
    pending = [],
    sender = { roomId: "session-7", send: mocks.send },
    hasMore = false,
    loadingMore = false,
}: SessionStub = {}) {
    const entries: ThreadEntry[] = [...messages, ...pending];
    const session: ChatSession = {
        status,
        messages,
        entries,
        sends: {
            sender,
            draftSender: null,
            retry: mocks.retrySend,
            discard: mocks.discardSend,
        },
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
        stubSession({ status: "idle", sender: null });

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

    it("shows the viewer's unconfirmed message after the transcript as undelivered", () => {
        // given a stream chat message that is still on its way
        stubSession({ messages: [makeMessage()], pending: [makePendingSend({ body: "first!" })] });

        // when
        renderPanel();

        // then
        const bubbles = screen.getAllByTestId("bubble");
        expect(bubbles.map(bubble => bubble.getAttribute("data-delivery"))).toEqual(["delivered", "sending"]);
        expect(bubbles[1]).toHaveTextContent("first!");
    });

    it("lets the viewer retry a failed stream chat message from its bubble", () => {
        // given
        const failed = makePendingSend({ delivery: "failed", error: "network is down" });
        stubSession({ pending: [failed] });

        // when
        renderPanel();
        fireEvent.click(screen.getByRole("button", { name: "retry" }));

        // then
        expect(mocks.retrySend).toHaveBeenCalledExactlyOnceWith(failed.nonce);
    });
});
