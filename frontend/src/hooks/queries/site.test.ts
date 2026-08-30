import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeSiteInfo } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useRules, useSiteInfoQuery, useStaff } from "./site";

const endpoints = vi.hoisted(() => ({
    getRules: vi.fn(),
    getSiteInfo: vi.fn(),
    getStaff: vi.fn(),
}));

vi.mock("../../api/endpoints/site", () => endpoints);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { ...rendered, queryClient };
}

function firstKey(queryClient: QueryClient): readonly unknown[] {
    return queryClient.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    endpoints.getSiteInfo.mockResolvedValue(makeSiteInfo());
    endpoints.getStaff.mockResolvedValue([]);
    endpoints.getRules.mockResolvedValue({ page: "chat", rules: "" });
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

describe("useRules", () => {
    it("returns the rules body for the requested page", async () => {
        // given
        endpoints.getRules.mockResolvedValue({ page: "chat", rules: "Be kind to the furniture" });
        const client = createTestQueryClient();

        // when
        const { result } = renderHook(() => useRules("chat"), { wrapper: providerWrapper({ queryClient: client }) });

        // then
        await waitFor(() => expect(result.current.rules).toBe("Be kind to the furniture"));
        expect(endpoints.getRules).toHaveBeenCalledWith("chat");
        expect(client.getQueryData(["rules", "chat"])).toBeDefined();
    });

    it("does not fetch when no page is named", () => {
        // given
        endpoints.getRules.mockResolvedValue({ page: "chat", rules: "Be kind to the furniture" });

        // when
        const { result } = renderHook(() => useRules(""), { wrapper: providerWrapper() });

        // then
        expect(endpoints.getRules).not.toHaveBeenCalled();
        expect(result.current.rules).toBe("");
    });
});
