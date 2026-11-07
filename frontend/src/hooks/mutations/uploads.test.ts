import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makePendingUpload } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useBeginUpload, useRemoveUpload, useUploadStatuses } from "./uploads";

const endpoints = vi.hoisted(() => ({
    beginUpload: vi.fn(),
    getUploadStatuses: vi.fn(),
    removeUpload: vi.fn(),
}));

vi.mock("../../api/endpoints/uploads", () => endpoints);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { result, invalidate };
}

beforeEach(() => {
    endpoints.beginUpload.mockResolvedValue(makePendingUpload());
    endpoints.getUploadStatuses.mockResolvedValue([makePendingUpload()]);
    endpoints.removeUpload.mockResolvedValue(undefined);
});

describe("useBeginUpload", () => {
    it("starts an upload of the kind it was built for with the progress callback and abort signal", async () => {
        // given
        const { result, invalidate } = setup(() => useBeginUpload("document"));
        const file = new File(["x"], "notes.pdf");
        const onProgress = vi.fn();
        const signal = new AbortController().signal;

        // when
        let upload: unknown;
        await act(async () => {
            upload = await result.current.mutateAsync({ file, onProgress, signal });
        });

        // then
        expect(endpoints.beginUpload).toHaveBeenCalledExactlyOnceWith("document", file, onProgress, signal);
        expect(upload).toEqual(makePendingUpload());
        expect(invalidate).not.toHaveBeenCalled();
    });
});

describe("useUploadStatuses", () => {
    it("asks for the state of the uploads it is given", async () => {
        // given
        const { result } = setup(() => useUploadStatuses());

        // when
        let uploads: unknown;
        await act(async () => {
            uploads = await result.current.mutateAsync(["upload-1"]);
        });

        // then
        expect(endpoints.getUploadStatuses).toHaveBeenCalledWith(["upload-1"], expect.anything());
        expect(uploads).toEqual([makePendingUpload()]);
    });
});

describe("useRemoveUpload", () => {
    it("discards the upload it is given", async () => {
        // given
        const { result } = setup(() => useRemoveUpload());

        // when
        await act(async () => {
            await result.current.mutateAsync("upload-1");
        });

        // then
        expect(endpoints.removeUpload).toHaveBeenCalledWith("upload-1", expect.anything());
    });
});
