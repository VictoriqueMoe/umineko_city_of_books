import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useAnnouncement, useAnnouncementList, useLatestAnnouncement } from "./announcement";

const endpoints = vi.hoisted(() => ({
    getAnnouncement: vi.fn(),
    getLatestAnnouncement: vi.fn(),
    listAnnouncements: vi.fn(),
}));

vi.mock("../endpoints", () => endpoints);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { ...rendered, queryClient };
}

function firstKey(queryClient: QueryClient): readonly unknown[] {
    return queryClient.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    endpoints.getAnnouncement.mockResolvedValue({ id: "a-1", title: "A new arc" });
    endpoints.getLatestAnnouncement.mockResolvedValue({ announcement: { id: "a-9" } });
    endpoints.listAnnouncements.mockResolvedValue({ announcements: [], total: 0 });
});

describe("useAnnouncementList", () => {
    it("asks for twenty announcements from the top when no window is given", async () => {
        // given
        const { result, queryClient } = setup(() => useAnnouncementList());

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listAnnouncements).toHaveBeenCalledWith(20, 0);
        expect(firstKey(queryClient)).toEqual(["announcements", "list", { limit: 20, offset: 0 }]);
    });

    it("forwards a chosen page window and keys the cache entry by it", async () => {
        // given
        endpoints.listAnnouncements.mockResolvedValue({ announcements: [{ id: "a-1" }], total: 31 });

        // when
        const { result, queryClient } = setup(() => useAnnouncementList(5, 10));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listAnnouncements).toHaveBeenCalledWith(5, 10);
        expect(firstKey(queryClient)).toEqual(["announcements", "list", { limit: 5, offset: 10 }]);
        expect(result.current.announcements).toEqual([{ id: "a-1" }]);
        expect(result.current.total).toBe(31);
    });

    it("reports an empty list and a zero total while the request is in flight", async () => {
        // given
        const { result } = setup(() => useAnnouncementList());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.announcements).toEqual([]);
        expect(initial.total).toBe(0);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("defaults the list and the total when the response carries neither", async () => {
        // given
        endpoints.listAnnouncements.mockResolvedValue({});

        // when
        const { result } = setup(() => useAnnouncementList());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.announcements).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useAnnouncement", () => {
    it("loads the announcement behind the id it was given", async () => {
        // given
        const { result, queryClient } = setup(() => useAnnouncement("a-1"));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.getAnnouncement).toHaveBeenCalledWith("a-1");
        expect(firstKey(queryClient)).toEqual(["announcements", "detail", "a-1"]);
        expect(result.current.announcement).toEqual({ id: "a-1", title: "A new arc" });
    });

    it("stays idle and reports no announcement when the id is empty", () => {
        // given
        const { result } = setup(() => useAnnouncement(""));

        // when
        const current = result.current;

        // then
        expect(endpoints.getAnnouncement).not.toHaveBeenCalled();
        expect(current.announcement).toBeNull();
        expect(current.loading).toBe(false);
    });

    it("exposes a refresh handle for the detail query", async () => {
        // given
        const { result } = setup(() => useAnnouncement("a-1"));
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await result.current.refresh();

        // then
        expect(endpoints.getAnnouncement).toHaveBeenCalledTimes(2);
    });
});

describe("useLatestAnnouncement", () => {
    it("unwraps the announcement out of the latest response", async () => {
        // given
        const { result, queryClient } = setup(() => useLatestAnnouncement());

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(queryClient)).toEqual(["announcements", "latest"]);
        expect(result.current.announcement).toEqual({ id: "a-9" });
    });

    it("reports no announcement when the site has never published one", async () => {
        // given
        endpoints.getLatestAnnouncement.mockResolvedValue({ announcement: null });

        // when
        const { result } = setup(() => useLatestAnnouncement());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.announcement).toBeNull();
    });

    it("reports no announcement before the request settles", async () => {
        // given
        const { result } = setup(() => useLatestAnnouncement());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.announcement).toBeNull();
        await waitFor(() => expect(result.current.loading).toBe(false));
    });
});
