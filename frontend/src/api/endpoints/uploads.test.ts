import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./uploads";
import {
    deleteMock,
    fetchMock,
    lastFormData,
    postFormDataMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

describe("beginUpload", () => {
    it.each([
        { kind: "media" as const, path: "/uploads/media" },
        { kind: "document" as const, path: "/uploads/files" },
    ])("sends a $kind file on its own to $path with the progress callback and abort signal", async ({ kind, path }) => {
        // given
        const file = new File(["x"], "notes.pdf");
        const onProgress = vi.fn();
        const signal = new AbortController().signal;

        // when
        await api.beginUpload(kind, file, onProgress, signal);

        // then
        expect(postFormDataMock).toHaveBeenCalledExactlyOnceWith(path, expect.any(FormData), onProgress, signal);
        expect([...lastFormData().keys()]).toEqual(["file"]);
        expect(lastFormData().get("file")).toBe(file);
    });
});

describe("getUploadStatuses", () => {
    it("returns the uploads the server still holds", async () => {
        // given
        const upload = { id: "upload-1", media_type: "image", filename: "a.png", status: "ready" };
        fetchMock.mockResolvedValue({ uploads: [upload] });

        // when
        const uploads = await api.getUploadStatuses(["upload-1"]);

        // then
        expect(uploads).toEqual([upload]);
    });

    it("returns an empty list when the server holds none of them", async () => {
        // given
        fetchMock.mockResolvedValue({ uploads: null });

        // when
        const uploads = await api.getUploadStatuses(["upload-1"]);

        // then
        expect(uploads).toEqual([]);
    });
});

describe("upload requests", () => {
    const cases: RequestCase[] = [
        {
            name: "getUploadStatuses asks for every id in one request",
            call: () => api.getUploadStatuses(["upload-1", "upload-2"]),
            transport: fetchMock,
            request: ["/uploads?ids=upload-1,upload-2"],
        },
        {
            name: "removeUpload deletes the upload by its id",
            call: () => api.removeUpload("upload-1"),
            transport: deleteMock,
            request: ["/uploads/upload-1"],
        },
    ];

    runRequestCases(cases);
});
