import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserProfile } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useCreateReport, useResolveReport } from "./report";

const endpoints = vi.hoisted(() => ({
    createReport: vi.fn(),
    resolveReport: vi.fn(),
}));

vi.mock("../../api/endpoints/report", () => endpoints);

const signedInUser = makeUser({ id: "99999999-9999-9999-9999-999999999999" });

function setup<T>(hook: () => T, user: UserProfile | null = null) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient, user }) });

    return { result, invalidate };
}

beforeEach(() => {
    endpoints.createReport.mockResolvedValue(undefined);
    endpoints.resolveReport.mockResolvedValue(undefined);
});

describe("useResolveReport", () => {
    it("resolves a report with the moderator comment and refreshes the report list", async () => {
        // given
        const { result, invalidate } = setup(() => useResolveReport());

        // when
        await act(async () => {
            await result.current.mutateAsync({ id: 7, comment: "handled" });
        });

        // then
        expect(endpoints.resolveReport).toHaveBeenCalledWith(7, "handled");
        expect(invalidate).toHaveBeenCalledWith({ queryKey: ["admin", "reports"] });
    });
});

describe("useCreateReport", () => {
    it("forwards the target, the reason and the context of the report in order", async () => {
        // given
        const { result } = setup(() => useCreateReport(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync({
                targetType: "post",
                targetId: "p-1",
                reason: "unfavourable behaviour",
                contextId: "room-7",
            });
        });

        // then
        expect(endpoints.createReport).toHaveBeenCalledWith("post", "p-1", "unfavourable behaviour", "room-7");
    });

    it("passes an undefined context when the report has none", async () => {
        // given
        const { result } = setup(() => useCreateReport(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync({ targetType: "user", targetId: "u-2", reason: "spam" });
        });

        // then
        expect(endpoints.createReport).toHaveBeenCalledWith("user", "u-2", "spam", undefined);
    });

    it("invalidates nothing because a report changes nothing the viewer can see", async () => {
        // given
        const { result, invalidate } = setup(() => useCreateReport(), signedInUser);

        // when
        await act(async () => {
            await result.current.mutateAsync({ targetType: "post", targetId: "p-1", reason: "spam" });
        });

        // then
        expect(invalidate).not.toHaveBeenCalled();
    });

    it("surfaces a rejected report to the caller", async () => {
        // given
        endpoints.createReport.mockRejectedValue(new Error("report limit reached"));
        const { result } = setup(() => useCreateReport(), signedInUser);

        // when
        const attempt = act(async () => {
            await result.current.mutateAsync({ targetType: "post", targetId: "p-1", reason: "spam" });
        });

        // then
        await expect(attempt).rejects.toThrow("report limit reached");
    });
});
