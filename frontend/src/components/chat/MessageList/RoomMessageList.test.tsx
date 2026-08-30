import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, ChatRoom, User, UserProfile } from "../../../types/api";
import { RoomMessageList, type RoomMessageListProps } from "./RoomMessageList";

const mocks = vi.hoisted(() => ({ useBlockedUserIds: vi.fn(), replyHandlers: [] as unknown[] }));

vi.mock("../../../hooks/useBlockedUserIds", () => ({ useBlockedUserIds: mocks.useBlockedUserIds }));

vi.mock("../MessageBubble/MessageBubble", () => ({
    MessageBubble: ({
        message,
        isOwn,
        senderBlocked,
        highlighted,
        notifiesViewer,
        editing,
        canPin,
        canModerate,
        canReact,
        canEdit,
        senderIsStaff,
        onReply,
        onPinToggle,
        onReactionToggle,
        onEditStart,
        onEditCancel,
    }: {
        message: ChatMessage;
        isOwn: boolean;
        senderBlocked?: boolean;
        highlighted?: boolean;
        notifiesViewer?: boolean;
        editing?: boolean;
        canPin?: boolean;
        canModerate?: boolean;
        canReact?: boolean;
        canEdit?: boolean;
        senderIsStaff?: boolean;
        onReply?: (msg: ChatMessage) => void;
        onPinToggle?: (msg: ChatMessage) => void;
        onReactionToggle?: (msg: ChatMessage, emoji: string) => void;
        onEditStart?: (msg: ChatMessage) => void;
        onEditCancel?: () => void;
    }) => {
        mocks.replyHandlers.push(onReply);

        return (
            <div
                data-testid={`bubble-${message.id}`}
                data-own={String(isOwn)}
                data-blocked={String(senderBlocked)}
                data-highlighted={String(highlighted)}
                data-notifies={String(notifiesViewer)}
                data-editing={String(editing)}
                data-can-pin={String(canPin)}
                data-moderate={String(canModerate)}
                data-can-react={String(canReact)}
                data-can-edit={String(canEdit)}
                data-staff={String(senderIsStaff)}
                data-has-pin-handler={String(Boolean(onPinToggle))}
            >
                <span>{message.body}</span>
                <button type="button" onClick={() => onReply?.(message)}>
                    reply to {message.id}
                </button>
                <button type="button" onClick={() => onReactionToggle?.(message, "❤")}>
                    react to {message.id}
                </button>
                <button type="button" onClick={() => onPinToggle?.(message)}>
                    pin {message.id}
                </button>
                <button type="button" onClick={() => onEditStart?.(message)}>
                    edit {message.id}
                </button>
                <button type="button" onClick={() => onEditCancel?.()}>
                    cancel {message.id}
                </button>
            </div>
        );
    },
}));

const classes = { messages: "messages", loadMoreBar: "load-more", empty: "empty" };

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeSender(overrides: Partial<User> = {}): User {
    return { id: "u2", username: "battler", display_name: "Battler", ...overrides };
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: makeSender(),
        body: "the golden truth",
        is_system: false,
        created_at: "2026-08-01T10:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Rokkenjima",
        description: "",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_role: "member",
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [],
        created_at: "2026-07-01T00:00:00Z",
        ...overrides,
    };
}

interface ListOptions {
    user?: UserProfile;
    room?: ChatRoom;
    messages?: ChatMessage[];
    hasMore?: boolean;
    loadingMore?: boolean;
    highlightedMessageId?: string | null;
    matchesViewerMention?: ((body: string) => boolean) | null;
    viewerTimedOut?: boolean;
    editingMessageId?: string | null;
    onReply?: RoomMessageListProps["onReply"];
    onStartEditing?: RoomMessageListProps["onStartEditing"];
    onCancelEditing?: RoomMessageListProps["onCancelEditing"];
    onToggleReaction?: RoomMessageListProps["onToggleReaction"];
    onTogglePin?: RoomMessageListProps["onTogglePin"];
}

function makeProps(options: ListOptions = {}): RoomMessageListProps {
    return {
        viewer: options.user ?? viewer,
        room: options.room ?? makeRoom(),
        messages: options.messages ?? [],
        hasMore: options.hasMore ?? false,
        loadingMore: options.loadingMore ?? false,
        highlightedMessageId: options.highlightedMessageId ?? null,
        editingMessageId: options.editingMessageId ?? null,
        viewerTimedOut: options.viewerTimedOut ?? false,
        matchesViewerMention: options.matchesViewerMention ?? null,
        containerRef: { current: null },
        contentRef: { current: null },
        endRef: { current: null },
        onScroll: vi.fn(),
        onLightbox: vi.fn(),
        onReply: options.onReply ?? vi.fn(),
        onStartEditing: options.onStartEditing ?? vi.fn(),
        onCancelEditing: options.onCancelEditing ?? vi.fn(),
        onToggleReaction: options.onToggleReaction ?? vi.fn(),
        onTogglePin: options.onTogglePin ?? vi.fn(),
        onDelete: vi.fn(),
        onEdit: vi.fn(),
        classes,
    };
}

