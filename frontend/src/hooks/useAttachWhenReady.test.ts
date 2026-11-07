import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useAttachWhenReady } from "./useAttachWhenReady";
import type { StagedAttachment, UseStagedMediaResult } from "./useStagedMedia";

function staged(overrides: Partial<StagedAttachment> = {}): StagedAttachment {
    return {
        key: "key-1",
        file: new File(["x"], "notes.pdf"),
        isSpoiler: false,
        state: "ready",
        percent: 100,
        uploadId: "upload-1",
        error: null,
        ...overrides,
    };
}

function stagedMedia(items: StagedAttachment[]): UseStagedMediaResult {
    return {
        items,
        files: items.map(item => item.file),
        spoilers: items.map(item => item.isSpoiler),
        progress: [],
        refs: [],
        ready: true,
        busy: false,
        add: vi.fn(),
        remove: vi.fn(),
        reorder: vi.fn(),
        toggleSpoiler: vi.fn(),
        forget: vi.fn(),
        fail: vi.fn(),
        reset: vi.fn(),
    };
}

describe("useAttachWhenReady", () => {
    it("attaches each ready upload, hands over the result and lets the attachment go", async () => {
        // given
        const media = stagedMedia([staged({ isSpoiler: true })]);
        const attach = vi.fn().mockResolvedValue({ id: 3 });
        const onAttached = vi.fn();

        // when
        renderHook(() => useAttachWhenReady(media, attach, onAttached));

        // then
        await waitFor(() => expect(media.forget).toHaveBeenCalledExactlyOnceWith("key-1"));
        expect(attach).toHaveBeenCalledExactlyOnceWith({ upload_id: "upload-1", is_spoiler: true });
        expect(onAttached).toHaveBeenCalledExactlyOnceWith({ id: 3 });
    });

    it("leaves uploads alone until the server has finished with them", () => {
        // given
        const media = stagedMedia([
            staged({ key: "uploading", state: "uploading", uploadId: null }),
            staged({ key: "processing", state: "processing" }),
            staged({ key: "failed", state: "failed" }),
        ]);
        const attach = vi.fn().mockResolvedValue({});

        // when
        renderHook(() => useAttachWhenReady(media, attach, vi.fn()));

        // then
        expect(attach).not.toHaveBeenCalled();
    });

    it("attaches an upload only once however often it re-renders", async () => {
        // given
        let finish: (value: unknown) => void = () => {};
        const attach = vi.fn(() => new Promise(resolve => (finish = resolve)));
        const { rerender } = renderHook(({ items }) => useAttachWhenReady(stagedMedia(items), attach, vi.fn()), {
            initialProps: { items: [staged()] },
        });

        // when
        rerender({ items: [staged()] });
        finish({});

        // then
        await waitFor(() => expect(attach).toHaveBeenCalledOnce());
    });

    it("marks the attachment failed with the reason when attaching is refused", async () => {
        // given
        const media = stagedMedia([staged()]);
        const attach = vi.fn().mockRejectedValue(new Error("the mystery is sealed"));
        const onAttached = vi.fn();

        // when
        renderHook(() => useAttachWhenReady(media, attach, onAttached));

        // then
        await waitFor(() => expect(media.fail).toHaveBeenCalledExactlyOnceWith("key-1", "the mystery is sealed"));
        expect(media.forget).not.toHaveBeenCalled();
        expect(onAttached).not.toHaveBeenCalled();
    });
});
