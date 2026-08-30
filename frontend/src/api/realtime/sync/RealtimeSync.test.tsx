import { act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils/render";
import { FakeWebSocket } from "../../../test-utils/ws";
import type { UserProfile } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { RealtimeSync } from "./RealtimeSync";

const { getAuthToken, isNativeApp } = vi.hoisted(() => ({
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
}));

vi.mock("../../authToken", () => ({ getAuthToken }));
vi.mock("../../../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

const signedIn = makeUser({ id: "user-1", username: "beatrice" });

function renderSync(viewer: UserProfile | null = signedIn) {
    const showNotification = vi.fn();
    const playSound = vi.fn();
    const queryClient = createTestQueryClient();
    const result = renderWithProviders(
        <RealtimeSync viewer={viewer} showNotification={showNotification} playSound={playSound} />,
        { queryClient },
    );

    return { ...result, queryClient, showNotification, playSound };
}

function lastSocket(): FakeWebSocket {
    const socket = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!socket) {
        throw new Error("no websocket was opened");
    }

    return socket;
}

function openSocket(): void {
    act(() => {
        lastSocket().onopen?.();
    });
}

function emit(msg: { type: string; data?: unknown }): void {
    act(() => {
        lastSocket().onmessage?.({ data: JSON.stringify(msg) });
    });
}

beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    getAuthToken.mockReturnValue(null);
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("RealtimeSync", () => {
    it("never opens a socket for a signed out visitor", () => {
        // given
        const nobody = null;

        // when
        renderSync(nobody);

        // then
        expect(FakeWebSocket.instances).toHaveLength(0);
    });

    it("opens exactly one socket for the signed in viewer", () => {
        // given
        const viewer = signedIn;

        // when
        renderSync(viewer);

        // then
        expect(FakeWebSocket.instances).toHaveLength(1);
    });

    it("closes the socket when it unmounts", () => {
        // given
        const { unmount } = renderSync();
        const socket = lastSocket();

        // when
        unmount();

        // then
        expect(socket.closeCount).toBe(1);
    });

    it("counts an incoming notification exactly once", () => {
        // given
        const { queryClient } = renderSync();

        // when
        emit({ type: "notification", data: { id: 4, type: "chat_mention" } });

        // then
        expect(queryClient.getQueryData(queryKeys.notifications.unreadCount())).toEqual({ count: 1 });
    });

    it("hands the notification to the desktop notifier and plays the sound", () => {
        // given
        const { showNotification, playSound } = renderSync();

        // when
        emit({ type: "notification", data: { id: 4, type: "chat_mention" } });

        // then
        expect(showNotification).toHaveBeenCalledWith({ id: 4, type: "chat_mention" });
        expect(playSound).toHaveBeenCalledOnce();
    });

    it("updates the signed in user when their own role changes", () => {
        // given
        const { queryClient } = renderSync();
        queryClient.setQueryData(queryKeys.auth.me(), signedIn);

        // when
        emit({ type: "role_changed", data: { user_id: "user-1", role: "moderator" } });

        // then
        expect(queryClient.getQueryData(queryKeys.auth.me())).toEqual(
            expect.objectContaining({ id: "user-1", role: "moderator" }),
        );
    });

    it("stores the chat unread total that the server sent", () => {
        // given
        const { queryClient } = renderSync();

        // when
        emit({ type: "chat_unread_bumped", data: { total: 7 } });

        // then
        expect(queryClient.getQueryData(queryKeys.chat.unreadCount())).toEqual({ count: 7 });
    });

    it("stores the live games count that the server sent", () => {
        // given
        const { queryClient } = renderSync();

        // when
        emit({ type: "live_games_count", data: { count: 3 } });

        // then
        expect(queryClient.getQueryData(queryKeys.gameRoom.live())).toEqual({ rooms: [], total: 3 });
    });

    it("refetches the signed in viewer when their permissions change", () => {
        // given
        const { queryClient } = renderSync();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emit({ type: "permissions_changed", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.auth.me() });
    });

    it("refetches the chatbot list when a chatbot is created, edited or deleted", () => {
        // given
        const { queryClient } = renderSync();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emit({ type: "chatbots_changed", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chatbots.all });
    });

    it("refetches the live streams when a stream goes live", () => {
        // given
        const { queryClient } = renderSync();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emit({ type: "stream_live", data: { id: "stream-1" } });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.streams.live() });
    });

    it("asks for fresh site info as soon as the socket opens", () => {
        // given
        const { queryClient } = renderSync();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        openSocket();

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.auth.siteInfo() });
    });

    it("stops reacting once it is unmounted", () => {
        // given
        const { queryClient, unmount } = renderSync();
        const socket = lastSocket();

        // when
        unmount();
        act(() => {
            socket.onmessage?.({ data: JSON.stringify({ type: "chat_unread_bumped", data: { total: 7 } }) });
        });

        // then
        expect(queryClient.getQueryData(queryKeys.chat.unreadCount())).toBeUndefined();
    });
});
