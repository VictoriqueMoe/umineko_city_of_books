import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { HomeActivityResponse, SidebarActivityResponse, SidebarLastVisitedResponse } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getHomeActivity, getSidebarActivity, getSidebarLastVisited } from "../endpoints";
import { useHomeActivity, useSidebarActivity, useSidebarLastVisited } from "./sidebar";

vi.mock("../endpoints", () => ({
    getHomeActivity: vi.fn(),
    getSidebarActivity: vi.fn(),
    getSidebarLastVisited: vi.fn(),
}));

const mockedGetHomeActivity = vi.mocked(getHomeActivity);
const mockedGetSidebarActivity = vi.mocked(getSidebarActivity);
const mockedGetSidebarLastVisited = vi.mocked(getSidebarLastVisited);

function makeHomeActivity(onlineCount: number): HomeActivityResponse {
    return {
        online_count: onlineCount,
        recent_activity: [],
        recent_members: [],
        public_rooms: [],
        corner_activity: [],
    };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedGetHomeActivity.mockResolvedValue(makeHomeActivity(3));
    mockedGetSidebarActivity.mockResolvedValue({ activity: { theories: "2026-01-01T00:00:00Z" } });
    mockedGetSidebarLastVisited.mockResolvedValue({ visited: { theories: "2026-01-01T00:00:00Z" } });
});

describe("useHomeActivity", () => {
    it("keys the home activity query under the home namespace", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useHomeActivity(), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["home", "activity"]);
        expect(mockedGetHomeActivity).toHaveBeenCalledOnce();
    });

    it("reports no data while the activity is loading", () => {
        // given
        mockedGetHomeActivity.mockReturnValue(new Promise<HomeActivityResponse>(() => {}));

        // when
        const { result } = renderHook(() => useHomeActivity(), { wrapper: providerWrapper() });

        // then
        expect(result.current.data).toBeNull();
        expect(result.current.loading).toBe(true);
    });

    it("returns the whole response once the activity settles", async () => {
        // given
        const response = makeHomeActivity(11);
        mockedGetHomeActivity.mockResolvedValue(response);

        // when
        const { result } = renderHook(() => useHomeActivity(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.data).toEqual(response);
    });

    it("fetches the activity even for a signed out visitor", async () => {
        // given
        const wrapper = providerWrapper({ user: null });

        // when
        const { result } = renderHook(() => useHomeActivity(), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedGetHomeActivity).toHaveBeenCalledOnce();
    });
});

describe("useSidebarActivity", () => {
    it("keys the sidebar activity query under the sidebar namespace", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useSidebarActivity(), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["sidebar", "activity"]);
        expect(mockedGetSidebarActivity).toHaveBeenCalledOnce();
    });

    it("returns the activity map once it settles", async () => {
        // given
        const response: SidebarActivityResponse = { activity: { art: "2026-02-02T00:00:00Z" } };
        mockedGetSidebarActivity.mockResolvedValue(response);

        // when
        const { result } = renderHook(() => useSidebarActivity(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.data).toEqual(response);
    });
});

describe("useSidebarLastVisited", () => {
    it("keys the last visited query under the sidebar namespace", async () => {
        // given
        const qc = createTestQueryClient();
        const wrapper = providerWrapper({ queryClient: qc, user: makeUser() });

        // when
        const { result } = renderHook(() => useSidebarLastVisited(), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["sidebar", "last-visited"]);
        expect(mockedGetSidebarLastVisited).toHaveBeenCalledOnce();
    });

    it("does not ask the server when nobody is signed in", () => {
        // given
        const wrapper = providerWrapper({ user: null });

        // when
        const { result } = renderHook(() => useSidebarLastVisited(), { wrapper });

        // then
        expect(mockedGetSidebarLastVisited).not.toHaveBeenCalled();
        expect(result.current.data).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("returns the visited map once it settles for a member", async () => {
        // given
        const response: SidebarLastVisitedResponse = { visited: { chat: "2026-03-03T00:00:00Z" } };
        mockedGetSidebarLastVisited.mockResolvedValue(response);
        const wrapper = providerWrapper({ user: makeUser() });

        // when
        const { result } = renderHook(() => useSidebarLastVisited(), { wrapper });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.data).toEqual(response);
    });

    it("fetches the visited map again when refresh is called", async () => {
        // given
        const wrapper = providerWrapper({ user: makeUser() });
        const { result } = renderHook(() => useSidebarLastVisited(), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetSidebarLastVisited).toHaveBeenCalledTimes(2);
    });
});
