import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { DmController } from "../../../hooks/useDmController";
import { makeUser } from "../../../test-utils/fixtures";
import { renderWithProviders } from "../../../test-utils/render";
import type { ChatMessage, ChatRoom, User } from "../../../types/api";
import { DmMessageList } from "./DmMessageList";

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
    }) => (
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
    ),
}));

const classes = { messages: "messages", loadMoreBar: "load-more" };

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeSender(overrides: Partial<User> = {}): User {
    return { id: "u2", username: "battler", display_name: "Battler", ...overrides };
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: makeSender(),
        body: "without love it cannot be seen",
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
        name: "",
        description: "",
        type: "dm",
        is_public: false,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [{ id: "u1", username: "beatrice", display_name: "Beatrice" }, makeSender()],
        created_at: "2026-07-01T00:00:00Z",
        ...overrides,
    };
}

function makeController(overrides: Partial<DmController> = {}): DmController {
    const base = {
        user: viewer,
        activeRoom: makeRoom(),
        messages: [],
        hasMore: false,
        loadingMore: false,
        messagesContainerRef: { current: null },
        messagesContentRef: { current: null },
        messagesEndRef: { current: null },
        handleDmScroll: vi.fn(),
        readReceipts: {},
        matchesViewerMention: null,
        setLightboxSrc: vi.fn(),
        setReplyingTo: vi.fn(),
        handleDeleteMessage: vi.fn(),
        handleEditMessage: vi.fn(),
        editingMessageId: null,
        setEditingMessageId: vi.fn(),
    };

    return { ...base, ...overrides } as unknown as DmController;
}

function renderList(overrides: Partial<DmController> = {}) {
    const controller = makeController(overrides);
    const result = renderWithProviders(<DmMessageList controller={controller} classes={classes} />);

    return { ...result, controller };
}

describe("DmMessageList", () => {
    it("renders nothing until the viewer is known", () => {
        // given
        const user = null;

        // when
        const { container } = renderList({ user, messages: [makeMessage()] });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders nothing until a conversation is open", () => {
        // given
        const activeRoom = undefined;

        // when
        const { container } = renderList({ activeRoom, messages: [makeMessage()] });

        // then
        expect(container).toBeEmptyDOMElement();
    });

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
        const setReplyingTo = vi.fn();
        const user = userEvent.setup();
        renderList({ setReplyingTo, messages: [makeMessage({ id: "m1", body: "a short claim" })] });

        // when
        await user.click(screen.getByRole("button", { name: "reply to m1" }));

        // then
        expect(setReplyingTo).toHaveBeenCalledWith({
            id: "m1",
            senderName: "Battler",
            bodyPreview: "a short claim",
        });
    });

    it("truncates a long body when the viewer replies to it", async () => {
        // given
        const setReplyingTo = vi.fn();
        const body = "x".repeat(120);
        const user = userEvent.setup();
        renderList({ setReplyingTo, messages: [makeMessage({ id: "m1", body })] });

        // when
        await user.click(screen.getByRole("button", { name: "reply to m1" }));

        // then
        expect(setReplyingTo).toHaveBeenCalledWith({
            id: "m1",
            senderName: "Battler",
            bodyPreview: `${"x".repeat(80)}...`,
        });
    });

    it("opens the editor for the message the viewer chose", async () => {
        // given
        const setEditingMessageId = vi.fn();
        const user = userEvent.setup();
        renderList({ setEditingMessageId, messages: [makeMessage({ id: "m1" })] });

        // when
        await user.click(screen.getByRole("button", { name: "edit m1" }));

        // then
        expect(setEditingMessageId).toHaveBeenCalledWith("m1");
    });

    it("closes the editor when the viewer abandons the edit", async () => {
        // given
        const setEditingMessageId = vi.fn();
        const user = userEvent.setup();
        renderList({ setEditingMessageId, messages: [makeMessage({ id: "m1" })] });

        // when
        await user.click(screen.getByRole("button", { name: "cancel m1" }));

        // then
        expect(setEditingMessageId).toHaveBeenCalledWith(null);
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
