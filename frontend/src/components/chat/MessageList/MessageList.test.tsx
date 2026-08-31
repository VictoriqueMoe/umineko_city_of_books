import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage, makeChatRoom, makeDmRoom, makePublicUser, makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, ChatRoom, User, UserProfile } from "../../../types/api";
import { MessageList, type MessageListProps } from "./MessageList";

const mocks = vi.hoisted(() => ({ useBlockedUserIds: vi.fn(), replyHandlers: [] as unknown[] }));

vi.mock("../../../hooks/useBlockedUserIds", () => ({ useBlockedUserIds: mocks.useBlockedUserIds }));

vi.mock("../MessageBubble/MessageBubble", () => ({
    MessageBubble: ({
        message,
        isOwn,
        senderBlocked,
        highlighted,
        notifiesViewer,
        seenLabel,
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
        seenLabel?: string | null;
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
                data-seen={seenLabel ?? ""}
                data-editing={String(editing)}
                data-can-pin={String(canPin)}
                data-moderate={String(canModerate)}
                data-can-react={String(canReact)}
                data-can-edit={String(canEdit)}
                data-staff={String(senderIsStaff)}
                data-has-pin-handler={String(Boolean(onPinToggle))}
                data-has-reaction-handler={String(Boolean(onReactionToggle))}
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
    return makeChatMessage({
        sender: makeSender(),
        body: "without love it cannot be seen",
        created_at: "2026-08-01T10:00:00Z",
        ...overrides,
    });
}

function makeGroupRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({
        name: "Rokkenjima",
        viewer_role: "member",
        created_at: "2026-07-01T00:00:00Z",
        ...overrides,
    });
}

function makePairRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
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
    highlightedMessageId?: string | null;
    editingMessageId?: string | null;
    viewerTimedOut?: boolean;
    readReceipts?: Record<string, Record<string, string>>;
    matchesViewerMention?: ((body: string) => boolean) | null;
    onReply?: MessageListProps["onReply"];
    onStartEditing?: MessageListProps["onStartEditing"];
    onCancelEditing?: MessageListProps["onCancelEditing"];
    onToggleReaction?: MessageListProps["onToggleReaction"];
    onTogglePin?: MessageListProps["onTogglePin"];
}

function makeProps(options: ListOptions = {}): MessageListProps {
    return {
        viewer: options.user ?? viewer,
        room: options.room ?? makeGroupRoom(),
        messages: options.messages ?? [],
        hasMore: options.hasMore ?? false,
        loadingMore: options.loadingMore ?? false,
        highlightedMessageId: options.highlightedMessageId ?? null,
        editingMessageId: options.editingMessageId ?? null,
        viewerTimedOut: options.viewerTimedOut ?? false,
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
        onToggleReaction: options.onToggleReaction ?? vi.fn(),
        onTogglePin: options.onTogglePin,
        onDelete: vi.fn(),
        onEdit: vi.fn(),
        classes,
    };
}

function renderList(options: ListOptions = {}) {
    const props = makeProps(options);
    const result = renderWithProviders(<MessageList {...props} />);

    return { ...result, props };
}

interface RoomKind {
    name: string;
    makeRoom: (overrides?: Partial<ChatRoom>) => ChatRoom;
}

const roomKinds: RoomKind[] = [
    { name: "a private thread", makeRoom: makePairRoom },
    { name: "a group room", makeRoom: makeGroupRoom },
];

beforeEach(() => {
    mocks.useBlockedUserIds.mockReturnValue(new Set<string>());
    mocks.replyHandlers.length = 0;
});

describe("MessageList, the surface both room kinds share", () => {
    for (const kind of roomKinds) {
        it(`greets the viewer when nobody has spoken in ${kind.name} yet`, () => {
            // given
            const messages: ChatMessage[] = [];

            // when
            renderList({ room: kind.makeRoom(), messages, hasMore: false });

            // then
            expect(screen.getByText("No messages yet. Say hello!")).toBeInTheDocument();
        });

        it(`withholds the empty greeting in ${kind.name} while older messages are still unfetched`, () => {
            // given
            const hasMore = true;

            // when
            renderList({ room: kind.makeRoom(), messages: [], hasMore });

            // then
            expect(screen.queryByText("No messages yet. Say hello!")).not.toBeInTheDocument();
            expect(screen.getByText("Scroll up for more")).toBeInTheDocument();
        });

        it(`invites the viewer of ${kind.name} to scroll up while older messages are still on the server`, () => {
            // given
            const hasMore = true;

            // when
            renderList({ room: kind.makeRoom(), hasMore, messages: [makeMessage()] });

            // then
            expect(screen.getByText("Scroll up for more")).toBeInTheDocument();
        });

        it(`says ${kind.name} is fetching while older messages are on their way`, () => {
            // given
            const loadingMore = true;

            // when
            renderList({ room: kind.makeRoom(), hasMore: true, loadingMore, messages: [makeMessage()] });

            // then
            expect(screen.getByText("Loading older messages...")).toBeInTheDocument();
            expect(screen.queryByText("Scroll up for more")).not.toBeInTheDocument();
        });

        it(`drops the scroll hint once the whole of ${kind.name} is loaded`, () => {
            // given
            const hasMore = false;

            // when
            renderList({ room: kind.makeRoom(), hasMore, messages: [makeMessage()] });

            // then
            expect(screen.queryByText("Scroll up for more")).not.toBeInTheDocument();
            expect(screen.queryByText("Loading older messages...")).not.toBeInTheDocument();
        });

        it(`renders every message of ${kind.name} in the order it was given`, () => {
            // given
            const messages = [
                makeMessage({ id: "m1", body: "first" }),
                makeMessage({ id: "m2", body: "second" }),
                makeMessage({ id: "m3", body: "third" }),
            ];

            // when
            renderList({ room: kind.makeRoom(), messages });

            // then
            const bodies = screen.getAllByText(/^(first|second|third)$/).map(node => node.textContent);
            expect(bodies).toEqual(["first", "second", "third"]);
        });

        it(`marks the viewer's own messages in ${kind.name} as theirs and the other person's as not`, () => {
            // given
            const messages = [
                makeMessage({ id: "mine", sender: makeSender({ id: "u1" }) }),
                makeMessage({ id: "theirs" }),
            ];

            // when
            renderList({ room: kind.makeRoom(), messages });

            // then
            expect(screen.getByTestId("bubble-mine")).toHaveAttribute("data-own", "true");
            expect(screen.getByTestId("bubble-theirs")).toHaveAttribute("data-own", "false");
        });

        it(`masks the person the viewer blocked in ${kind.name} without touching anyone else`, () => {
            // given
            mocks.useBlockedUserIds.mockReturnValue(new Set(["u2"]));
            const messages = [
                makeMessage({ id: "blocked" }),
                makeMessage({ id: "fine", sender: makeSender({ id: "u3" }) }),
            ];

            // when
            renderList({ room: kind.makeRoom(), messages });

            // then
            expect(screen.getByTestId("bubble-blocked")).toHaveAttribute("data-blocked", "true");
            expect(screen.getByTestId("bubble-fine")).toHaveAttribute("data-blocked", "false");
        });

        it(`flags a reply to the viewer of ${kind.name} and a message that mentions them`, () => {
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
            renderList({ room: kind.makeRoom(), matchesViewerMention, messages });

            // then
            expect(screen.getByTestId("bubble-reply")).toHaveAttribute("data-notifies", "true");
            expect(screen.getByTestId("bubble-mention")).toHaveAttribute("data-notifies", "true");
            expect(screen.getByTestId("bubble-plain")).toHaveAttribute("data-notifies", "false");
        });

        it(`leaves an ordinary message in ${kind.name} unflagged when no mention matcher exists`, () => {
            // given
            const matchesViewerMention = null;

            // when
            renderList({ room: kind.makeRoom(), matchesViewerMention, messages: [makeMessage({ id: "m1" })] });

            // then
            expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-notifies", "false");
        });

        it(`highlights only the message the viewer of ${kind.name} jumped to`, () => {
            // given
            const highlightedMessageId = "m2";

            // when
            renderList({
                room: kind.makeRoom(),
                highlightedMessageId,
                messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })],
            });

            // then
            expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-highlighted", "false");
            expect(screen.getByTestId("bubble-m2")).toHaveAttribute("data-highlighted", "true");
        });

        it(`quotes a short body whole when the viewer of ${kind.name} replies`, async () => {
            // given
            const onReply = vi.fn();
            const user = userEvent.setup();
            renderList({
                room: kind.makeRoom(),
                onReply,
                messages: [makeMessage({ id: "m1", body: "a short claim" })],
            });

            // when
            await user.click(screen.getByRole("button", { name: "reply to m1" }));

            // then
            expect(onReply).toHaveBeenCalledWith({
                id: "m1",
                senderName: "Battler",
                bodyPreview: "a short claim",
            });
        });

        it(`truncates a long body when the viewer of ${kind.name} replies to it`, async () => {
            // given
            const onReply = vi.fn();
            const body = "x".repeat(120);
            const user = userEvent.setup();
            renderList({ room: kind.makeRoom(), onReply, messages: [makeMessage({ id: "m1", body })] });

            // when
            await user.click(screen.getByRole("button", { name: "reply to m1" }));

            // then
            expect(onReply).toHaveBeenCalledWith({
                id: "m1",
                senderName: "Battler",
                bodyPreview: `${"x".repeat(80)}...`,
            });
        });

        it(`opens the editor in ${kind.name} for the message the viewer chose`, async () => {
            // given
            const onStartEditing = vi.fn();
            const message = makeMessage({ id: "m1" });
            const user = userEvent.setup();
            renderList({ room: kind.makeRoom(), onStartEditing, messages: [message] });

            // when
            await user.click(screen.getByRole("button", { name: "edit m1" }));

            // then
            expect(onStartEditing).toHaveBeenCalledExactlyOnceWith(message);
        });

        it(`closes the editor in ${kind.name} when the viewer abandons the edit`, async () => {
            // given
            const onCancelEditing = vi.fn();
            const user = userEvent.setup();
            renderList({ room: kind.makeRoom(), onCancelEditing, messages: [makeMessage({ id: "m1" })] });

            // when
            await user.click(screen.getByRole("button", { name: "cancel m1" }));

            // then
            expect(onCancelEditing).toHaveBeenCalledOnce();
        });

        it(`puts only the chosen message of ${kind.name} into edit mode`, () => {
            // given
            const editingMessageId = "m2";

            // when
            renderList({
                room: kind.makeRoom(),
                editingMessageId,
                messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })],
            });

            // then
            expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-editing", "false");
            expect(screen.getByTestId("bubble-m2")).toHaveAttribute("data-editing", "true");
        });

        it(`marks a message written by staff in ${kind.name} so it stays protected`, () => {
            // given
            const messages = [
                makeMessage({ id: "staff", sender: makeSender({ role: "admin" }) }),
                makeMessage({ id: "member" }),
            ];

            // when
            renderList({ room: kind.makeRoom(), messages });

            // then
            expect(screen.getByTestId("bubble-staff")).toHaveAttribute("data-staff", "true");
            expect(screen.getByTestId("bubble-member")).toHaveAttribute("data-staff", "false");
        });

        it(`hands every message in ${kind.name} a reaction handler so the control is reachable`, () => {
            // given
            const messages = [makeMessage({ id: "m1" })];

            // when
            renderList({ room: kind.makeRoom(), messages });

            // then
            expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-has-reaction-handler", "true");
        });

        it(`sends a reaction from ${kind.name} up with the emoji that was picked`, async () => {
            // given
            const onToggleReaction = vi.fn();
            const message = makeMessage({ id: "m1" });
            const user = userEvent.setup();
            renderList({ room: kind.makeRoom(), onToggleReaction, messages: [message] });

            // when
            await user.click(screen.getByRole("button", { name: "react to m1" }));

            // then
            expect(onToggleReaction).toHaveBeenCalledWith(message, "❤");
        });

        it(`uses the newest reaction handler of ${kind.name} after the page re-renders`, async () => {
            // given
            const stale = vi.fn();
            const fresh = vi.fn();
            const room = kind.makeRoom();
            const message = makeMessage({ id: "m1" });
            const user = userEvent.setup();
            const { rerender } = renderList({ room, onToggleReaction: stale, messages: [message] });

            // when
            rerender(<MessageList {...makeProps({ room, onToggleReaction: fresh, messages: [message] })} />);
            await user.click(screen.getByRole("button", { name: "react to m1" }));

            // then
            expect(fresh).toHaveBeenCalledWith(message, "❤");
            expect(stale).not.toHaveBeenCalled();
        });

        it(`keeps one reply handler for ${kind.name} across a re-render so a memoised bubble is not invalidated`, () => {
            // given
            const props = makeProps({ room: kind.makeRoom(), messages: [makeMessage({ id: "m1" })] });

            // when
            const { rerender } = renderWithProviders(<MessageList {...props} />);
            rerender(<MessageList {...props} hasMore={true} />);

            // then
            expect(mocks.replyHandlers).toHaveLength(2);
            expect(mocks.replyHandlers[0]).toBe(mocks.replyHandlers[1]);
        });

        it(`skips the whole of ${kind.name} when nothing it renders has changed`, () => {
            // given
            const props = makeProps({ room: kind.makeRoom(), messages: [makeMessage({ id: "m1" })] });

            // when
            const { rerender } = renderWithProviders(<MessageList {...props} />);
            rerender(<MessageList {...props} />);

            // then
            expect(mocks.replyHandlers).toHaveLength(1);
        });

        it(`lets a member of ${kind.name} react and edit while they are in good standing`, () => {
            // given
            const viewerTimedOut = false;

            // when
            renderList({ room: kind.makeRoom(), viewerTimedOut, messages: [makeMessage({ id: "m1" })] });

            // then
            const bubble = screen.getByTestId("bubble-m1");
            expect(bubble).toHaveAttribute("data-can-react", "true");
            expect(bubble).toHaveAttribute("data-can-edit", "true");
        });

        it(`takes reacting and editing in ${kind.name} away from a timed out member`, () => {
            // given
            const viewerTimedOut = true;

            // when
            renderList({ room: kind.makeRoom(), viewerTimedOut, messages: [makeMessage({ id: "m1" })] });

            // then
            const bubble = screen.getByTestId("bubble-m1");
            expect(bubble).toHaveAttribute("data-can-react", "false");
            expect(bubble).toHaveAttribute("data-can-edit", "false");
        });
    }
});

