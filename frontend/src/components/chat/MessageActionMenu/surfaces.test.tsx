import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage, makeChatRoom, makeDmRoom, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatSession } from "../../../hooks/chat/useChatSession";
import type { ChatMessage, ChatRoom, User } from "../../../types/api";
import { MessageList, type MessageListProps } from "../MessageList/MessageList";
import { RoomChatPanel } from "../RoomChatPanel/RoomChatPanel";

const mocks = vi.hoisted(() => ({
    useChatSession: vi.fn(),
    useBlockedUserIds: vi.fn(),
    setMessages: vi.fn(),
    addMessage: vi.fn(),
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

vi.mock("../ChatComposer/ChatComposer", () => ({
    ChatComposer: () => <div data-testid="composer" />,
}));

const viewer = makeUser({ id: "viewer-1", username: "beatrice", display_name: "Beatrice" });

const me: User = { id: viewer.id, username: "beatrice", display_name: "Beatrice" };

const other: User = { id: "u2", username: "battler", display_name: "Battler" };

function ownMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({ id: "m1", sender: me, ...overrides });
}

function theirMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({ id: "m1", sender: other, ...overrides });
}

function hostedRoom(): ChatRoom {
    return makeChatRoom({ name: "Rokkenjima", viewer_role: "host" });
}

function privateThread(): ChatRoom {
    return makeDmRoom({ members: [me, other] });
}

function listProps(
    room: ChatRoom,
    messages: ChatMessage[],
    overrides: Partial<MessageListProps> = {},
): MessageListProps {
    return {
        viewer,
        room,
        messages,
        hasMore: false,
        loadingMore: false,
        editingMessageId: null,
        matchesViewerMention: null,
        containerRef: { current: null },
        contentRef: { current: null },
        endRef: { current: null },
        onScroll: vi.fn(),
        onLightbox: vi.fn(),
        onReply: vi.fn(),
        onStartEditing: vi.fn(),
        onCancelEditing: vi.fn(),
        onToggleReaction: vi.fn(),
        onTogglePin: vi.fn(),
        onDelete: vi.fn(),
        onEdit: vi.fn(),
        classes: { messages: "messages", loadMoreBar: "load-more", empty: "empty" },
        ...overrides,
    };
}

function renderList(room: ChatRoom, messages: ChatMessage[], overrides: Partial<MessageListProps> = {}) {
    const props = listProps(room, messages, overrides);
    renderWithProviders(<MessageList {...props} />, { user: viewer });

    return props;
}

function stubSession(messages: ChatMessage[]) {
    const session: ChatSession = {
        status: "live",
        messages,
        history: {
            hasMore: false,
            loadingMore: false,
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

function renderPanel(messages: ChatMessage[]) {
    stubSession(messages);

    return renderWithProviders(<RoomChatPanel roomId="session-7" title="Stream chat" canSend />, { user: viewer });
}

function bubbleOf(id: string): HTMLElement {
    const bubble = document.getElementById(`chat-msg-${id}`);
    if (!bubble) {
        throw new Error(`no message bubble for ${id}`);
    }

    return bubble;
}

const ALL_LABELS = ["React", "Reply", "Pin message", "Unpin message", "Edit message", "Delete message"];

function labelOf(control: HTMLElement): string {
    const text = control.getAttribute("aria-label") ?? control.textContent ?? "";
    const label = ALL_LABELS.find(candidate => text.endsWith(candidate));
    if (!label) {
        throw new Error(`an action control carried no recognisable label: "${text}"`);
    }

    return label;
}

function hoverBarLabels(): string[] {
    return screen
        .queryAllByRole("button")
        .filter(button => button.hasAttribute("aria-label"))
        .map(labelOf);
}

function menuLabels(): string[] {
    return screen.queryAllByRole("menuitem").map(labelOf);
}

interface SurfaceCase {
    name: string;
    open: () => void;
    want: string[];
}

const surfaceCases: SurfaceCase[] = [
    {
        name: "a private thread showing the viewer's own message",
        open: () => renderList(privateThread(), [ownMessage()]),
        want: ["React", "Reply", "Pin message", "Edit message", "Delete message"],
    },
    {
        name: "a chat room showing another member's message to its host",
        open: () => renderList(hostedRoom(), [theirMessage()]),
        want: ["React", "Reply", "Pin message", "Delete message"],
    },
    {
        name: "a chat room showing another member's message to an ordinary member",
        open: () => renderList(makeChatRoom({ viewer_role: "member" }), [theirMessage()]),
        want: ["React", "Reply"],
    },
    {
        name: "a live stream chat showing the viewer's own message",
        open: () => renderPanel([ownMessage()]),
        want: ["Reply", "Edit message"],
    },
    {
        name: "a live stream chat showing somebody else's message",
        open: () => renderPanel([theirMessage()]),
        want: ["Reply"],
    },
];

beforeEach(() => {
    mocks.useBlockedUserIds.mockReturnValue(new Set<string>());
});

describe("the message action menu on every chat surface", () => {
    for (const surfaceCase of surfaceCases) {
        it(`offers the right click what the hover bar offers in ${surfaceCase.name}`, () => {
            // given
            surfaceCase.open();
            const hovered = hoverBarLabels();

            // when
            fireEvent.contextMenu(bubbleOf("m1"));

            // then
            expect(hovered).toEqual(surfaceCase.want);
            expect(menuLabels()).toEqual(hovered);
        });
    }

    it("reaches the message list's own reply handler", async () => {
        // given
        const user = userEvent.setup();
        const props = renderList(hostedRoom(), [theirMessage({ body: "a short claim" })]);
        fireEvent.contextMenu(bubbleOf("m1"));

        // when
        await user.click(screen.getByRole("menuitem", { name: "Reply" }));

        // then
        expect(props.onReply).toHaveBeenCalledExactlyOnceWith({
            id: "m1",
            senderName: "Battler",
            bodyPreview: "a short claim",
        });
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("reaches the message list's own pin handler", async () => {
        // given
        const user = userEvent.setup();
        const message = theirMessage();
        const props = renderList(hostedRoom(), [message]);
        fireEvent.contextMenu(bubbleOf("m1"));

        // when
        await user.click(screen.getByRole("menuitem", { name: "Pin message" }));

        // then
        expect(props.onTogglePin).toHaveBeenCalledExactlyOnceWith(message);
    });

    it("reaches the live stream chat's own editor", async () => {
        // given
        const user = userEvent.setup();
        const message = ownMessage();
        renderPanel([message]);
        fireEvent.contextMenu(bubbleOf("m1"));

        // when
        await user.click(screen.getByRole("menuitem", { name: "Edit message" }));

        // then
        expect(mocks.startEditing).toHaveBeenCalledExactlyOnceWith(message);
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
    });

    it("opens on a long press in a live stream chat, where the hover bar is hardest to hit", () => {
        // given
        vi.useFakeTimers();
        renderPanel([ownMessage()]);

        // when
        fireEvent.pointerDown(bubbleOf("m1"), { pointerType: "touch", clientX: 40, clientY: 60 });
        act(() => {
            vi.advanceTimersByTime(450);
        });

        // then
        expect(menuLabels()).toEqual(["Reply", "Edit message"]);
    });

    it("closes a stream chat menu on Escape", async () => {
        // given
        const user = userEvent.setup();
        renderPanel([ownMessage()]);
        fireEvent.contextMenu(bubbleOf("m1"));
        expect(screen.getByRole("menu", { name: "Message actions" })).toBeInTheDocument();

        // when
        await user.keyboard("{Escape}");

        // then
        expect(screen.queryByRole("menu")).not.toBeInTheDocument();
        expect(mocks.startEditing).not.toHaveBeenCalled();
    });
});
