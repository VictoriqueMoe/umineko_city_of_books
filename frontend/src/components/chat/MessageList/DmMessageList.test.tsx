import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage, makeDmRoom, makePublicUser, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, ChatRoom, User, UserProfile } from "../../../types/api";
import { DmMessageList, type DmMessageListProps } from "./DmMessageList";

const spies = vi.hoisted(() => ({ replyHandlers: [] as unknown[] }));

vi.mock("../MessageBubble/MessageBubble", () => ({
    MessageBubble: ({
        message,
        isOwn,
        notifiesViewer,
        seenLabel,
        editing,
        canModerate,
        senderIsStaff,
        onReply,
        onEditStart,
        onEditCancel,
    }: {
        message: ChatMessage;
        isOwn: boolean;
        notifiesViewer?: boolean;
        seenLabel?: string | null;
        editing?: boolean;
        canModerate?: boolean;
        senderIsStaff?: boolean;
        onReply?: (msg: ChatMessage) => void;
        onEditStart?: (msg: ChatMessage) => void;
        onEditCancel?: () => void;
    }) => {
        spies.replyHandlers.push(onReply);

        return (
            <div
                data-testid={`bubble-${message.id}`}
                data-own={String(isOwn)}
                data-notifies={String(notifiesViewer)}
                data-seen={seenLabel ?? ""}
                data-editing={String(editing)}
                data-moderate={String(canModerate)}
                data-staff={String(senderIsStaff)}
            >
                <span>{message.body}</span>
                <button type="button" onClick={() => onReply?.(message)}>
                    reply to {message.id}
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

const classes = { messages: "messages", loadMoreBar: "load-more" };

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeSender(overrides: Partial<User> = {}): User {
    return { id: "u2", username: "battler", display_name: "Battler", ...overrides };
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        sender: makeSender(),
        body: "without love it cannot be seen",
        created_at: "2026-08-01T10:00:00Z",
        ...overrides,
    });
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({
        members: [makePublicUser(), makeSender()],
        created_at: "2026-07-01T00:00:00Z",
        ...overrides,
    });
}

interface ListOptions {
    user?: UserProfile;
    room?: ChatRoom;
    messages?: ChatMessage[];
    hasMore?: boolean;
    loadingMore?: boolean;
    editingMessageId?: string | null;
    readReceipts?: Record<string, Record<string, string>>;
    matchesViewerMention?: ((body: string) => boolean) | null;
    onReply?: DmMessageListProps["onReply"];
    onStartEditing?: DmMessageListProps["onStartEditing"];
    onCancelEditing?: DmMessageListProps["onCancelEditing"];
}

function makeProps(options: ListOptions = {}): DmMessageListProps {
    return {
        viewer: options.user ?? viewer,
        room: options.room ?? makeRoom(),
        messages: options.messages ?? [],
        hasMore: options.hasMore ?? false,
        loadingMore: options.loadingMore ?? false,
        editingMessageId: options.editingMessageId ?? null,
        readReceipts: options.readReceipts ?? {},
        matchesViewerMention: options.matchesViewerMention ?? null,
        containerRef: { current: null },
        contentRef: { current: null },
        endRef: { current: null },
        onScroll: vi.fn(),
        onLightbox: vi.fn(),
        onReply: options.onReply ?? vi.fn(),
        onStartEditing: options.onStartEditing ?? vi.fn(),
        onCancelEditing: options.onCancelEditing ?? vi.fn(),
        onDelete: vi.fn(),
        onEdit: vi.fn(),
        classes,
    };
}

function renderList(options: ListOptions = {}) {
    const props = makeProps(options);
    const result = renderWithProviders(<DmMessageList {...props} />);

    return { ...result, props };
}

beforeEach(() => {
    spies.replyHandlers.length = 0;
});

describe("DmMessageList", () => {
    it("renders every message in the conversation in the order it was given", () => {
        // given
        const messages = [
            makeMessage({ id: "m1", body: "first" }),
            makeMessage({ id: "m2", body: "second" }),
            makeMessage({ id: "m3", body: "third" }),
        ];

        // when
        renderList({ messages });

        // then
        const bodies = screen.getAllByText(/^(first|second|third)$/).map(node => node.textContent);
        expect(bodies).toEqual(["first", "second", "third"]);
    });

    it("invites the viewer to scroll up while older messages are still on the server", () => {
        // given
        const hasMore = true;

        // when
        renderList({ hasMore, messages: [makeMessage()] });

        // then
        expect(screen.getByText("Scroll up for more")).toBeInTheDocument();
    });

    it("says it is fetching while older messages are on their way", () => {
        // given
        const loadingMore = true;

        // when
        renderList({ hasMore: true, loadingMore, messages: [makeMessage()] });

        // then
        expect(screen.getByText("Loading older messages...")).toBeInTheDocument();
        expect(screen.queryByText("Scroll up for more")).not.toBeInTheDocument();
    });

    it("drops the scroll hint once the whole conversation is loaded", () => {
        // given
        const hasMore = false;

        // when
        renderList({ hasMore, messages: [makeMessage()] });

        // then
        expect(screen.queryByText("Scroll up for more")).not.toBeInTheDocument();
        expect(screen.queryByText("Loading older messages...")).not.toBeInTheDocument();
    });

    it("marks the viewer's own messages as theirs and the other person's as not", () => {
        // given
        const messages = [makeMessage({ id: "mine", sender: makeSender({ id: "u1" }) }), makeMessage({ id: "theirs" })];

        // when
        renderList({ messages });

        // then
        expect(screen.getByTestId("bubble-mine")).toHaveAttribute("data-own", "true");
        expect(screen.getByTestId("bubble-theirs")).toHaveAttribute("data-own", "false");
    });

    it("flags a reply to the viewer as something worth their attention", () => {
        // given
        const messages = [
            makeMessage({
                id: "reply",
                reply_to: { id: "m0", sender_id: "u1", sender_name: "Beatrice", body_preview: "earlier" },
            }),
        ];

        // when
        renderList({ messages });

        // then
        expect(screen.getByTestId("bubble-reply")).toHaveAttribute("data-notifies", "true");
    });

    it("flags a message that mentions the viewer", () => {
        // given
        const matchesViewerMention = (body: string) => body.includes("@beatrice");

        // when
        renderList({ matchesViewerMention, messages: [makeMessage({ id: "m1", body: "@beatrice explain" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-notifies", "true");
    });

    it("leaves an ordinary message unflagged when no mention matcher exists", () => {
        // given
        const matchesViewerMention = null;

        // when
        renderList({ matchesViewerMention, messages: [makeMessage({ id: "m1" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-notifies", "false");
    });

    it("labels only the viewer's last message as seen", () => {
        // given
        const messages = [
            makeMessage({ id: "mine-early", sender: makeSender({ id: "u1" }), created_at: "2026-08-01T10:00:00Z" }),
            makeMessage({ id: "mine-late", sender: makeSender({ id: "u1" }), created_at: "2026-08-01T10:05:00Z" }),
        ];
        const readReceipts = { "room-1": { u2: "2026-08-01T11:00:00Z" } };

        // when
        renderList({ messages, readReceipts });

        // then
        expect(screen.getByTestId("bubble-mine-early")).toHaveAttribute("data-seen", "");
        expect(screen.getByTestId("bubble-mine-late").getAttribute("data-seen")).toMatch(/^seen /);
    });

    it("never labels the other person's message as seen", () => {
        // given
        const messages = [makeMessage({ id: "theirs" })];
        const readReceipts = { "room-1": { u2: "2026-08-01T11:00:00Z" } };

        // when
        renderList({ messages, readReceipts });

        // then
        expect(screen.getByTestId("bubble-theirs")).toHaveAttribute("data-seen", "");
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

    it("keeps one reply handler across a re-render so a memoised bubble is not invalidated", () => {
        // given
        const props = makeProps({ messages: [makeMessage({ id: "m1" })] });

        // when
        const { rerender } = renderWithProviders(<DmMessageList {...props} />);
        rerender(<DmMessageList {...props} hasMore={true} />);

        // then
        expect(spies.replyHandlers).toHaveLength(2);
        expect(spies.replyHandlers[0]).toBe(spies.replyHandlers[1]);
    });

    it("skips the whole list when nothing it renders has changed", () => {
        // given
        const props = makeProps({ messages: [makeMessage({ id: "m1" })] });

        // when
        const { rerender } = renderWithProviders(<DmMessageList {...props} />);
        rerender(<DmMessageList {...props} />);

        // then
        expect(spies.replyHandlers).toHaveLength(1);
    });

    it("opens the editor for the message the viewer chose", async () => {
        // given
        const onStartEditing = vi.fn();
        const message = makeMessage({ id: "m1" });
        const user = userEvent.setup();
        renderList({ onStartEditing, messages: [message] });

        // when
        await user.click(screen.getByRole("button", { name: "edit m1" }));

        // then
        expect(onStartEditing).toHaveBeenCalledWith(message);
    });

    it("closes the editor when the viewer abandons the edit", async () => {
        // given
        const onCancelEditing = vi.fn();
        const user = userEvent.setup();
        renderList({ onCancelEditing, messages: [makeMessage({ id: "m1" })] });

        // when
        await user.click(screen.getByRole("button", { name: "cancel m1" }));

        // then
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

    it("denies moderation to an ordinary member", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice" });

        // when
        renderList({ user, messages: [makeMessage({ id: "m1" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-moderate", "false");
    });

    it("lets site staff moderate the conversation", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice", role: "moderator" });

        // when
        renderList({ user, messages: [makeMessage({ id: "m1" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-moderate", "true");
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
});
