import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makePendingUpload } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import { emitRealtimeEvent, makeWSHarness, type WSHarness } from "../test-utils/ws";
import type { PendingUpload } from "../types/api";
import { useStagedMedia } from "./useStagedMedia";

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

const uploads = vi.hoisted(() => ({ begin: vi.fn(), remove: vi.fn(), statuses: vi.fn(), kinds: [] as string[] }));

vi.mock("../api/realtime/pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

vi.mock("./mutations/uploads", () => ({
    useBeginUpload: (kind: string) => {
        uploads.kinds.push(kind);

        return { mutateAsync: uploads.begin };
    },
    useRemoveUpload: () => ({ mutateAsync: uploads.remove }),
    useUploadStatuses: () => ({ mutateAsync: uploads.statuses }),
}));

type BeginInput = {
    file: File;
    signal: AbortSignal;
    onProgress: (sent: number, total: number) => void;
};

type Deferred = {
    input: BeginInput;
    resolve: (upload: PendingUpload) => void;
    reject: (err: unknown) => void;
};

function deferUploads(): Deferred[] {
    const started: Deferred[] = [];
    uploads.begin.mockImplementation(
        (input: BeginInput) =>
            new Promise<PendingUpload>((resolve, reject) => {
                started.push({ input, resolve, reject });
            }),
    );

    return started;
}

function image(name: string): File {
    return new File(["x"], name, { type: "image/png" });
}

function renderStaged(kind?: "media" | "document") {
    return renderHook(() => useStagedMedia(kind), { wrapper: providerWrapper() });
}

beforeEach(() => {
    holder.ws = makeWSHarness();
    uploads.kinds.length = 0;
    uploads.remove.mockResolvedValue(undefined);
    uploads.statuses.mockResolvedValue([]);
    uploads.begin.mockImplementation(({ file }: BeginInput) =>
        Promise.resolve(makePendingUpload({ id: `upload-${file.name}`, filename: file.name })),
    );
});

describe("useStagedMedia", () => {
    it("starts uploading every added file at once and reports each one's own progress", () => {
        // given
        const started = deferUploads();
        const { result } = renderStaged();

        // when
        act(() => {
            result.current.add([image("a.png"), image("b.png")]);
        });
        act(() => {
            started[1].input.onProgress(30, 100);
        });

        // then
        expect(started.map(entry => entry.input.file.name)).toEqual(["a.png", "b.png"]);
        expect(result.current.progress).toEqual([
            { state: "uploading", percent: 0, error: null },
            { state: "uploading", percent: 30, error: null },
        ]);
        expect(result.current.busy).toBe(true);
        expect(result.current.ready).toBe(false);
    });

    it("asks for the upload endpoint of the kind it was built for", () => {
        // when
        renderStaged("document");

        // then
        expect(uploads.kinds).toContain("document");
    });

    it("becomes ready with a reference per upload once the server has stored them", async () => {
        // given
        const { result } = renderStaged();

        // when
        act(() => {
            result.current.add([image("a.png"), image("b.png")]);
        });

        // then
        await waitFor(() => expect(result.current.ready).toBe(true));
        expect(result.current.busy).toBe(false);
        expect(result.current.refs).toEqual([
            { upload_id: "upload-a.png", is_spoiler: false },
            { upload_id: "upload-b.png", is_spoiler: false },
        ]);
    });

    it("waits on the server while an upload is processing and settles when it reports back", async () => {
        // given
        uploads.begin.mockResolvedValue(makePendingUpload({ id: "upload-1", status: "processing" }));
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("clip.mp4")]);
        });
        await waitFor(() => expect(result.current.items[0].state).toBe("processing"));
        expect(result.current.ready).toBe(false);
        expect(result.current.busy).toBe(true);

        // when
        emitRealtimeEvent({ type: "upload_ready", data: makePendingUpload({ id: "upload-1" }) });

        // then
        expect(result.current.items[0].state).toBe("ready");
        expect(result.current.ready).toBe(true);
    });

    it("marks an upload failed when the server reports it could not process it", async () => {
        // given
        uploads.begin.mockResolvedValue(makePendingUpload({ id: "upload-1", status: "processing" }));
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("clip.mp4")]);
        });
        await waitFor(() => expect(result.current.items[0].state).toBe("processing"));

        // when
        emitRealtimeEvent({ type: "upload_failed", data: makePendingUpload({ id: "upload-1", status: "failed" }) });

        // then
        expect(result.current.items[0].state).toBe("failed");
        expect(result.current.ready).toBe(false);
        expect(result.current.busy).toBe(false);
    });

    it("keeps a result the server pushed before the upload request itself returned", async () => {
        // given
        const started = deferUploads();
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("clip.mp4")]);
        });

        // when
        emitRealtimeEvent({ type: "upload_ready", data: makePendingUpload({ id: "upload-1" }) });
        await act(async () => {
            started[0].resolve(makePendingUpload({ id: "upload-1", status: "processing" }));
        });

        // then
        expect(result.current.items[0].state).toBe("ready");
    });

    it("ignores results pushed for uploads it does not hold", async () => {
        // given
        uploads.begin.mockResolvedValue(makePendingUpload({ id: "upload-1", status: "processing" }));
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("clip.mp4")]);
        });
        await waitFor(() => expect(result.current.items[0].state).toBe("processing"));

        // when
        emitRealtimeEvent({ type: "upload_ready", data: makePendingUpload({ id: "someone-elses" }) });

        // then
        expect(result.current.items[0].state).toBe("processing");
    });

    it("shows why an upload was refused", async () => {
        // given
        uploads.begin.mockRejectedValue(new Error("the file is cursed"));
        const { result } = renderStaged();

        // when
        act(() => {
            result.current.add([image("a.png")]);
        });

        // then
        await waitFor(() => expect(result.current.items[0].state).toBe("failed"));
        expect(result.current.items[0].error).toBe("the file is cursed");
        expect(result.current.busy).toBe(false);
    });

    it("cancels an upload still in flight when it is removed, without reporting a failure", async () => {
        // given
        const started = deferUploads();
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("a.png"), image("b.png")]);
        });

        // when
        act(() => {
            result.current.remove(0);
        });
        await act(async () => {
            started[0].reject(new DOMException("aborted", "AbortError"));
        });

        // then
        expect(started[0].input.signal.aborted).toBe(true);
        expect(started[1].input.signal.aborted).toBe(false);
        expect(result.current.files.map(file => file.name)).toEqual(["b.png"]);
        expect(uploads.remove).not.toHaveBeenCalled();
    });

    it("discards a finished upload on the server when it is removed", async () => {
        // given
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("a.png")]);
        });
        await waitFor(() => expect(result.current.ready).toBe(true));

        // when
        act(() => {
            result.current.remove(0);
        });

        // then
        expect(uploads.remove).toHaveBeenCalledExactlyOnceWith("upload-a.png");
        expect(result.current.items).toEqual([]);
    });

    it("cancels every upload still in flight when the composer goes away", () => {
        // given
        const started = deferUploads();
        const { result, unmount } = renderStaged();
        act(() => {
            result.current.add([image("a.png")]);
        });

        // when
        unmount();

        // then
        expect(started[0].input.signal.aborted).toBe(true);
    });

    it("keeps each spoiler flag with its own upload when the attachments are reordered", async () => {
        // given
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("a.png"), image("b.png")]);
        });
        await waitFor(() => expect(result.current.ready).toBe(true));

        // when
        act(() => {
            result.current.toggleSpoiler(1);
        });
        act(() => {
            result.current.reorder(1, 0);
        });

        // then
        expect(result.current.spoilers).toEqual([true, false]);
        expect(result.current.refs).toEqual([
            { upload_id: "upload-b.png", is_spoiler: true },
            { upload_id: "upload-a.png", is_spoiler: false },
        ]);
    });

    it("asks the server for the result of every upload still processing after the connection comes back", async () => {
        // given two uploads waiting on the server while the connection drops
        uploads.begin.mockImplementation(({ file }: BeginInput) =>
            Promise.resolve(makePendingUpload({ id: `upload-${file.name}`, status: "processing" })),
        );
        uploads.statuses.mockResolvedValue([makePendingUpload({ id: "upload-a.png", status: "ready" })]);
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("a.png"), image("b.png")]);
        });
        await waitFor(() => expect(result.current.items.map(item => item.state)).toEqual(["processing", "processing"]));

        // when
        holder.ws.reconnect();

        // then
        await waitFor(() => expect(result.current.items[0].state).toBe("ready"));
        expect(uploads.statuses).toHaveBeenCalledExactlyOnceWith(["upload-a.png", "upload-b.png"]);
        expect(result.current.items[1]).toMatchObject({ state: "failed", error: "This upload expired, add it again" });
    });

    it("does not ask the server for anything after a reconnect when nothing is processing", async () => {
        // given
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("a.png")]);
        });
        await waitFor(() => expect(result.current.ready).toBe(true));

        // when
        holder.ws.reconnect();

        // then
        expect(uploads.statuses).not.toHaveBeenCalled();
    });

    it("lets a caller fail, forget and clear attachments", async () => {
        // given
        const { result } = renderStaged();
        act(() => {
            result.current.add([image("a.png"), image("b.png")]);
        });
        await waitFor(() => expect(result.current.ready).toBe(true));
        const [first, second] = result.current.items;

        // when
        act(() => {
            result.current.fail(first.key, "the mystery is sealed");
        });
        const failed = result.current.items[0];
        act(() => {
            result.current.forget(second.key);
        });
        const afterForget = result.current.items.length;
        act(() => {
            result.current.reset();
        });

        // then
        expect(failed).toMatchObject({ state: "failed", error: "the mystery is sealed" });
        expect(afterForget).toBe(1);
        expect(result.current.items).toEqual([]);
        expect(uploads.remove).not.toHaveBeenCalled();
    });
});
