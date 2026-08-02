import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { renderWithProviders } from "../../test-utils/render";
import type { Notification, WSMessage } from "../../types/api";
import { NotificationsPage } from "./NotificationsPage";

const { useNotificationsQuery, navigate } = vi.hoisted(() => ({
    useNotificationsQuery: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/notification", () => ({ useNotifications: useNotificationsQuery }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const beatrice = { id: "user-1", username: "beatrice", display_name: "Beatrice" };
const ange = { id: "user-2", username: "ange", display_name: "Ange" };

const listKey = queryKeys.notifications.list({ limit: 50, offset: 0 });

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
}

function stubQuery(options: StubOptions = {}) {
    const refresh = vi.fn(() => Promise.resolve(undefined));
    useNotificationsQuery.mockReturnValue({
        notifications: options.notifications ?? [],
        total: options.total ?? options.notifications?.length ?? 0,
        loading: options.loading ?? false,
        refresh,
    });

    return { refresh };
}

function seededClient(notifications: Notification[]): QueryClient {
    const client = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
    });
    client.setQueryData(listKey, { notifications, total: notifications.length });

    return client;
}

function cachedList(client: QueryClient) {
    return client.getQueryData<{ notifications: Notification[]; total: number }>(listKey);
}

interface PageOptions {
    unreadCount?: number;
    markRead?: (id: number) => Promise<void>;
    markAllRead?: () => Promise<void>;
    listeners?: ((msg: WSMessage) => void)[];
    queryClient?: QueryClient;
}

function renderPage(options: PageOptions = {}) {
    const listeners = options.listeners ?? [];

    return renderWithProviders(<NotificationsPage />, {
        route: "/notifications",
        queryClient: options.queryClient,
        notification: {
            unreadCount: options.unreadCount ?? 0,
            markRead: options.markRead ?? (() => Promise.resolve()),
            markAllRead: options.markAllRead ?? (() => Promise.resolve()),
            addWSListener: listener => {
                listeners.push(listener);
                return () => {};
            },
        },
    });
}

