import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeSiteInfo, makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useMe, useSiteInfoQuery, useStaff } from "./auth";

const endpoints = vi.hoisted(() => ({
    getMe: vi.fn(),
    getSiteInfo: vi.fn(),
    getStaff: vi.fn(),
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
    endpoints.getMe.mockResolvedValue(makeUser());
    endpoints.getSiteInfo.mockResolvedValue(makeSiteInfo());
    endpoints.getStaff.mockResolvedValue([]);
});

describe("useMe", () => {
    it("exposes the signed in profile under the auth me key", async () => {
        // given
        const me = makeUser({ username: "battler" });
        endpoints.getMe.mockResolvedValue(me);

        // when
        const { result, queryClient } = setup(() => useMe());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["auth", "me"]);
        expect(result.current.me).toEqual(me);
    });

    it("reports no profile while the session is still being checked", async () => {
        // given
        const { result } = setup(() => useMe());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.me).toBeNull();
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("reports no profile for a signed out visitor", async () => {
        // given
        endpoints.getMe.mockResolvedValue(null);

        // when
        const { result } = setup(() => useMe());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.me).toBeNull();
    });

    it("refetches the session when refresh is called", async () => {
        // given
        const { result } = setup(() => useMe());
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await result.current.refresh();

        // then
        expect(endpoints.getMe).toHaveBeenCalledTimes(2);
    });
});

describe("useSiteInfoQuery", () => {
    it("exposes the site info under the site info key", async () => {
        // given
        const info = makeSiteInfo({ site_name: "City of Books" });
        endpoints.getSiteInfo.mockResolvedValue(info);

        // when
        const { result, queryClient } = setup(() => useSiteInfoQuery());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["site-info"]);
        expect(result.current.siteInfo).toEqual(info);
    });

    it("reports no site info before the request settles", async () => {
        // given
        const { result } = setup(() => useSiteInfoQuery());

        // when
        const initial = result.current;

        // then
        expect(initial.siteInfo).toBeNull();
        expect(initial.dataUpdatedAt).toBe(0);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("stamps the moment the site info last arrived", async () => {
        // given
        const { result } = setup(() => useSiteInfoQuery());

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(result.current.dataUpdatedAt).toBeGreaterThan(0);
    });
});

describe("useStaff", () => {
    it("exposes the staff list under the staff key", async () => {
        // given
        endpoints.getStaff.mockResolvedValue([{ id: "u-1", username: "kanon" }]);

        // when
        const { result, queryClient } = setup(() => useStaff());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["staff"]);
        expect(result.current.staff).toEqual([{ id: "u-1", username: "kanon" }]);
    });

    it("reports an empty staff list before the request settles", async () => {
        // given
        const { result } = setup(() => useStaff());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.staff).toEqual([]);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });
});