describe("MessageList, seen labels stay inside private threads", () => {
    it("labels only the viewer's last message in a private thread as seen", () => {
        // given
        const messages = [
            makeMessage({ id: "mine-early", sender: makeSender({ id: "u1" }), created_at: "2026-08-01T10:00:00Z" }),
            makeMessage({ id: "mine-late", sender: makeSender({ id: "u1" }), created_at: "2026-08-01T10:05:00Z" }),
        ];
        const readReceipts = { "room-1": { u2: "2026-08-01T11:00:00Z" } };

        // when
        renderList({ room: makePairRoom(), messages, readReceipts });

        // then
        expect(screen.getByTestId("bubble-mine-early")).toHaveAttribute("data-seen", "");
        expect(screen.getByTestId("bubble-mine-late").getAttribute("data-seen")).toMatch(/^seen /);
    });

    it("never labels the other person's message in a private thread as seen", () => {
        // given
        const messages = [makeMessage({ id: "theirs" })];
        const readReceipts = { "room-1": { u2: "2026-08-01T11:00:00Z" } };

        // when
        renderList({ room: makePairRoom(), messages, readReceipts });

        // then
        expect(screen.getByTestId("bubble-theirs")).toHaveAttribute("data-seen", "");
    });

    it("never labels a group room message as seen, whatever receipts arrive", () => {
        // given
        const messages = [
            makeMessage({ id: "mine", sender: makeSender({ id: "u1" }), created_at: "2026-08-01T10:00:00Z" }),
        ];
        const room = makeGroupRoom({ members: [makePublicUser(), makeSender()] });
        const readReceipts = { "room-1": { u2: "2026-08-01T11:00:00Z" } };

        // when
        renderList({ room, messages, readReceipts });

        // then
        expect(screen.getByTestId("bubble-mine")).toHaveAttribute("data-seen", "");
    });
});

