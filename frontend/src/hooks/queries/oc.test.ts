import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { OC, OCDetail, OCListResponse, OCSummary } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getOC, listOCs } from "../../api/endpoints/oc";
import { listUserOCs, listUserOCSummaries } from "../../api/endpoints/user";
import { useOwnOCSync } from "../../api/realtime/sync/useOwnOCSync";
import { useOC, useOCList, useUserOCs, useUserOCSummaries } from "./oc";

vi.mock("../../api/endpoints/oc", () => ({
    getOC: vi.fn(),
    listOCs: vi.fn(),
}));

vi.mock("../../api/endpoints/user", () => ({
    listUserOCs: vi.fn(),
    listUserOCSummaries: vi.fn(),
}));

vi.mock("../../api/realtime/sync/useOwnOCSync", () => ({ useOwnOCSync: vi.fn() }));

const mockedGetOC = vi.mocked(getOC);
const mockedListOCs = vi.mocked(listOCs);
const mockedListUserOCs = vi.mocked(listUserOCs);
const mockedListUserOCSummaries = vi.mocked(listUserOCSummaries);
const mockedUseOwnOCSync = vi.mocked(useOwnOCSync);

const viewerId = "11111111-1111-1111-1111-111111111111";

function makeOC(id: string): OC {
    return { id, name: `oc ${id}` } as unknown as OC;
}

function makeOCList(ocs: OC[], total: number): OCListResponse {
    return { ocs, total, limit: 20, offset: 0 };
}

function makeSummary(id: string, name: string): OCSummary {
    return { id, name, series: "umineko" };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedListOCs.mockResolvedValue(makeOCList([makeOC("oc-1")], 1));
    mockedGetOC.mockResolvedValue({ id: "oc-1", name: "oc oc-1" } as unknown as OCDetail);
    mockedListUserOCs.mockResolvedValue(makeOCList([makeOC("oc-1")], 1));
    mockedListUserOCSummaries.mockResolvedValue([makeSummary("b", "Beatrice")]);
});

describe("useOCList", () => {
    it("keys the feed query by the params it was handed", async () => {
        // given
        const qc = createTestQueryClient();
        const params = { sort: "new", series: "umineko", limit: 20, offset: 0 };

        // when
        const { result } = renderHook(() => useOCList(params), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["oc", "feed", params]);
        expect(mockedListOCs).toHaveBeenCalledWith(params);
    });

    it("reports empty values while the feed is loading", () => {
        // given
        mockedListOCs.mockReturnValue(new Promise<OCListResponse>(() => {}));

        // when
        const { result } = renderHook(() => useOCList({}), { wrapper: providerWrapper() });

        // then
        expect(result.current.ocs).toEqual([]);
        expect(result.current.total).toBe(0);
        expect(result.current.loading).toBe(true);
    });

    it("exposes the ocs and the total once the response arrives", async () => {
        // given
        mockedListOCs.mockResolvedValue(makeOCList([makeOC("oc-1"), makeOC("oc-2")], 7));

        // when
        const { result } = renderHook(() => useOCList({ crack: true }), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.ocs).toHaveLength(2);
        expect(result.current.total).toBe(7);
    });
});

describe("useOC", () => {
    it("keys the detail query by the oc id", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useOC("oc-5"), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.oc).not.toBeNull());

        // then
        expect(firstKey(qc)).toEqual(["oc", "detail", "oc-5"]);
        expect(mockedGetOC).toHaveBeenCalledWith("oc-5");
    });

    it("does not ask the server for an oc without an id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useOC(""), { wrapper });

        // then
        expect(mockedGetOC).not.toHaveBeenCalled();
        expect(result.current.oc).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("fetches the oc again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useOC("oc-1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetOC).toHaveBeenCalledTimes(2);
    });
});

describe("useUserOCs", () => {
    it("keys the list query by the owner id", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserOCs(viewerId), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["oc", "userList", viewerId]);
        expect(mockedListUserOCs).toHaveBeenCalledWith(viewerId);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserOCs(""), { wrapper });

        // then
        expect(mockedListUserOCs).not.toHaveBeenCalled();
        expect(result.current.ocs).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useUserOCSummaries", () => {
    it("keys the summaries query by the owner id", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useUserOCSummaries(viewerId), {
            wrapper: providerWrapper({ queryClient: qc }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["oc", "userSummaries", viewerId]);
        expect(mockedListUserOCSummaries).toHaveBeenCalledWith(viewerId);
    });

    it("does not ask the server without an owner id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useUserOCSummaries(""), { wrapper });

        // then
        expect(mockedListUserOCSummaries).not.toHaveBeenCalled();
        expect(result.current.summaries).toEqual([]);
    });

    it("hands the owner and the viewer to the own-oc sync", async () => {
        // given
        const wrapper = providerWrapper({ queryClient: createTestQueryClient() });

        // when
        const { result } = renderHook(() => useUserOCSummaries(viewerId, viewerId), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedUseOwnOCSync).toHaveBeenCalledWith(viewerId, viewerId);
    });

    it("still mounts the own-oc sync for a list the viewer does not own", async () => {
        // given
        const wrapper = providerWrapper({ queryClient: createTestQueryClient() });

        // when
        const { result } = renderHook(() => useUserOCSummaries(viewerId, "someone-else"), { wrapper });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(mockedUseOwnOCSync).toHaveBeenCalledWith(viewerId, "someone-else");
    });

    it("exposes the summaries the server returned", async () => {
        // given
        mockedListUserOCSummaries.mockResolvedValue([makeSummary("b", "Beatrice"), makeSummary("v", "Virgilia")]);
        const wrapper = providerWrapper({ queryClient: createTestQueryClient() });

        // when
        const { result } = renderHook(() => useUserOCSummaries(viewerId, viewerId), { wrapper });

        // then
        await waitFor(() => expect(result.current.summaries.map(item => item.name)).toEqual(["Beatrice", "Virgilia"]));
    });
});
