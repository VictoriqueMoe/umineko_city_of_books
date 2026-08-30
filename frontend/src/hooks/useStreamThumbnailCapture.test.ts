import { act, renderHook } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import type { Room } from "livekit-client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import {
    THUMBNAIL_UPLOAD_FAILED,
    useStreamThumbnailCapture,
    type UseStreamThumbnailCaptureOptions,
} from "./useStreamThumbnailCapture";

const mocks = vi.hoisted(() => ({
    uploadStreamThumbnail: vi.fn(),
}));

vi.mock("../api/endpoints/stream", () => ({
    getStreamViewerToken: vi.fn(),
    joinStreamChat: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
    uploadStreamThumbnail: mocks.uploadStreamThumbnail,
}));

const { uploadStreamThumbnail } = mocks;

const room = { name: "room-1" } as unknown as Room;

function stubCanvas(): void {
    Object.defineProperty(HTMLCanvasElement.prototype, "getContext", {
        configurable: true,
        writable: true,
        value: () => ({ drawImage: () => {} }),
    });
    Object.defineProperty(HTMLCanvasElement.prototype, "toBlob", {
        configurable: true,
        writable: true,
        value: (callback: (blob: Blob) => void) => callback(new Blob(["frame"])),
    });
}

function makeStage(videoWidth = 1920, videoHeight = 1080): HTMLDivElement {
    const stage = document.createElement("div");
    const video = document.createElement("video");
    Object.defineProperty(video, "videoWidth", { configurable: true, value: videoWidth });
    Object.defineProperty(video, "videoHeight", { configurable: true, value: videoHeight });
    stage.appendChild(video);

    return stage;
}

let queryClient: QueryClient;

function mount(overrides: Partial<UseStreamThumbnailCaptureOptions> = {}) {
    const stage = makeStage();
    const options: UseStreamThumbnailCaptureOptions = {
        streamId: "stream-1",
        isOwnStream: true,
        isLive: true,
        room,
        stageRef: { current: stage },
        ...overrides,
    };

    const view = renderHook(() => useStreamThumbnailCapture(options), {
        wrapper: providerWrapper({ queryClient }),
    });

    return { ...view, stage };
}

beforeEach(() => {
    queryClient = createTestQueryClient();
    uploadStreamThumbnail.mockReset();
    uploadStreamThumbnail.mockResolvedValue(undefined);
    stubCanvas();
    vi.useFakeTimers();
});

afterEach(() => {
    vi.useRealTimers();
});

describe("useStreamThumbnailCapture", () => {
    it("sends a thumbnail from the streamer's own preview", async () => {
        // given
        mount();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(9000);
        });

        // then
        expect(uploadStreamThumbnail).toHaveBeenCalledWith("stream-1", expect.any(Blob));
    });

    it("keeps sending one every fifty seconds", async () => {
        // given
        mount();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(9000);
        });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(50000);
        });

        // then
        expect(uploadStreamThumbnail).toHaveBeenCalledTimes(2);
    });

    it("never sends a thumbnail from somebody watching another player's stream", async () => {
        // given
        mount({ isOwnStream: false });

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(uploadStreamThumbnail).not.toHaveBeenCalled();
    });

    it("never sends a thumbnail while the stream is offline", async () => {
        // given
        mount({ isLive: false });

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(uploadStreamThumbnail).not.toHaveBeenCalled();
    });

    it("never sends a thumbnail before the room is up", async () => {
        // given
        mount({ room: null });

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(uploadStreamThumbnail).not.toHaveBeenCalled();
    });

    it("sends nothing when the stage holds no video", async () => {
        // given
        mount({ stageRef: { current: document.createElement("div") } });

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(uploadStreamThumbnail).not.toHaveBeenCalled();
    });

    it("sends nothing while the video has no picture yet", async () => {
        // given
        mount({ stageRef: { current: makeStage(0, 0) } });

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(uploadStreamThumbnail).not.toHaveBeenCalled();
    });

    it("sends nothing while the tab is in the background", async () => {
        // given
        Object.defineProperty(document, "visibilityState", { configurable: true, value: "hidden" });
        mount();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(uploadStreamThumbnail).not.toHaveBeenCalled();
        Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
    });

    it("stops capturing once the streamer leaves the page", async () => {
        // given
        const view = mount();

        // when
        view.unmount();
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(uploadStreamThumbnail).not.toHaveBeenCalled();
    });

    it("says nothing while the thumbnails are going through", async () => {
        // given
        const view = mount();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(9000);
        });

        // then
        expect(view.result.current.lastError).toBe("");
    });

    it("tells the streamer their thumbnail is not updating", async () => {
        // given
        uploadStreamThumbnail.mockRejectedValue(new Error("the ingress refused it"));
        const view = mount();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(9000);
        });

        await act(async () => {
            await vi.advanceTimersByTimeAsync(10);
        });

        // then
        expect(view.result.current.lastError).toBe(`${THUMBNAIL_UPLOAD_FAILED} the ingress refused it`);
    });
});
