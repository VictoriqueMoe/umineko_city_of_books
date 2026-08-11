import { act, screen, waitFor } from "@testing-library/react";
import { useContext, useEffect } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import { makeUser } from "../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../test-utils/render";
import type { UserProfile } from "../types/api";
import { NotificationProvider } from "./NotificationContext";
import { NotificationContext, type NotificationContextValue } from "./notificationContextValue";

const {
    useUnreadCount,
    useChatUnreadCount,
    useLiveGameRooms,
    listLiveStreams,
    markNotificationRead,
    markAllNotificationsRead,
    unreadRefresh,
    showDesktopNotification,
    playNotificationSound,
    getAuthToken,
} = vi.hoisted(() => ({
    useUnreadCount: vi.fn(),
    useChatUnreadCount: vi.fn(),
    useLiveGameRooms: vi.fn(),
    listLiveStreams: vi.fn(),
    markNotificationRead: vi.fn(),
    markAllNotificationsRead: vi.fn(),
    unreadRefresh: vi.fn(),
    showDesktopNotification: vi.fn(),
    playNotificationSound: vi.fn(),
    getAuthToken: vi.fn(),
}));

vi.mock("../api/queries/notification", () => ({ useUnreadCount }));
vi.mock("../api/queries/chat", () => ({ useChatUnreadCount }));
vi.mock("../api/queries/gameRoom", () => ({ useLiveGameRooms }));
vi.mock("../api/endpoints", () => ({ listLiveStreams }));
vi.mock("../api/mutations/notification", () => ({
    useMarkNotificationRead: () => ({ mutateAsync: markNotificationRead }),
    useMarkAllNotificationsRead: () => ({ mutateAsync: markAllNotificationsRead }),
}));
vi.mock("../utils/notifications", () => ({ showDesktopNotification }));
vi.mock("../utils/sound", () => ({ playNotificationSound }));
vi.mock("../utils/authToken", () => ({ getAuthToken, isNativeApp: () => false, clientPlatform: () => "web" }));

type CountUpdater = (prev: { count: number } | undefined) => { count: number };

class FakeWebSocket {
    static readonly OPEN = 1;
    static readonly CLOSED = 3;
    static instances: FakeWebSocket[] = [];

    readyState = 1;
    closeCount = 0;
    sent: string[] = [];
    onopen: (() => void) | null = null;
    onmessage: ((event: { data: string }) => void) | null = null;
    onclose: (() => void) | null = null;
    onerror: (() => void) | null = null;

    constructor(readonly url: string) {
        FakeWebSocket.instances.push(this);
    }

    send(payload: string): void {
        this.sent.push(payload);
    }

    close(): void {
        this.closeCount += 1;
        this.readyState = FakeWebSocket.CLOSED;
    }
}

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
            <p>{`epoch: ${value.wsEpoch}`}</p>
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
    const setUser = vi.fn();
    const queryClient = createTestQueryClient();
    const result = renderWithProviders(
        <NotificationProvider>
            <Probe />
        </NotificationProvider>,
        { user, auth: { setUser }, queryClient },
    );

    return { ...result, queryClient, setUser };
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

function setVisibility(state: "visible" | "hidden"): void {
    Object.defineProperty(document, "visibilityState", { configurable: true, get: () => state });
}

beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    useUnreadCount.mockReturnValue({ count: 0, refresh: unreadRefresh });
    useChatUnreadCount.mockReturnValue({ count: 0, refresh: vi.fn() });
    useLiveGameRooms.mockReturnValue({ total: 0, rooms: [], loading: false, error: "", refresh: vi.fn() });
    listLiveStreams.mockResolvedValue({ streams: [] });
    markNotificationRead.mockResolvedValue(undefined);
    markAllNotificationsRead.mockResolvedValue(undefined);
    unreadRefresh.mockResolvedValue(undefined);
    getAuthToken.mockReturnValue(null);
});

afterEach(() => {
    captured = null;
    vi.restoreAllMocks();
    Reflect.deleteProperty(document, "visibilityState");
});

