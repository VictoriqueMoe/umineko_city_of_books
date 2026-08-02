import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { CreateResponsePayload, CreateTheoryPayload } from "../../types/api";
import {
    useCreateResponse,
    useCreateTheory,
    useDeleteResponse,
    useDeleteTheory,
    useUpdateTheory,
    useVoteResponse,
    useVoteTheory,
} from "./theory";

const mocks = vi.hoisted(() => ({
    createResponse: vi.fn(),
    createTheory: vi.fn(),
    deleteResponse: vi.fn(),
    deleteTheory: vi.fn(),
    updateTheory: vi.fn(),
    voteResponse: vi.fn(),
    voteTheory: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const theoryId = "55555555-5555-5555-5555-555555555555";
const responseId = "66666666-6666-6666-6666-666666666666";

const theoryPayload: CreateTheoryPayload = {
    title: "the culprit is on the island",
    body: "seventeenth person or not",
    episode: 1,
    series: "umineko",
    evidence: [],
};

const responsePayload: CreateResponsePayload = {
    side: "without_love",
    body: "the chain was intact",
    evidence: [],
};

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { result, queryClient, invalidate };
}

beforeEach(() => {
    mocks.createResponse.mockResolvedValue({ id: responseId });
    mocks.createTheory.mockResolvedValue({ id: theoryId });
    mocks.deleteResponse.mockResolvedValue(undefined);
    mocks.deleteTheory.mockResolvedValue(undefined);
    mocks.updateTheory.mockResolvedValue({ status: "ok" });
    mocks.voteResponse.mockResolvedValue(undefined);
    mocks.voteTheory.mockResolvedValue(undefined);
});

describe("useCreateTheory", () => {
    it("sends the theory payload untouched and returns the new id", async () => {
        // given
        const { result } = setup(() => useCreateTheory());

        // when
        act(() => {
            result.current.mutate(theoryPayload);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createTheory).toHaveBeenCalledWith(theoryPayload);
        expect(result.current.data).toEqual({ id: theoryId });
    });

    it("refreshes every cached theory once the theory is stored", async () => {
        // given
        const { result, invalidate } = setup(() => useCreateTheory());

        // when
        act(() => {
            result.current.mutate(theoryPayload);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidate).toHaveBeenCalledTimes(1);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory"] });
    });

    it("leaves the cached theories untouched when creation fails", async () => {
        // given
        mocks.createTheory.mockRejectedValue(new Error("the red truth denies it"));
        const { result, invalidate } = setup(() => useCreateTheory());

        // when
        act(() => {
            result.current.mutate(theoryPayload);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUpdateTheory", () => {
    it("edits the theory it was built for", async () => {
        // given
        const { result } = setup(() => useUpdateTheory(theoryId));

        // when
        act(() => {
            result.current.mutate({ ...theoryPayload, title: "a revised claim" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.updateTheory).toHaveBeenCalledWith(theoryId, { ...theoryPayload, title: "a revised claim" });
    });

    it("refreshes both the single theory and the theory lists", async () => {
        // given
        const { result, invalidate } = setup(() => useUpdateTheory(theoryId));

        // when
        act(() => {
            result.current.mutate(theoryPayload);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidate).toHaveBeenCalledTimes(2);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory", "detail", theoryId] });
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory"] });
    });
});

describe("useDeleteTheory", () => {
    it("deletes the theory it is given and refreshes the theory lists", async () => {
        // given
        const { result, invalidate } = setup(() => useDeleteTheory());

        // when
        act(() => {
            result.current.mutate(theoryId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.deleteTheory).toHaveBeenCalledWith(theoryId);
        expect(invalidate).toHaveBeenCalledTimes(1);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory"] });
    });

    it("leaves the cached theories untouched when the deletion is refused", async () => {
        // given
        mocks.deleteTheory.mockRejectedValue(new Error("forbidden"));
        const { result, invalidate } = setup(() => useDeleteTheory());

        // when
        act(() => {
            result.current.mutate(theoryId);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useVoteTheory", () => {
    it("votes on the theory it was built for and refreshes only that theory", async () => {
        // given
        const { result, invalidate } = setup(() => useVoteTheory(theoryId));

        // when
        act(() => {
            result.current.mutate(-1);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.voteTheory).toHaveBeenCalledWith(theoryId, -1);
        expect(invalidate).toHaveBeenCalledTimes(1);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory", "detail", theoryId] });
    });
});

describe("useCreateResponse", () => {
    it("posts the response against the theory it was built for", async () => {
        // given
        const { result } = setup(() => useCreateResponse(theoryId));

        // when
        act(() => {
            result.current.mutate(responsePayload);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createResponse).toHaveBeenCalledWith(theoryId, responsePayload);
        expect(result.current.data).toEqual({ id: responseId });
    });

    it("refreshes only the theory the response belongs to", async () => {
        // given
        const { result, invalidate } = setup(() => useCreateResponse(theoryId));

        // when
        act(() => {
            result.current.mutate({ ...responsePayload, parent_id: responseId });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.createResponse).toHaveBeenCalledWith(theoryId, { ...responsePayload, parent_id: responseId });
        expect(invalidate).toHaveBeenCalledTimes(1);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory", "detail", theoryId] });
    });
});

describe("useDeleteResponse", () => {
    it("deletes by the response id while refreshing the theory it was built for", async () => {
        // given
        const { result, invalidate } = setup(() => useDeleteResponse(theoryId));

        // when
        act(() => {
            result.current.mutate(responseId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.deleteResponse).toHaveBeenCalledWith(responseId);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory", "detail", theoryId] });
    });

    it("leaves the cached theory untouched when the deletion is refused", async () => {
        // given
        mocks.deleteResponse.mockRejectedValue(new Error("forbidden"));
        const { result, invalidate } = setup(() => useDeleteResponse(theoryId));

        // when
        act(() => {
            result.current.mutate(responseId);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useVoteResponse", () => {
    it("votes on the response in the payload rather than on the theory", async () => {
        // given
        const { result, invalidate } = setup(() => useVoteResponse(theoryId));

        // when
        act(() => {
            result.current.mutate({ responseId, value: 1 });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.voteResponse).toHaveBeenCalledWith(responseId, 1);
        expect(invalidate).toHaveBeenCalledTimes(1);
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["theory", "detail", theoryId] });
    });
});
