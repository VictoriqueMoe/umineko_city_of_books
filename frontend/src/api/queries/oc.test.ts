import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { OC, OCDetail, OCListResponse, OCSummary, WSMessage } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getOC, listOCs, listUserOCs, listUserOCSummaries } from "../endpoints";
import { useOC, useOCList, useUserOCs, useUserOCSummaries } from "./oc";

vi.mock("../endpoints", () => ({
    getOC: vi.fn(),
    listOCs: vi.fn(),
    listUserOCs: vi.fn(),
    listUserOCSummaries: vi.fn(),
}));

const mockedGetOC = vi.mocked(getOC);
const mockedListOCs = vi.mocked(listOCs);
const mockedListUserOCs = vi.mocked(listUserOCs);
const mockedListUserOCSummaries = vi.mocked(listUserOCSummaries);

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

interface SummarySetup {
    result: { current: ReturnType<typeof useUserOCSummaries> };
    emit: (message: WSMessage) => Promise<void>;
    unsubscribe: ReturnType<typeof vi.fn>;
    listeners: number;
    unmount: () => void;
}

function setupSummaries(userId: string, currentUserId?: string): SummarySetup {
    const handlers: ((message: WSMessage) => void)[] = [];
    const unsubscribe = vi.fn();
    const addWSListener = vi.fn((handler: (message: WSMessage) => void) => {
        handlers.push(handler);
        return unsubscribe;
    });

    const wrapper = providerWrapper({ queryClient: createTestQueryClient(), notification: { addWSListener } });
    const { result, unmount } = renderHook(() => useUserOCSummaries(userId, currentUserId), { wrapper });

    return {
        result,
        emit: async message => {
            await act(async () => {
                for (const handler of handlers) {
                    handler(message);
                }
            });
        },
        unsubscribe,
        listeners: handlers.length,
        unmount,
    };
}

function changed(action: string, oc: OCSummary): WSMessage {
    return { type: "user_ocs_changed", data: { action, oc } };
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

    it("subscribes to live updates only when the owner is the viewer", () => {
        // given
        const own = setupSummaries(viewerId, viewerId);

        // when
        const other = setupSummaries(viewerId, "someone-else");

        // then
        expect(own.listeners).toBe(1);
        expect(other.listeners).toBe(0);
    });

    it("appends a newly created oc in alphabetical order", async () => {
        // given
        mockedListUserOCSummaries.mockResolvedValue([makeSummary("b", "Beatrice"), makeSummary("v", "Virgilia")]);
        const { result, emit } = setupSummaries(viewerId, viewerId);
        await waitFor(() => expect(result.current.summaries).toHaveLength(2));

        // when
        await emit(changed("created", makeSummary("l", "Lambdadelta")));

        // then
        await waitFor(() =>
            expect(result.current.summaries.map(item => item.name)).toEqual(["Beatrice", "Lambdadelta", "Virgilia"]),
        );
    });

    it("ignores a created oc that is already in the list", async () => {
        // given
        const { result, emit } = setupSummaries(viewerId, viewerId);
        await waitFor(() => expect(result.current.summaries).toHaveLength(1));

        // when
        await emit(changed("created", makeSummary("b", "Beatrice renamed")));

        // then
        expect(result.current.summaries).toEqual([makeSummary("b", "Beatrice")]);
    });

    it("replaces the matching oc when one is updated", async () => {
        // given
        const { result, emit } = setupSummaries(viewerId, viewerId);
        await waitFor(() => expect(result.current.summaries).toHaveLength(1));

        // when
        await emit(changed("updated", makeSummary("b", "Beato")));

        // then
        await waitFor(() => expect(result.current.summaries).toEqual([makeSummary("b", "Beato")]));
    });

    it("drops the matching oc when one is deleted", async () => {
        // given
        const { result, emit } = setupSummaries(viewerId, viewerId);
        await waitFor(() => expect(result.current.summaries).toHaveLength(1));

        // when
        await emit(changed("deleted", makeSummary("b", "Beatrice")));

        // then
        await waitFor(() => expect(result.current.summaries).toEqual([]));
    });

    it("leaves the list alone for an action it does not recognise", async () => {
        // given
        const { result, emit } = setupSummaries(viewerId, viewerId);
        await waitFor(() => expect(result.current.summaries).toHaveLength(1));

        // when
        await emit(changed("archived", makeSummary("b", "Beatrice")));

        // then
        expect(result.current.summaries).toEqual([makeSummary("b", "Beatrice")]);
    });

    it("ignores socket messages about anything other than the viewer's ocs", async () => {
        // given
        const { result, emit } = setupSummaries(viewerId, viewerId);
        await waitFor(() => expect(result.current.summaries).toHaveLength(1));

        // when
        await emit({ type: "notification", data: { action: "deleted", oc: makeSummary("b", "Beatrice") } });

        // then
        expect(result.current.summaries).toEqual([makeSummary("b", "Beatrice")]);
    });

    it("stops listening when the hook unmounts", () => {
        // given
        const { unsubscribe, unmount } = setupSummaries(viewerId, viewerId);

        // when
        unmount();

        // then
        expect(unsubscribe).toHaveBeenCalledOnce();
    });
});
