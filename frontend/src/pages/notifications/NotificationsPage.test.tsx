import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { Notification } from "../../types/api";
import { NotificationsPage } from "./NotificationsPage";

const { useNotificationFeed, navigate } = vi.hoisted(() => ({
    useNotificationFeed: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../hooks/useNotificationFeed", () => ({ useNotificationFeed }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const beatrice = { id: "user-1", username: "beatrice", display_name: "Beatrice" };
const ange = { id: "user-2", username: "ange", display_name: "Ange" };

function makeNotification(overrides: Partial<Notification> = {}): Notification {
    return {
        id: 1,
        type: "post_liked",
        reference_id: "post-1",
        reference_type: "post",
        actor: beatrice,
        read: false,
        created_at: "2026-07-01T10:00:00Z",
        count: 1,
        ...overrides,
    };
}

const liked = makeNotification({ id: 1, type: "post_liked", read: false });
const followed = makeNotification({
    id: 2,
    type: "new_follower",
    reference_type: "user",
    reference_id: "user-2",
    actor: ange,
    read: true,
});
const artLiked = makeNotification({
    id: 3,
    type: "art_liked",
    reference_type: "art",
    reference_id: "art-1",
    read: false,
});

interface StubOptions {
    notifications?: Notification[];
    total?: number;
    loading?: boolean;
    markingId?: number | null;
    markRead?: (notif: Notification) => Promise<void>;
    markAllRead?: () => Promise<void>;
}

function stubFeed(options: StubOptions = {}) {
    const notifications = options.notifications ?? [];
    const total = options.total ?? notifications.length;
    const markRead = vi.fn(options.markRead ?? (() => Promise.resolve()));
    const markAllRead = vi.fn(options.markAllRead ?? (() => Promise.resolve()));
    const loadMore = vi.fn();

    useNotificationFeed.mockReturnValue({
        notifications,
        total,
        loading: options.loading ?? false,
        hasMore: notifications.length < total,
        loadMore,
        markingId: options.markingId ?? null,
        markRead,
        markAllRead,
    });

    return { markRead, markAllRead, loadMore };
}

interface PageOptions {
    unreadCount?: number;
}

function renderPage(options: PageOptions = {}) {
    return renderWithProviders(<NotificationsPage />, {
        route: "/notifications",
        notification: {
            unreadCount: options.unreadCount ?? 0,
        },
    });
}

describe("NotificationsPage", () => {
    it("waits while the first page of notifications is loading", () => {
        // given
        stubFeed({ loading: true, notifications: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading notifications...")).toBeInTheDocument();
    });

    it("says the inbox is empty when nothing has ever arrived", () => {
        // given
        stubFeed({ notifications: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No notifications yet")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Unread/ })).not.toBeInTheDocument();
    });

    it("shows the feed the hook handed it", () => {
        // given
        stubFeed({ notifications: [liked] });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.getByText("liked your post")).toHaveTextContent("Beatrice liked your post");
    });

    it("opens on the unread tab and leaves the read ones out", () => {
        // given
        stubFeed({ notifications: [liked, followed] });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.getByText("liked your post")).toHaveTextContent("Beatrice liked your post");
        expect(screen.queryByText("started following you")).not.toBeInTheDocument();
    });

    it("says there is nothing unread once everything has been read", () => {
        // given
        stubFeed({ notifications: [followed] });

        // when
        renderPage({ unreadCount: 0 });

        // then
        expect(screen.getByText("No unread notifications")).toBeInTheDocument();
    });

    it("hides the mark all button when nothing is unread", () => {
        // given
        stubFeed({ notifications: [followed] });

        // when
        renderPage({ unreadCount: 0 });

        // then
        expect(screen.queryByRole("button", { name: "Mark all as read" })).not.toBeInTheDocument();
    });

    it("asks the feed to mark the whole inbox read", async () => {
        // given
        const { markAllRead } = stubFeed({ notifications: [liked, artLiked] });
        const user = userEvent.setup();
        renderPage({ unreadCount: 2 });

        // when
        await user.click(screen.getByRole("button", { name: "Mark all as read" }));

        // then
        expect(markAllRead).toHaveBeenCalledOnce();
    });

    it("marks a notification read and follows it to the content it points at", async () => {
        // given
        const { markRead } = stubFeed({ notifications: [liked] });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByText("liked your post"));

        // then
        expect(markRead).toHaveBeenCalledWith(liked);
        expect(navigate).toHaveBeenCalledWith("/game-board/post-1");
    });

    it("does not mark an already read notification again", async () => {
        // given
        const { markRead } = stubFeed({ notifications: [followed] });
        const user = userEvent.setup();
        renderPage({ unreadCount: 0 });
        await user.click(screen.getByRole("button", { name: "All" }));

        // when
        await user.click(screen.getByText("started following you"));

        // then
        expect(markRead).not.toHaveBeenCalled();
        expect(navigate).toHaveBeenCalledWith("/user/ange");
    });

    it("keeps the reader in place when they only dismiss a notification", async () => {
        // given
        const { markRead } = stubFeed({ notifications: [liked] });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByRole("button", { name: "Mark as read" }));

        // then
        expect(markRead).toHaveBeenCalledWith(liked);
        expect(navigate).not.toHaveBeenCalled();
    });

    it("shows the dismissal is in flight while the feed says it is still marking", async () => {
        // given
        stubFeed({ notifications: [liked], markingId: 1 });

        // when
        renderPage({ unreadCount: 1 });

        // then
        await waitFor(() => expect(screen.getByRole("button", { name: "Marking..." })).toBeDisabled());
    });

    it("offers a tab for every category that has something in it", () => {
        // given
        stubFeed({ notifications: [liked, followed, artLiked] });

        // when
        renderPage({ unreadCount: 2 });

        // then
        expect(screen.getByRole("button", { name: /Game Board/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Gallery/ })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: /Social/ })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Theories/ })).not.toBeInTheDocument();
    });

    it("badges a category tab with how many of its notifications are unread", () => {
        // given
        stubFeed({ notifications: [liked, followed, artLiked] });

        // when
        renderPage({ unreadCount: 2 });

        // then
        expect(screen.getByRole("button", { name: /Game Board/ })).toHaveTextContent("Game Board1");
        expect(screen.getByRole("button", { name: /Social/ })).toHaveTextContent("Social");
        expect(screen.getByRole("button", { name: /Social/ })).not.toHaveTextContent("1");
    });

    it("groups everything by category once the reader asks for all of it", async () => {
        // given
        stubFeed({ notifications: [liked, followed, artLiked] });
        const user = userEvent.setup();
        renderPage({ unreadCount: 2 });

        // when
        await user.click(screen.getByRole("button", { name: "All" }));

        // then
        expect(screen.getByText("liked your post")).toHaveTextContent("Beatrice liked your post");
        expect(screen.getByText("started following you")).toHaveTextContent("Ange started following you");
        expect(screen.getByText("liked your art")).toHaveTextContent("Beatrice liked your art");
    });

    it("narrows the list to the single category the reader picked", async () => {
        // given
        stubFeed({ notifications: [liked, followed, artLiked] });
        const user = userEvent.setup();
        renderPage({ unreadCount: 2 });

        // when
        await user.click(screen.getByRole("button", { name: /Gallery/ }));

        // then
        expect(screen.getByText("liked your art")).toHaveTextContent("Beatrice liked your art");
        expect(screen.queryByText("liked your post")).not.toBeInTheDocument();
        expect(screen.queryByText("started following you")).not.toBeInTheDocument();
    });

    it("hides the load more button when the whole inbox is already on screen", () => {
        // given
        stubFeed({ notifications: [liked], total: 1 });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
    });

    it("asks the feed for a bigger page when there is more to see", async () => {
        // given
        const { loadMore } = stubFeed({ notifications: [liked], total: 80 });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByRole("button", { name: "Load more" }));

        // then
        expect(loadMore).toHaveBeenCalledOnce();
    });

    it("keeps asking every time the reader wants more", async () => {
        // given
        const { loadMore } = stubFeed({ notifications: [liked], total: 300 });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByRole("button", { name: "Load more" }));
        await user.click(screen.getByRole("button", { name: "Load more" }));

        // then
        expect(loadMore).toHaveBeenCalledTimes(2);
    });

    it("still follows the notification when marking it read fails", async () => {
        // given
        stubFeed({ notifications: [liked], markRead: () => Promise.reject(new Error("the servants are asleep")) });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByText("liked your post"));

        // then
        expect(navigate).toHaveBeenCalledWith("/game-board/post-1");
    });

    it("keeps the dismissal offered when marking one read fails", async () => {
        // given
        stubFeed({ notifications: [liked], markRead: () => Promise.reject(new Error("the servants are asleep")) });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByRole("button", { name: "Mark as read" }));

        // then
        expect(await screen.findByRole("button", { name: "Mark as read" })).toBeEnabled();
    });

    it("stays on its feet when marking everything read fails", async () => {
        // given
        stubFeed({
            notifications: [liked, artLiked],
            markAllRead: () => Promise.reject(new Error("the servants are asleep")),
        });
        const user = userEvent.setup();
        renderPage({ unreadCount: 2 });

        // when
        await user.click(screen.getByRole("button", { name: "Mark all as read" }));

        // then
        expect(screen.getByText("liked your post")).toBeInTheDocument();
    });

    it("shows a bundled chat room notification as the server worded it", () => {
        // given
        stubFeed({
            notifications: [
                makeNotification({
                    id: 7,
                    type: "chat_room_message",
                    reference_type: "chat",
                    count: 4,
                    message: "4 new messages in Rokkenjima",
                }),
            ],
        });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.getByText("4 new messages in Rokkenjima")).toBeInTheDocument();
        expect(screen.queryByText(/Beatrice/)).not.toBeInTheDocument();
    });

    it("names the staff role behind an edit to the reader's content", () => {
        // given
        stubFeed({
            notifications: [
                makeNotification({
                    id: 8,
                    type: "content_edited",
                    reference_type: "post",
                    message: "your post was edited",
                    actor: { ...beatrice, role: "moderator" },
                }),
            ],
        });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.getByText("your post was edited by Witch")).toHaveTextContent(
            "your post was edited by Witch Beatrice",
        );
    });

    it("falls back to the plain wording when there is no actor to name", () => {
        // given
        stubFeed({
            notifications: [
                makeNotification({
                    id: 9,
                    type: "journal_archived",
                    reference_type: "journal",
                    actor: { id: "", username: "", display_name: "" },
                }),
            ],
        });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.getByText("your journal was archived after 7 days of inactivity")).toBeInTheDocument();
    });
});