function renderList(options: ListOptions = {}) {
    const props = makeProps(options);
    const result = renderWithProviders(<RoomMessageList {...props} />);

    return { ...result, props };
}

beforeEach(() => {
    mocks.useBlockedUserIds.mockReturnValue(new Set<string>());
    mocks.replyHandlers.length = 0;
});

describe("RoomMessageList", () => {
    it("greets the viewer when nobody has spoken in the room yet", () => {
        // given
        const messages: ChatMessage[] = [];

        // when
        renderList({ messages, hasMore: false });

        // then
        expect(screen.getByText("No messages yet. Say hello!")).toBeInTheDocument();
    });

    it("withholds the empty greeting while older messages are still unfetched", () => {
        // given
        const hasMore = true;

        // when
        renderList({ messages: [], hasMore });

        // then
        expect(screen.queryByText("No messages yet. Say hello!")).not.toBeInTheDocument();
        expect(screen.getByText("Scroll up for more")).toBeInTheDocument();
    });

    it("says it is fetching while older messages are on their way", () => {
        // given
        const loadingMore = true;

        // when
        renderList({ messages: [makeMessage()], hasMore: true, loadingMore });

        // then
        expect(screen.getByText("Loading older messages...")).toBeInTheDocument();
        expect(screen.queryByText("Scroll up for more")).not.toBeInTheDocument();
    });

    it("renders every message in the order it was given", () => {
        // given
        const messages = [makeMessage({ id: "m1", body: "first" }), makeMessage({ id: "m2", body: "second" })];

        // when
        renderList({ messages });

        // then
        const bodies = screen.getAllByText(/^(first|second)$/).map(node => node.textContent);
        expect(bodies).toEqual(["first", "second"]);
    });

    it("marks the viewer's own message as theirs", () => {
        // given
        const messages = [makeMessage({ id: "mine", sender: makeSender({ id: "u1" }) }), makeMessage({ id: "theirs" })];

        // when
        renderList({ messages });

        // then
        expect(screen.getByTestId("bubble-mine")).toHaveAttribute("data-own", "true");
        expect(screen.getByTestId("bubble-theirs")).toHaveAttribute("data-own", "false");
    });

    it("marks a message from someone the viewer blocked", () => {
        // given
        mocks.useBlockedUserIds.mockReturnValue(new Set(["u2"]));
        const messages = [
            makeMessage({ id: "blocked" }),
            makeMessage({ id: "fine", sender: makeSender({ id: "u3" }) }),
        ];

        // when
        renderList({ messages });

        // then
        expect(screen.getByTestId("bubble-blocked")).toHaveAttribute("data-blocked", "true");
        expect(screen.getByTestId("bubble-fine")).toHaveAttribute("data-blocked", "false");
    });

    it("highlights only the message the viewer jumped to", () => {
        // given
        const highlightedMessageId = "m2";

        // when
        renderList({ highlightedMessageId, messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-highlighted", "false");
        expect(screen.getByTestId("bubble-m2")).toHaveAttribute("data-highlighted", "true");
    });

    it("flags a reply to the viewer and a message that mentions them", () => {
        // given
        const matchesViewerMention = (body: string) => body.includes("@beatrice");
        const messages = [
            makeMessage({
                id: "reply",
                reply_to: { id: "m0", sender_id: "u1", sender_name: "Beatrice", body_preview: "earlier" },
            }),
            makeMessage({ id: "mention", body: "@beatrice explain" }),
            makeMessage({ id: "plain" }),
        ];

        // when
        renderList({ matchesViewerMention, messages });

        // then
        expect(screen.getByTestId("bubble-reply")).toHaveAttribute("data-notifies", "true");
        expect(screen.getByTestId("bubble-mention")).toHaveAttribute("data-notifies", "true");
        expect(screen.getByTestId("bubble-plain")).toHaveAttribute("data-notifies", "false");
    });

    it("denies pinning and moderation to an ordinary member", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        renderList({ room, messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "false");
        expect(bubble).toHaveAttribute("data-moderate", "false");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "false");
    });

    it("lets the room host pin and moderate", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        renderList({ room, messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "true");
        expect(bubble).toHaveAttribute("data-moderate", "true");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "true");
    });

    it("lets site staff pin and moderate a room they do not host", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice", role: "moderator" });

        // when
        renderList({ user, room: makeRoom({ viewer_role: "member" }), messages: [makeMessage({ id: "m1" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-moderate", "true");
    });

    it("lets a member react and edit while they are in good standing", () => {
        // given
        const viewerTimedOut = false;

        // when
        renderList({ viewerTimedOut, messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-react", "true");
        expect(bubble).toHaveAttribute("data-can-edit", "true");
    });

    it("takes reacting and editing away from a timed out member", () => {
        // given
        const viewerTimedOut = true;

        // when
        renderList({ viewerTimedOut, messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-react", "false");
        expect(bubble).toHaveAttribute("data-can-edit", "false");
    });

    it("marks a message written by staff so it stays protected", () => {
        // given
        const messages = [
            makeMessage({ id: "staff", sender: makeSender({ role: "admin" }) }),
            makeMessage({ id: "member" }),
        ];

        // when
        renderList({ messages });

        // then
        expect(screen.getByTestId("bubble-staff")).toHaveAttribute("data-staff", "true");
        expect(screen.getByTestId("bubble-member")).toHaveAttribute("data-staff", "false");
    });

    it("quotes a short body whole when the viewer replies", async () => {
        // given
        const onReply = vi.fn();
        const user = userEvent.setup();
        renderList({ onReply, messages: [makeMessage({ id: "m1", body: "a short claim" })] });

        // when
        await user.click(screen.getByRole("button", { name: "reply to m1" }));

        // then
        expect(onReply).toHaveBeenCalledWith({
            id: "m1",
            senderName: "Battler",
            bodyPreview: "a short claim",
        });
    });

    it("truncates a long body when the viewer replies to it", async () => {
        // given
        const onReply = vi.fn();
        const body = "x".repeat(120);
        const user = userEvent.setup();
        renderList({ onReply, messages: [makeMessage({ id: "m1", body })] });

        // when
        await user.click(screen.getByRole("button", { name: "reply to m1" }));

        // then
        expect(onReply).toHaveBeenCalledWith({
            id: "m1",
            senderName: "Battler",
            bodyPreview: `${"x".repeat(80)}...`,
        });
    });

    it("opens and closes the editor through the handlers it was given", async () => {
        // given
        const onStartEditing = vi.fn();
        const onCancelEditing = vi.fn();
        const message = makeMessage({ id: "m1" });
        const user = userEvent.setup();
        renderList({ onStartEditing, onCancelEditing, messages: [message] });

        // when
        await user.click(screen.getByRole("button", { name: "edit m1" }));
        await user.click(screen.getByRole("button", { name: "cancel m1" }));

        // then
        expect(onStartEditing).toHaveBeenCalledExactlyOnceWith(message);
        expect(onCancelEditing).toHaveBeenCalledOnce();
    });

    it("puts only the chosen message into edit mode", () => {
        // given
        const editingMessageId = "m2";

        // when
        renderList({ editingMessageId, messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-editing", "false");
        expect(screen.getByTestId("bubble-m2")).toHaveAttribute("data-editing", "true");
    });

    it("sends a reaction up with the emoji that was picked", async () => {
        // given
        const onToggleReaction = vi.fn();
        const message = makeMessage({ id: "m1" });
        const user = userEvent.setup();
        renderList({ onToggleReaction, messages: [message] });

        // when
        await user.click(screen.getByRole("button", { name: "react to m1" }));

        // then
        expect(onToggleReaction).toHaveBeenCalledWith(message, "❤");
    });

    it("uses the newest reaction handler after the page re-renders", async () => {
        // given
        const stale = vi.fn();
        const fresh = vi.fn();
        const message = makeMessage({ id: "m1" });
        const user = userEvent.setup();
        const { rerender } = renderList({ onToggleReaction: stale, messages: [message] });

        // when
        rerender(<RoomMessageList {...makeProps({ onToggleReaction: fresh, messages: [message] })} />);
        await user.click(screen.getByRole("button", { name: "react to m1" }));

        // then
        expect(fresh).toHaveBeenCalledWith(message, "❤");
        expect(stale).not.toHaveBeenCalled();
    });

    it("keeps one reply handler across a re-render so a memoised bubble is not invalidated", () => {
        // given
        const props = makeProps({ messages: [makeMessage({ id: "m1" })] });

        // when
        const { rerender } = renderWithProviders(<RoomMessageList {...props} />);
        rerender(<RoomMessageList {...props} hasMore={true} />);

        // then
        expect(mocks.replyHandlers).toHaveLength(2);
        expect(mocks.replyHandlers[0]).toBe(mocks.replyHandlers[1]);
    });

    it("skips the whole list when nothing it renders has changed", () => {
        // given
        const props = makeProps({ messages: [makeMessage({ id: "m1" })] });

        // when
        const { rerender } = renderWithProviders(<RoomMessageList {...props} />);
        rerender(<RoomMessageList {...props} />);

        // then
        expect(mocks.replyHandlers).toHaveLength(1);
    });

    it("pins through the handler it was given", async () => {
        // given
        const onTogglePin = vi.fn();
        const message = makeMessage({ id: "m1" });
        const user = userEvent.setup();
        renderList({ onTogglePin, room: makeRoom({ viewer_role: "host" }), messages: [message] });

        // when
        await user.click(screen.getByRole("button", { name: "pin m1" }));

        // then
        expect(onTogglePin).toHaveBeenCalledWith(message);
    });
});
