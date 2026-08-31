import { act, screen } from "@testing-library/react";
import { useContext, useEffect } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import { makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../test-utils/render";
import { FakeWebSocket } from "../test-utils/ws";
import type { UserProfile } from "../types/api";
import { NotificationProvider } from "./NotificationContext";
import { NotificationContext, type NotificationContextValue } from "./notificationContextValue";

const {
    useUnreadCount,
    useChatUnreadCount,
    useLiveGameRooms,
    useLiveStreamsCount,
    markNotificationRead,
    markAllNotificationsRead,
    unreadRefresh,
    showDesktopNotification,
    playNotificationSound,
    getAuthToken,
    isNativeApp,
} = vi.hoisted(() => ({
    useUnreadCount: vi.fn(),
    useChatUnreadCount: vi.fn(),
    useLiveGameRooms: vi.fn(),
    useLiveStreamsCount: vi.fn(),
    markNotificationRead: vi.fn(),
    markAllNotificationsRead: vi.fn(),
    unreadRefresh: vi.fn(),
    showDesktopNotification: vi.fn(),
    playNotificationSound: vi.fn(),
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
}));

vi.mock("../hooks/queries/notification", () => ({ useUnreadCount }));
vi.mock("../hooks/queries/chat", () => ({ useChatUnreadCount }));
vi.mock("../hooks/queries/gameRoom", () => ({ useLiveGameRooms }));
vi.mock("../hooks/queries/stream", () => ({ useLiveStreamsCount }));
vi.mock("../hooks/mutations/notification", () => ({
    useMarkNotificationRead: () => ({ mutateAsync: markNotificationRead }),
    useMarkAllNotificationsRead: () => ({ mutateAsync: markAllNotificationsRead }),
}));
vi.mock("../platform/desktopNotifications", () => ({ showDesktopNotification }));
vi.mock("../platform/sound", () => ({ playNotificationSound }));
vi.mock("../api/authToken", () => ({ getAuthToken }));
vi.mock("../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

let captured: NotificationContextValue | null = null;

function Probe() {
    const value = useContext(NotificationContext);
    useEffect(() => {
        captured = value;
    }, [value]);

    if (!value) {
        return <p>no notification context</p>;
    }

    return (
        <div>
            <p>{`unread: ${value.unreadCount}`}</p>
            <p>{`chat: ${value.chatUnreadCount}`}</p>
            <p>{`games: ${value.liveGamesCount}`}</p>
            <p>{`streams: ${value.liveStreamsCount}`}</p>
        </div>
    );
}

function context(): NotificationContextValue {
    if (!captured) {
        throw new Error("the notification context was never rendered");
    }

    return captured;
}

const signedIn = makeUser({ id: "user-1", username: "beatrice" });

function renderProvider(user: UserProfile | null = signedIn) {
    const queryClient = createTestQueryClient();
    const result = renderWithProviders(
        <NotificationProvider>
            <Probe />
        </NotificationProvider>,
        { user, queryClient },
    );

    return { ...result, queryClient };
}

function lastSocket(): FakeWebSocket {
    const socket = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!socket) {
        throw new Error("no websocket was opened");
    }

    return socket;
}

function openSocket(socket: FakeWebSocket = lastSocket()): void {
    act(() => {
        socket.onopen?.();
    });
}

function emit(msg: { type: string; data?: unknown }, socket: FakeWebSocket = lastSocket()): void {
    act(() => {
        socket.onmessage?.({ data: JSON.stringify(msg) });
    });
}

beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    useUnreadCount.mockReturnValue({ count: 0, refresh: unreadRefresh });
    useChatUnreadCount.mockReturnValue({ count: 0, refresh: vi.fn() });
    useLiveGameRooms.mockReturnValue({ total: 0, rooms: [], loading: false, error: "", refresh: vi.fn() });
    useLiveStreamsCount.mockReturnValue({ count: 0 });
    markNotificationRead.mockResolvedValue(undefined);
    markAllNotificationsRead.mockResolvedValue(undefined);
    unreadRefresh.mockResolvedValue(undefined);
    getAuthToken.mockReturnValue(null);
});

afterEach(() => {
    captured = null;
    vi.restoreAllMocks();
});

describe("NotificationProvider", () => {
    it("closes the socket when the provider goes away", () => {
        // given
        const { unmount } = renderProvider();
        const socket = lastSocket();

        // when
        unmount();

        // then
        expect(socket.closeCount).toBe(1);
    });

    it("shows a desktop notification and plays the sound by default", () => {
        // given
        renderProvider();
        openSocket();

        // when
        emit({ type: "notification", data: { id: 4, type: "chat_mention" } });

        // then
        expect(showDesktopNotification).toHaveBeenCalledWith({ id: 4, type: "chat_mention" });
        expect(playNotificationSound).toHaveBeenCalledOnce();
    });

    it("hides the unread counts from a signed out visitor", () => {
        // given
        useUnreadCount.mockReturnValue({ count: 5, refresh: unreadRefresh });
        useChatUnreadCount.mockReturnValue({ count: 6, refresh: vi.fn() });

        // when
        renderProvider(null);

        // then
        expect(screen.getByText("unread: 0")).toBeInTheDocument();
        expect(screen.getByText("chat: 0")).toBeInTheDocument();
    });

    it("shows the unread counts to a signed in user", () => {
        // given
        useUnreadCount.mockReturnValue({ count: 5, refresh: unreadRefresh });
        useChatUnreadCount.mockReturnValue({ count: 6, refresh: vi.fn() });

        // when
        renderProvider();

        // then
        expect(screen.getByText("unread: 5")).toBeInTheDocument();
        expect(screen.getByText("chat: 6")).toBeInTheDocument();
    });

    it("counts the live games and the live streams for everybody", () => {
        // given
        useLiveGameRooms.mockReturnValue({ total: 2, rooms: [], loading: false, error: "", refresh: vi.fn() });
        useLiveStreamsCount.mockReturnValue({ count: 3 });

        // when
        renderProvider(null);

        // then
        expect(screen.getByText("games: 2")).toBeInTheDocument();
        expect(screen.getByText("streams: 3")).toBeInTheDocument();
    });

    it("marks one notification read and then refreshes the unread count", async () => {
        // given
        renderProvider();
        openSocket();

        // when
        await act(async () => {
            await context().markRead(12);
        });

        // then
        expect(markNotificationRead).toHaveBeenCalledWith(12);
        expect(unreadRefresh.mock.invocationCallOrder[0]).toBeGreaterThan(
            markNotificationRead.mock.invocationCallOrder[0],
        );
    });

    it("empties the unread count once everything is marked read", async () => {
        // given
        const { queryClient } = renderProvider();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        openSocket();

        // when
        await act(async () => {
            await context().markAllRead();
        });

        // then
        expect(markAllNotificationsRead).toHaveBeenCalledOnce();
        expect(setQueryData).toHaveBeenCalledWith(queryKeys.notifications.unreadCount(), { count: 0 });
    });
});
