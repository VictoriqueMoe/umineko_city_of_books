import { act, renderHook } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FakeWebSocket } from "../../../test-utils/ws";
import { queryKeys } from "../../queryKeys";
import type * as SocketModule from "../socket";
import type * as SyncModule from "./useSiteInfoSync";

const { getAuthToken, isNativeApp } = vi.hoisted(() => ({
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
}));

vi.mock("../../authToken", () => ({ getAuthToken }));
vi.mock("../../../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

const LEADERBOARD_EVENTS = [
    "top_detective_changed",
    "top_gm_changed",
    "top_chess_changed",
    "top_checkers_changed",
    "top_othello_changed",
    "top_minesweeper_changed",
    "vanity_roles_changed",
    "rules_page_changed",
] as const;

const MIN_INTERVAL = 2000;

const START = new Date("2026-08-28T12:00:00Z").getTime();

let sync: typeof SyncModule;
let socket: typeof SocketModule;

function makeClient(): QueryClient {
    return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
}

function wrapperFor(client: QueryClient) {
    return function Wrapper({ children }: PropsWithChildren) {
        return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
    };
}

function lastSocket(): FakeWebSocket {
    const instance = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!instance) {
        throw new Error("no websocket was opened");
    }

    return instance;
}

function connect(): void {
    act(() => {
        socket.openRealtimeSocket("user-1");
        lastSocket().onopen?.();
    });
}

function reconnect(): void {
    act(() => {
        socket.closeRealtimeSocket("user-1");
        socket.openRealtimeSocket("user-1");
        lastSocket().onopen?.();
    });
}

function receive(message: { type: string; data?: unknown }): void {
    act(() => {
        lastSocket().onmessage?.({ data: JSON.stringify(message) });
    });
}

function mountSync(client: QueryClient): void {
    renderHook(() => sync.useSiteInfoSync(), { wrapper: wrapperFor(client) });
}

beforeEach(async () => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    getAuthToken.mockReturnValue(null);
    isNativeApp.mockReturnValue(false);
    vi.useFakeTimers();
    vi.setSystemTime(START);
    vi.resetModules();
    socket = await import("../socket");
    sync = await import("./useSiteInfoSync");
});

afterEach(() => {
    vi.useRealTimers();
});

describe("useSiteInfoSync", () => {
    it("asks for fresh site info when a leaderboard changes", () => {
        // given
        const client = makeClient();
        mountSync(client);
        connect();
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");
        vi.setSystemTime(START + MIN_INTERVAL);

        // when
        receive({ type: "top_detective_changed", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.auth.siteInfo() });
    });

    it("collapses a burst of eight leaderboard events into one refetch", () => {
        // given
        const client = makeClient();
        mountSync(client);
        connect();
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");
        vi.setSystemTime(START + MIN_INTERVAL);

        // when
        for (const type of LEADERBOARD_EVENTS) {
            receive({ type, data: {} });
        }

        // then
        expect(invalidateQueries).toHaveBeenCalledTimes(1);
    });

    it("refetches for every one of the eight events when the clock moves between them", () => {
        // given
        const client = makeClient();
        mountSync(client);
        connect();
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");

        // when
        let elapsed = 0;
        for (const type of LEADERBOARD_EVENTS) {
            elapsed += MIN_INTERVAL;
            vi.setSystemTime(START + elapsed);
            receive({ type, data: {} });
        }

        // then
        expect(invalidateQueries).toHaveBeenCalledTimes(LEADERBOARD_EVENTS.length);
    });

    it("refetches again once the throttle window has passed", () => {
        // given
        const client = makeClient();
        mountSync(client);
        connect();
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");
        vi.setSystemTime(START + MIN_INTERVAL);
        receive({ type: "top_gm_changed", data: {} });

        // when
        vi.setSystemTime(START + MIN_INTERVAL * 2);
        receive({ type: "top_gm_changed", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledTimes(2);
    });

    it("leaves site info alone when the cache was filled moments ago", () => {
        // given
        const client = makeClient();
        mountSync(client);
        connect();
        vi.setSystemTime(START + MIN_INTERVAL);
        client.setQueryData(queryKeys.auth.siteInfo(), { site_name: "City of Books" });
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");

        // when
        vi.setSystemTime(START + MIN_INTERVAL + 1999);
        receive({ type: "rules_page_changed", data: {} });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });

    it("refetches when the socket comes back", () => {
        // given
        const client = makeClient();
        mountSync(client);
        connect();
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");
        vi.setSystemTime(START + MIN_INTERVAL);

        // when
        reconnect();

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.auth.siteInfo() });
    });

    it("counts a reconnect and the events that follow it as one refetch", () => {
        // given
        const client = makeClient();
        mountSync(client);
        connect();
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");
        vi.setSystemTime(START + MIN_INTERVAL);

        // when
        reconnect();
        for (const type of LEADERBOARD_EVENTS) {
            receive({ type, data: {} });
        }

        // then
        expect(invalidateQueries).toHaveBeenCalledTimes(1);
    });

    it("stops refetching once the consumer unmounts", () => {
        // given
        const client = makeClient();
        const view = renderHook(() => sync.useSiteInfoSync(), { wrapper: wrapperFor(client) });
        connect();
        const invalidateQueries = vi.spyOn(client, "invalidateQueries");
        vi.setSystemTime(START + MIN_INTERVAL);

        // when
        view.unmount();
        receive({ type: "vanity_roles_changed", data: {} });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});