describe("NotificationsPage", () => {
    it("waits while the first page of notifications is loading", () => {
        // given
        stubQuery({ loading: true, notifications: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Loading notifications...")).toBeInTheDocument();
    });

    it("says the inbox is empty when nothing has ever arrived", () => {
        // given
        stubQuery({ notifications: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No notifications yet")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Unread/ })).not.toBeInTheDocument();
    });

    it("asks for the first fifty notifications", () => {
        // given
        stubQuery();

        // when
        renderPage();

        // then
        expect(useNotificationsQuery).toHaveBeenLastCalledWith(50, 0);
    });

    it("opens on the unread tab and leaves the read ones out", () => {
        // given
        stubQuery({ notifications: [liked, followed] });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.getByText("liked your post")).toHaveTextContent("Beatrice liked your post");
        expect(screen.queryByText("started following you")).not.toBeInTheDocument();
    });

    it("says there is nothing unread once everything has been read", () => {
        // given
        stubQuery({ notifications: [followed] });

        // when
        renderPage({ unreadCount: 0 });

        // then
        expect(screen.getByText("No unread notifications")).toBeInTheDocument();
    });

    it("hides the mark all button when nothing is unread", () => {
        // given
        stubQuery({ notifications: [followed] });

        // when
        renderPage({ unreadCount: 0 });

        // then
        expect(screen.queryByRole("button", { name: "Mark all as read" })).not.toBeInTheDocument();
    });

    it("marks the whole inbox read and stops showing them as unread", async () => {
        // given
        stubQuery({ notifications: [liked, artLiked] });
        const markAllRead = vi.fn(() => Promise.resolve());
        const queryClient = seededClient([liked, artLiked]);
        const user = userEvent.setup();
        renderPage({ unreadCount: 2, markAllRead, queryClient });

        // when
        await user.click(screen.getByRole("button", { name: "Mark all as read" }));

        // then
        expect(markAllRead).toHaveBeenCalledOnce();
        expect(cachedList(queryClient)?.notifications.every(n => n.read)).toBe(true);
    });

    it("marks a notification read and follows it to the content it points at", async () => {
        // given
        stubQuery({ notifications: [liked] });
        const markRead = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderPage({ unreadCount: 1, markRead });

        // when
        await user.click(screen.getByText("liked your post"));

        // then
        expect(markRead).toHaveBeenCalledWith(1);
        expect(navigate).toHaveBeenCalledWith("/game-board/post-1");
    });

    it("does not mark an already read notification again", async () => {
        // given
        stubQuery({ notifications: [followed] });
        const markRead = vi.fn(() => Promise.resolve());
        const user = userEvent.setup();
        renderPage({ unreadCount: 0, markRead });
        await user.click(screen.getByRole("button", { name: "All" }));

        // when
        await user.click(screen.getByText("started following you"));

        // then
        expect(markRead).not.toHaveBeenCalled();
        expect(navigate).toHaveBeenCalledWith("/user/ange");
    });

    it("keeps the reader in place when they only dismiss a notification", async () => {
        // given
        stubQuery({ notifications: [liked] });
        const markRead = vi.fn(() => Promise.resolve());
        const queryClient = seededClient([liked]);
        const user = userEvent.setup();
        renderPage({ unreadCount: 1, markRead, queryClient });

        // when
        await user.click(screen.getByRole("button", { name: "Mark as read" }));

        // then
        expect(markRead).toHaveBeenCalledWith(1);
        expect(navigate).not.toHaveBeenCalled();
        expect(cachedList(queryClient)?.notifications[0].read).toBe(true);
    });

    it("shows the dismissal is in flight while the server is still thinking", async () => {
        // given
        stubQuery({ notifications: [liked] });
        const markRead = vi.fn(() => new Promise<void>(() => {}));
        const user = userEvent.setup();
        renderPage({ unreadCount: 1, markRead });

        // when
        await user.click(screen.getByRole("button", { name: "Mark as read" }));

        // then
        await waitFor(() => expect(screen.getByRole("button", { name: "Marking..." })).toBeDisabled());
    });

    it("offers a tab for every category that has something in it", () => {
        // given
        stubQuery({ notifications: [liked, followed, artLiked] });

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
        stubQuery({ notifications: [liked, followed, artLiked] });

        // when
        renderPage({ unreadCount: 2 });

        // then
        expect(screen.getByRole("button", { name: /Game Board/ })).toHaveTextContent("Game Board1");
        expect(screen.getByRole("button", { name: /Social/ })).toHaveTextContent("Social");
        expect(screen.getByRole("button", { name: /Social/ })).not.toHaveTextContent("1");
    });

    it("groups everything by category once the reader asks for all of it", async () => {
        // given
        stubQuery({ notifications: [liked, followed, artLiked] });
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
        stubQuery({ notifications: [liked, followed, artLiked] });
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
        stubQuery({ notifications: [liked], total: 1 });

        // when
        renderPage({ unreadCount: 1 });

        // then
        expect(screen.queryByRole("button", { name: "Load more" })).not.toBeInTheDocument();
    });

    it("asks for a bigger page of the inbox when there is more to see", async () => {
        // given
        stubQuery({ notifications: [liked], total: 80 });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByRole("button", { name: "Load more" }));

        // then
        expect(useNotificationsQuery).toHaveBeenLastCalledWith(100, 0);
    });

    it("keeps growing the page every time the reader asks for more", async () => {
        // given
        stubQuery({ notifications: [liked], total: 300 });
        const user = userEvent.setup();
        renderPage({ unreadCount: 1 });

        // when
        await user.click(screen.getByRole("button", { name: "Load more" }));
        await user.click(screen.getByRole("button", { name: "Load more" }));

        // then
        expect(useNotificationsQuery).toHaveBeenLastCalledWith(150, 0);
    });

    it("subscribes to the socket once and keeps that listener through a filter change", async () => {
        // given
        stubQuery({ notifications: [liked, followed] });
        const listeners: ((msg: WSMessage) => void)[] = [];
        const user = userEvent.setup();
        renderPage({ unreadCount: 1, listeners });

        // when
        await user.click(screen.getByRole("button", { name: "All" }));

        // then
        expect(listeners).toHaveLength(1);
    });

    it("still follows the notification when marking it read fails", async () => {
        // given
        stubQuery({ notifications: [liked] });
        const markRead = vi.fn(() => Promise.reject(new Error("the servants are asleep")));
        const user = userEvent.setup();
        renderPage({ unreadCount: 1, markRead });

        // when
        await user.click(screen.getByText("liked your post"));

        // then
        expect(navigate).toHaveBeenCalledWith("/game-board/post-1");
    });

    it("offers the dismissal again when marking one read fails", async () => {
        // given
        stubQuery({ notifications: [liked] });
        const markRead = vi.fn(() => Promise.reject(new Error("the servants are asleep")));
        const queryClient = seededClient([liked]);
        const user = userEvent.setup();
        renderPage({ unreadCount: 1, markRead, queryClient });

        // when
        await user.click(screen.getByRole("button", { name: "Mark as read" }));

        // then
        expect(await screen.findByRole("button", { name: "Mark as read" })).toBeEnabled();
        expect(cachedList(queryClient)?.notifications[0].read).toBe(false);
    });

    it("leaves the inbox unread when marking everything read fails", async () => {
        // given
        stubQuery({ notifications: [liked, artLiked] });
        const markAllRead = vi.fn(() => Promise.reject(new Error("the servants are asleep")));
        const queryClient = seededClient([liked, artLiked]);
        const user = userEvent.setup();
        renderPage({ unreadCount: 2, markAllRead, queryClient });

        // when
        await user.click(screen.getByRole("button", { name: "Mark all as read" }));

        // then
        expect(cachedList(queryClient)?.notifications.every(n => !n.read)).toBe(true);
    });

    it("drops a live notification straight into the cached list", () => {
        // given
        stubQuery({ notifications: [liked] });
        const listeners: ((msg: WSMessage) => void)[] = [];
        const queryClient = seededClient([liked]);
        renderPage({ unreadCount: 1, listeners, queryClient });

        // when
        act(() => {
            for (const listener of listeners) {
                listener({ type: "notification", data: artLiked });
            }
        });

        // then
        expect(cachedList(queryClient)?.notifications.map(n => n.id)).toEqual([3, 1]);
        expect(cachedList(queryClient)?.total).toBe(2);
    });

    it("refuses to list the same live notification twice", () => {
        // given
        stubQuery({ notifications: [liked] });
        const listeners: ((msg: WSMessage) => void)[] = [];
        const queryClient = seededClient([liked]);
        renderPage({ unreadCount: 1, listeners, queryClient });

        // when
        act(() => {
            for (const listener of listeners) {
                listener({ type: "notification", data: liked });
            }
        });

        // then
        expect(cachedList(queryClient)?.notifications).toHaveLength(1);
        expect(cachedList(queryClient)?.total).toBe(1);
    });

    it("ignores socket traffic that is not a notification", () => {
        // given
        stubQuery({ notifications: [liked] });
        const listeners: ((msg: WSMessage) => void)[] = [];
        const queryClient = seededClient([liked]);
        renderPage({ unreadCount: 1, listeners, queryClient });

        // when
        act(() => {
            for (const listener of listeners) {
                listener({ type: "presence", data: artLiked });
            }
        });

        // then
        expect(cachedList(queryClient)?.notifications).toHaveLength(1);
    });

    it("shows a bundled chat room notification as the server worded it", () => {
        // given
        stubQuery({
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
        stubQuery({
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
        stubQuery({
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