describe("NotificationProvider", () => {
    it("never opens a socket for a signed out visitor", () => {
        // given
        const nobody = null;

        // when
        renderProvider(nobody);

        // then
        expect(FakeWebSocket.instances).toHaveLength(0);
    });

    it("opens a socket for a signed in user", () => {
        // given
        const wsOrigin = window.location.origin.replace(/^http/, "ws");

        // when
        renderProvider();

        // then
        expect(FakeWebSocket.instances).toHaveLength(1);
        expect(lastSocket().url).toBe(`${wsOrigin}/api/v1/ws`);
    });

    it("carries the native session token on the socket url", () => {
        // given
        getAuthToken.mockReturnValue("tok en&1");

        // when
        renderProvider();

        // then
        expect(lastSocket().url).toContain("?token=tok%20en%261");
    });

    it("closes the socket when the provider goes away", () => {
        // given
        const { unmount } = renderProvider();
        const socket = lastSocket();

        // when
        unmount();

        // then
        expect(socket.closeCount).toBe(1);
    });

    it("bumps the socket epoch every time a connection opens", () => {
        // given
        renderProvider();
        expect(screen.getByText("epoch: 0")).toBeInTheDocument();

        // when
        openSocket();

        // then
        expect(screen.getByText("epoch: 1")).toBeInTheDocument();
    });

    it("asks for fresh site info as soon as the socket opens", () => {
        // given
        const refreshes = vi.fn();
        window.addEventListener("site-info-refresh", refreshes);
        renderProvider();

        // when
        openSocket();

        // then
        expect(refreshes).toHaveBeenCalledOnce();
        window.removeEventListener("site-info-refresh", refreshes);
    });

    it("hands every message to the listeners that registered for them", () => {
        // given
        renderProvider();
        openSocket();
        const first = vi.fn();
        const second = vi.fn();
        context().addWSListener(first);
        context().addWSListener(second);

        // when
        emit({ type: "chat_message", data: { body: "uu~" } });

        // then
        expect(first).toHaveBeenCalledWith({ type: "chat_message", data: { body: "uu~" } });
        expect(second).toHaveBeenCalledOnce();
    });

    it("stops delivering to a listener that has been removed", () => {
        // given
        renderProvider();
        openSocket();
        const handler = vi.fn();
        const remove = context().addWSListener(handler);

        // when
        remove();
        emit({ type: "chat_message", data: {} });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("ignores a payload that is not valid json", () => {
        // given
        renderProvider();
        openSocket();
        const handler = vi.fn();
        context().addWSListener(handler);

        // when
        act(() => {
            lastSocket().onmessage?.({ data: "not json at all" });
        });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("swallows a keepalive pong without telling the listeners", () => {
        // given
        renderProvider();
        openSocket();
        const handler = vi.fn();
        context().addWSListener(handler);

        // when
        emit({ type: "pong", data: {} });

        // then
        expect(handler).not.toHaveBeenCalled();
    });

    it("counts an incoming notification and refreshes the notifications list", () => {
        // given
        const { queryClient } = renderProvider();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
        openSocket();

        // when
        emit({ type: "notification", data: { id: 4, type: "chat_mention" } });

        // then
        const [key, updater] = setQueryData.mock.calls[0];
        expect(key).toEqual(queryKeys.notifications.unreadCount());
        expect((updater as unknown as CountUpdater)(undefined)).toEqual({ count: 1 });
        expect((updater as unknown as CountUpdater)({ count: 4 })).toEqual({ count: 5 });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["notifications", "list"] });
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

    it("stays silent when the user has turned the notification sound off", () => {
        // given
        renderProvider(makeUser({ id: "user-1", private: { play_notification_sound: false } }));
        openSocket();

        // when
        emit({ type: "notification", data: { id: 4, type: "chat_mention" } });

        // then
        expect(showDesktopNotification).toHaveBeenCalledOnce();
        expect(playNotificationSound).not.toHaveBeenCalled();
    });

    it("updates the signed in user when their own role changes", () => {
        // given
        const { setUser } = renderProvider();
        openSocket();

        // when
        emit({ type: "role_changed", data: { user_id: "user-1", role: "moderator" } });

        // then
        expect(setUser).toHaveBeenCalledWith(expect.objectContaining({ id: "user-1", role: "moderator" }));
    });

    it("leaves the signed in user alone when somebody else changes role", () => {
        // given
        const { setUser } = renderProvider();
        openSocket();

        // when
        emit({ type: "role_changed", data: { user_id: "user-2", role: "moderator" } });

        // then
        expect(setUser).not.toHaveBeenCalled();
    });

    it("ignores a role change that names nobody", () => {
        // given
        const { setUser } = renderProvider();
        openSocket();

        // when
        emit({ type: "role_changed", data: { role: "moderator" } });

        // then
        expect(setUser).not.toHaveBeenCalled();
    });

    it("locks the signed in user when the lock message is about them", () => {
        // given
        const { setUser } = renderProvider();
        openSocket();

        // when
        emit({ type: "lock_changed", data: { user_id: "user-1", locked: true, lock_reason: "too much magic" } });

        // then
        expect(setUser).toHaveBeenCalledWith(expect.objectContaining({ locked: true, lock_reason: "too much magic" }));
    });

    it("bans the signed in user when the ban message is about them", () => {
        // given
        const { setUser } = renderProvider();
        openSocket();

        // when
        emit({ type: "ban_changed", data: { user_id: "user-1", banned: true } });

        // then
        expect(setUser).toHaveBeenCalledWith(expect.objectContaining({ banned: true, ban_reason: "" }));
    });

    it("patches only the profile fields the message carried", () => {
        // given
        const { setUser } = renderProvider();
        openSocket();

        // when
        emit({ type: "profile_changed", data: { user_id: "user-1", display_name: "Beato" } });

        // then
        expect(setUser).toHaveBeenCalledWith(expect.objectContaining({ display_name: "Beato", avatar_url: "" }));
    });

    it("asks for fresh site info when a leaderboard changes", () => {
        // given
        const refreshes = vi.fn();
        renderProvider();
        openSocket();
        window.addEventListener("site-info-refresh", refreshes);

        // when
        emit({ type: "top_detective_changed", data: {} });

        // then
        expect(refreshes).toHaveBeenCalledOnce();
        window.removeEventListener("site-info-refresh", refreshes);
    });

    it("stores the chat unread total that the server sent", () => {
        // given
        const { queryClient } = renderProvider();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        openSocket();

        // when
        emit({ type: "chat_unread_bumped", data: { total: 7 } });

        // then
        expect(setQueryData).toHaveBeenCalledWith(["chat", "unread-count"], { count: 7 });
    });

    it("ignores a chat unread message with no usable total", () => {
        // given
        const { queryClient } = renderProvider();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        openSocket();

        // when
        emit({ type: "chat_read", data: { total: "lots" } });

        // then
        expect(setQueryData).not.toHaveBeenCalled();
    });

    it("stores the live games count that the server sent", () => {
        // given
        const { queryClient } = renderProvider();
        const setQueryData = vi.spyOn(queryClient, "setQueryData");
        openSocket();

        // when
        emit({ type: "live_games_count", data: { count: 3 } });

        // then
        const [key, updater] = setQueryData.mock.calls[0];
        expect(key).toEqual(queryKeys.gameRoom.live());
        expect((updater as unknown as (prev: unknown) => unknown)(undefined)).toEqual({ rooms: [], total: 3 });
    });

    it("refetches the live streams when a stream goes live", () => {
        // given
        const { queryClient } = renderProvider();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
        openSocket();

        // when
        emit({ type: "stream_live", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["streams", "live"] });
    });

    it("refetches the chatbot list when a chatbot is created, edited or deleted", () => {
        // given
        const { queryClient } = renderProvider();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
        openSocket();

        // when
        emit({ type: "chatbots_changed", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chatbots.all });
    });

    it("announces a closed secret to the rest of the app", () => {
        // given
        const closed = vi.fn();
        window.addEventListener("secret-closed", closed);
        renderProvider();
        openSocket();

        // when
        emit({ type: "secret_closed", data: { secret_id: "secret-1" } });

        // then
        expect(closed).toHaveBeenCalledOnce();
        expect((closed.mock.calls[0][0] as CustomEvent).detail).toEqual({ secret_id: "secret-1" });
        window.removeEventListener("secret-closed", closed);
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

    it("counts the live games and the live streams for everybody", async () => {
        // given
        useLiveGameRooms.mockReturnValue({ total: 2, rooms: [], loading: false, error: "", refresh: vi.fn() });
        listLiveStreams.mockResolvedValue({ streams: [{ id: "s1" }, { id: "s2" }, { id: "s3" }] });

        // when
        renderProvider(null);

        // then
        expect(screen.getByText("games: 2")).toBeInTheDocument();
        await waitFor(() => expect(screen.getByText("streams: 3")).toBeInTheDocument());
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

    it("sends a message down an open socket", () => {
        // given
        renderProvider();
        openSocket();

        // when
        context().sendWSMessage({ type: "typing", data: { room_id: "room-1" } });

        // then
        expect(lastSocket().sent).toContain(JSON.stringify({ type: "typing", data: { room_id: "room-1" } }));
    });

    it("drops a message when the socket is not open", () => {
        // given
        renderProvider();
        openSocket();
        const socket = lastSocket();
        socket.readyState = FakeWebSocket.CLOSED;

        // when
        context().sendWSMessage({ type: "typing", data: {} });

        // then
        expect(socket.sent).toHaveLength(0);
    });

    it("pings the server while the connection is idle", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        renderProvider();
        openSocket();

        // when
        act(() => {
            vi.advanceTimersByTime(20_000);
        });

        // then
        expect(lastSocket().sent).toEqual([JSON.stringify({ type: "ping", data: {} })]);
    });

    it("probes a connection that has gone quiet for too long rather than closing it", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        renderProvider();
        openSocket();
        const socket = lastSocket();

        // when
        act(() => {
            vi.advanceTimersByTime(100_000);
        });

        // then
        expect(socket.sent).toHaveLength(5);
        expect(socket.closeCount).toBe(0);
    });

    it("closes a quiet connection once the probe goes unanswered", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        renderProvider();
        openSocket();
        const socket = lastSocket();

        // when
        act(() => {
            vi.advanceTimersByTime(100_000);
        });
        act(() => {
            vi.advanceTimersByTime(5_000);
        });

        // then
        expect(socket.closeCount).toBe(1);
    });

    it("keeps a quiet connection that answers the probe", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        renderProvider();
        openSocket();
        const socket = lastSocket();

        // when
        act(() => {
            vi.advanceTimersByTime(100_000);
        });
        act(() => {
            vi.advanceTimersByTime(1_000);
        });
        emit({ type: "pong", data: {} }, socket);
        act(() => {
            vi.advanceTimersByTime(10_000);
        });

        // then
        expect(socket.closeCount).toBe(0);
    });

    it("keeps the connection alive while messages are still arriving", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00Z"));
        renderProvider();
        openSocket();
        const socket = lastSocket();

        // when
        act(() => {
            vi.advanceTimersByTime(80_000);
        });
        emit({ type: "chat_message", data: {} }, socket);
        act(() => {
            vi.advanceTimersByTime(40_000);
        });

        // then
        expect(socket.closeCount).toBe(0);
    });

    it("reconnects with a growing backoff after the connection drops", () => {
        // given
        vi.useFakeTimers();
        vi.spyOn(Math, "random").mockReturnValue(1);
        renderProvider();
        openSocket();

        // when
        act(() => {
            lastSocket().onclose?.();
        });

        // then
        expect(FakeWebSocket.instances).toHaveLength(1);
        act(() => {
            vi.advanceTimersByTime(1000);
        });
        expect(FakeWebSocket.instances).toHaveLength(2);

        act(() => {
            lastSocket().onclose?.();
        });
        act(() => {
            vi.advanceTimersByTime(1000);
        });
        expect(FakeWebSocket.instances).toHaveLength(2);
        act(() => {
            vi.advanceTimersByTime(1000);
        });
        expect(FakeWebSocket.instances).toHaveLength(3);
    });

    it("spreads the reconnect across the backoff window instead of retrying in lockstep", () => {
        // given
        vi.useFakeTimers();
        vi.spyOn(Math, "random").mockReturnValue(0);
        renderProvider();
        openSocket();

        // when
        act(() => {
            lastSocket().onclose?.();
        });
        act(() => {
            vi.advanceTimersByTime(0);
        });

        // then
        expect(FakeWebSocket.instances).toHaveLength(2);
    });

    it("never waits longer than the backoff cap", () => {
        // given
        vi.useFakeTimers();
        vi.spyOn(Math, "random").mockReturnValue(0.999);
        renderProvider();
        openSocket();

        // when
        act(() => {
            lastSocket().onclose?.();
        });
        act(() => {
            vi.advanceTimersByTime(1000);
        });

        // then
        expect(FakeWebSocket.instances).toHaveLength(2);
    });

    it("ignores a late close from a socket that has already been replaced", () => {
        // given
        vi.useFakeTimers();
        const account = makeUser({ id: "user-1", username: "beatrice" });
        const queryClient = createTestQueryClient();
        const { rerender } = renderWithProviders(
            <NotificationProvider>
                <Probe />
            </NotificationProvider>,
            { user: account, auth: { setUser: vi.fn() }, queryClient },
        );
        openSocket();
        const stale = lastSocket();
        account.id = "user-2";
        act(() => {
            rerender(
                <NotificationProvider>
                    <Probe />
                </NotificationProvider>,
            );
        });
        const live = lastSocket();
        openSocket(live);

        // when
        act(() => {
            stale.onclose?.();
        });
        act(() => {
            vi.advanceTimersByTime(5000);
        });

        // then
        expect(FakeWebSocket.instances).toHaveLength(2);
        context().sendWSMessage({ type: "typing", data: {} });
        expect(live.sent).toContain(JSON.stringify({ type: "typing", data: {} }));
    });

    it("never reconnects once the provider has gone away", () => {
        // given
        vi.useFakeTimers();
        const { unmount } = renderProvider();
        openSocket();
        const socket = lastSocket();

        // when
        unmount();
        act(() => {
            socket.onclose?.();
        });
        act(() => {
            vi.advanceTimersByTime(30_000);
        });

        // then
        expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it("closes a socket that errored so the reconnect can take over", () => {
        // given
        renderProvider();
        openSocket();
        const socket = lastSocket();

        // when
        act(() => {
            socket.onerror?.();
        });

        // then
        expect(socket.closeCount).toBe(1);
    });

    it("pings the server when the tab becomes visible again", () => {
        // given
        setVisibility("visible");
        renderProvider();
        openSocket();

        // when
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });

        // then
        expect(lastSocket().sent).toEqual([JSON.stringify({ type: "ping", data: {} })]);
    });

    it("probes a stale socket when the tab becomes visible again rather than closing it", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        renderProvider();
        const socket = lastSocket();

        // when
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });

        // then
        expect(socket.sent).toHaveLength(1);
        expect(socket.closeCount).toBe(0);
    });

    it("closes the socket when the visibility probe goes unanswered", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        renderProvider();
        const socket = lastSocket();

        // when
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });
        act(() => {
            vi.advanceTimersByTime(5_000);
        });

        // then
        expect(socket.closeCount).toBe(1);
    });

    it("keeps the socket when the visibility probe is answered", () => {
        // given
        vi.useFakeTimers();
        setVisibility("visible");
        renderProvider();
        const socket = lastSocket();

        // when
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });
        act(() => {
            vi.advanceTimersByTime(1_000);
        });
        emit({ type: "pong", data: {} }, socket);
        act(() => {
            vi.advanceTimersByTime(10_000);
        });

        // then
        expect(socket.closeCount).toBe(0);
    });

    it("does nothing when the tab is being hidden", () => {
        // given
        setVisibility("hidden");
        renderProvider();
        openSocket();
        const socket = lastSocket();

        // when
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });

        // then
        expect(socket.sent).toHaveLength(0);
        expect(socket.closeCount).toBe(0);
    });
});
