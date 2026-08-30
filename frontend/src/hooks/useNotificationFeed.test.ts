import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import type * as BusModule from "../api/realtime/bus";
import { providerWrapper } from "../test-utils/render";
import { makeWSHarness, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import type { Notification } from "../types/api";
import type { NotificationPage } from "../domain/notifications";
import { useNotificationFeed } from "./useNotificationFeed";

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));
const { useNotificationsQuery } = vi.hoisted(() => ({ useNotificationsQuery: vi.fn() }));

vi.mock("./queries/notification", () => ({ useNotifications: useNotificationsQuery }));

vi.mock("../api/realtime/pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

vi.mock("../api/realtime/bus", async importOriginal => {
    const actual = await importOriginal<typeof BusModule>();

    return {
        ...actual,
        subscribe: (names: RealtimeTestNames, handler: BusModule.RealtimeEventHandler) =>
            holder.ws.subscribe(names, handler),
    };
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

const liked = makeNotification({ id: 1 });
const followed = makeNotification({ id: 2, type: "new_follower", actor: ange, read: true });
const artLiked = makeNotification({ id: 3, type: "art_liked", reference_type: "art" });

function pageKey(limit: number) {
    return queryKeys.notifications.list({ limit, offset: 0 });
}

interface MountOptions {
    notifications?: Notification[];
    total?: number;
    loading?: boolean;
    seed?: Notification[];
    markRead?: (id: number) => Promise<void>;
    markAllRead?: () => Promise<void>;
}

function mount(options: MountOptions = {}) {
    const notifications = options.notifications ?? [];
    useNotificationsQuery.mockReturnValue({
        notifications,
        total: options.total ?? notifications.length,
        loading: options.loading ?? false,
        refresh: vi.fn(() => Promise.resolve(undefined)),
    });

    const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
    });
    if (options.seed) {
        queryClient.setQueryData<NotificationPage>(pageKey(50), {
            notifications: options.seed,
            total: options.seed.length,
        });
    }

    const markRead = vi.fn(options.markRead ?? (() => Promise.resolve()));
    const markAllRead = vi.fn(options.markAllRead ?? (() => Promise.resolve()));

    const rendered = renderHook(() => useNotificationFeed(), {
        wrapper: providerWrapper({ queryClient, notification: { markRead, markAllRead } }),
    });

    return { ...rendered, queryClient, markRead, markAllRead };
}

function cachedPage(queryClient: QueryClient, limit = 50): NotificationPage | undefined {
    return queryClient.getQueryData<NotificationPage>(pageKey(limit));
}

function arrive(notif: Notification): void {
    holder.ws.emit({ type: "notification", data: notif });
}

describe("useNotificationFeed", () => {
    beforeEach(() => {
        holder.ws = makeWSHarness();
    });

    it("asks for the first fifty notifications", () => {
        // given
        const options: MountOptions = {};

        // when
        mount(options);

        // then
        expect(useNotificationsQuery).toHaveBeenLastCalledWith(50, 0);
    });

    it("hands the query's answer straight through", () => {
        // given
        const options: MountOptions = { notifications: [liked, followed], total: 80, loading: true };

        // when
        const { result } = mount(options);

        // then
        expect(result.current.notifications).toEqual([liked, followed]);
        expect(result.current.total).toBe(80);
        expect(result.current.loading).toBe(true);
    });

    it("says there is more to see while the page is shorter than the total", () => {
        // given
        const options: MountOptions = { notifications: [liked], total: 80 };

        // when
        const { result } = mount(options);

        // then
        expect(result.current.hasMore).toBe(true);
    });

    it("says there is nothing more once the whole inbox is on screen", () => {
        // given
        const options: MountOptions = { notifications: [liked], total: 1 };

        // when
        const { result } = mount(options);

        // then
        expect(result.current.hasMore).toBe(false);
    });

    it("grows the page by fifty every time the reader asks for more", () => {
        // given
        const { result } = mount({ notifications: [liked], total: 300 });

        // when
        act(() => {
            result.current.loadMore();
        });
        act(() => {
            result.current.loadMore();
        });

        // then
        expect(useNotificationsQuery).toHaveBeenLastCalledWith(150, 0);
    });

    it("drops a live notification straight into the cached page", () => {
        // given
        const { queryClient } = mount({ notifications: [liked], seed: [liked] });

        // when
        arrive(artLiked);

        // then
        expect(cachedPage(queryClient)?.notifications.map(n => n.id)).toEqual([3, 1]);
        expect(cachedPage(queryClient)?.total).toBe(2);
    });

    it("refuses to list the same live notification twice", () => {
        // given
        const { queryClient } = mount({ notifications: [liked], seed: [liked] });

        // when
        arrive(liked);

        // then
        expect(cachedPage(queryClient)?.notifications).toHaveLength(1);
        expect(cachedPage(queryClient)?.total).toBe(1);
    });

    it("leaves a page it has never cached alone", () => {
        // given
        const { queryClient } = mount({ notifications: [liked] });

        // when
        arrive(artLiked);

        // then
        expect(cachedPage(queryClient)).toBeUndefined();
    });

    it("writes an arrival into the page the reader is actually on after loading more", () => {
        // given
        const { result, queryClient } = mount({ notifications: [liked], total: 300 });
        queryClient.setQueryData<NotificationPage>(pageKey(100), { notifications: [liked], total: 1 });
        act(() => {
            result.current.loadMore();
        });

        // when
        arrive(artLiked);

        // then
        expect(cachedPage(queryClient, 100)?.notifications.map(n => n.id)).toEqual([3, 1]);
    });

    it("subscribes once and keeps that subscription through a page change", () => {
        // given
        const { result } = mount({ notifications: [liked], total: 300 });

        // when
        act(() => {
            result.current.loadMore();
        });

        // then
        expect(holder.ws.subscribe).toHaveBeenCalledOnce();
    });

    it("marks a notification read on the server and in the cached page", async () => {
        // given
        const { result, queryClient, markRead } = mount({ notifications: [liked], seed: [liked] });

        // when
        await act(async () => {
            await result.current.markRead(liked);
        });

        // then
        expect(markRead).toHaveBeenCalledWith(1);
        expect(cachedPage(queryClient)?.notifications[0].read).toBe(true);
    });

    it("leaves the page unread when the server refuses", async () => {
        // given
        const { result, queryClient } = mount({
            notifications: [liked],
            seed: [liked],
            markRead: () => Promise.reject(new Error("the servants are asleep")),
        });

        // when
        await act(async () => {
            await result.current.markRead(liked);
        });

        // then
        expect(cachedPage(queryClient)?.notifications[0].read).toBe(false);
        expect(result.current.markingId).toBeNull();
    });

    it("names the notification it is still marking", async () => {
        // given
        const { result } = mount({
            notifications: [liked],
            seed: [liked],
            markRead: () => new Promise<void>(() => {}),
        });

        // when
        act(() => {
            result.current.markRead(liked).catch(() => {});
        });

        // then
        await waitFor(() => {
            expect(result.current.markingId).toBe(1);
        });
    });

    it("does not ask the server again for a notification already read", async () => {
        // given
        const { result, markRead } = mount({ notifications: [followed], seed: [followed] });

        // when
        await act(async () => {
            await result.current.markRead(followed);
        });

        // then
        expect(markRead).not.toHaveBeenCalled();
    });

    it("marks the whole page read once the server agrees", async () => {
        // given
        const { result, queryClient, markAllRead } = mount({
            notifications: [liked, artLiked],
            seed: [liked, artLiked],
        });

        // when
        await act(async () => {
            await result.current.markAllRead();
        });

        // then
        expect(markAllRead).toHaveBeenCalledOnce();
        expect(cachedPage(queryClient)?.notifications.every(n => n.read)).toBe(true);
    });

    it("leaves the page unread when marking everything read fails", async () => {
        // given
        const { result, queryClient } = mount({
            notifications: [liked, artLiked],
            seed: [liked, artLiked],
            markAllRead: () => Promise.reject(new Error("the servants are asleep")),
        });

        // when
        await act(async () => {
            await result.current.markAllRead();
        });

        // then
        expect(cachedPage(queryClient)?.notifications.every(n => !n.read)).toBe(true);
    });
});
