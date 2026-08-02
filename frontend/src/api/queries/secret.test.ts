import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { SecretDetailResponse, SecretListResponse, SecretSummary } from "../../types/api";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getSecret, listSecrets } from "../endpoints";
import { useSecret, useSecretList } from "./secret";

vi.mock("../endpoints", () => ({
    getSecret: vi.fn(),
    listSecrets: vi.fn(),
}));

const mockedGetSecret = vi.mocked(getSecret);
const mockedListSecrets = vi.mocked(listSecrets);

function makeSummary(id: string): SecretSummary {
    return {
        id,
        title: `secret ${id}`,
        description: "",
        total_pieces: 3,
        solved: false,
        viewer_progress: 0,
        comment_count: 0,
    };
}

function makeList(secrets: SecretSummary[]): SecretListResponse {
    return { secrets, solvers_leaderboard: [] };
}

function makeDetail(id: string): SecretDetailResponse {
    return { ...makeSummary(id), riddle: "the seventh sacrifice", leaderboard: [], comments: [] };
}

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedListSecrets.mockResolvedValue(makeList([makeSummary("sec-1")]));
    mockedGetSecret.mockResolvedValue(makeDetail("sec-1"));
});

describe("useSecretList", () => {
    it("keys the list query under the secrets namespace", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useSecretList(), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["secrets", "list"]);
        expect(mockedListSecrets).toHaveBeenCalledOnce();
    });

    it("reports no data while the list is loading", () => {
        // given
        mockedListSecrets.mockReturnValue(new Promise<SecretListResponse>(() => {}));

        // when
        const { result } = renderHook(() => useSecretList(), { wrapper: providerWrapper() });

        // then
        expect(result.current.data).toBeNull();
        expect(result.current.loading).toBe(true);
    });

    it("returns the whole response once the list settles", async () => {
        // given
        const response = makeList([makeSummary("sec-1"), makeSummary("sec-2")]);
        mockedListSecrets.mockResolvedValue(response);

        // when
        const { result } = renderHook(() => useSecretList(), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.data).toEqual(response);
    });

    it("fetches the list again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useSecretList(), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedListSecrets).toHaveBeenCalledTimes(2);
    });
});

describe("useSecret", () => {
    it("keys the detail query by the secret id", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useSecret("sec-4"), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["secrets", "detail", "sec-4"]);
        expect(mockedGetSecret).toHaveBeenCalledWith("sec-4");
    });

    it("does not ask the server for a secret without an id", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useSecret(""), { wrapper });

        // then
        expect(mockedGetSecret).not.toHaveBeenCalled();
        expect(result.current.data).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("returns the whole response once the secret settles", async () => {
        // given
        const detail = makeDetail("sec-2");
        mockedGetSecret.mockResolvedValue(detail);

        // when
        const { result } = renderHook(() => useSecret("sec-2"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.data).toEqual(detail);
    });

    it("fetches the secret again when refresh is called", async () => {
        // given
        const { result } = renderHook(() => useSecret("sec-1"), { wrapper: providerWrapper() });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mockedGetSecret).toHaveBeenCalledTimes(2);
    });
});