describe("MessageList, moderation and pinning follow the room's capabilities", () => {
    it("denies pinning and moderation to an ordinary group member even where a pin handler exists", () => {
        // given
        const room = makeGroupRoom({ viewer_role: "member" });

        // when
        renderList({ room, onTogglePin: vi.fn(), messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "false");
        expect(bubble).toHaveAttribute("data-moderate", "false");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "false");
    });

    it("lets the group host pin and moderate", () => {
        // given
        const room = makeGroupRoom({ viewer_role: "host" });

        // when
        renderList({ room, onTogglePin: vi.fn(), messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "true");
        expect(bubble).toHaveAttribute("data-moderate", "true");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "true");
    });

    it("lets site staff moderate a group room they do not host", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice", role: "moderator" });

        // when
        renderList({
            user,
            room: makeGroupRoom({ viewer_role: "member" }),
            messages: [makeMessage({ id: "m1" })],
        });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-moderate", "true");
    });

    it("pins through the handler it was given", async () => {
        // given
        const onTogglePin = vi.fn();
        const message = makeMessage({ id: "m1" });
        const user = userEvent.setup();
        renderList({ room: makeGroupRoom({ viewer_role: "host" }), onTogglePin, messages: [message] });

        // when
        await user.click(screen.getByRole("button", { name: "pin m1" }));

        // then
        expect(onTogglePin).toHaveBeenCalledExactlyOnceWith(message);
    });

    it("denies moderation to an ordinary participant of a private thread", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice" });

        // when
        renderList({ user, room: makePairRoom(), messages: [makeMessage({ id: "m1" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-moderate", "false");
    });

    it("lets site staff moderate a private thread", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice", role: "moderator" });

        // when
        renderList({ user, room: makePairRoom(), messages: [makeMessage({ id: "m1" })] });

        // then
        expect(screen.getByTestId("bubble-m1")).toHaveAttribute("data-moderate", "true");
    });

    it("withholds the pin control from a participant of a private thread while no page wires one", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice" });

        // when
        renderList({ user, room: makePairRoom(), messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "false");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "false");
    });

    it("gives site staff no pin control inside a private thread while no page wires one", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice", role: "moderator" });

        // when
        renderList({ user, room: makePairRoom(), messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "false");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "false");
    });

    it("offers a participant of a private thread the pin their capability allows once a page wires one", () => {
        // given
        const user = makeUser({ id: "u1", username: "beatrice" });

        // when
        renderList({ user, room: makePairRoom(), onTogglePin: vi.fn(), messages: [makeMessage({ id: "m1" })] });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "true");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "true");
    });

    it("keeps the pin away from somebody who is not in the private thread at all", () => {
        // given
        const user = makeUser({ id: "u9", username: "erika" });

        // when
        renderList({
            user,
            room: makePairRoom({ is_member: false }),
            onTogglePin: vi.fn(),
            messages: [makeMessage({ id: "m1" })],
        });

        // then
        const bubble = screen.getByTestId("bubble-m1");
        expect(bubble).toHaveAttribute("data-can-pin", "false");
        expect(bubble).toHaveAttribute("data-has-pin-handler", "false");
    });
});
