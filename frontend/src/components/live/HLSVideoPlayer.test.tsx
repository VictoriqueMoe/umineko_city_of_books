import { act, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { HLSVideoPlayer } from "./HLSVideoPlayer";

type Listener = (event: string, data: unknown) => void;

interface FatalError {
    fatal: boolean;
    type: string;
}

const { hls } = vi.hoisted(() => {
    const events = { MANIFEST_PARSED: "hlsManifestParsed", ERROR: "hlsError" };
    const errorTypes = { MEDIA_ERROR: "mediaError", NETWORK_ERROR: "networkError" };

    class FakeHls {
        static Events = events;
        static ErrorTypes = errorTypes;
        static isSupported = vi.fn(() => true);

        config: unknown;
        listeners = new Map<string, Listener[]>();
        loadSource = vi.fn();
        startLoad = vi.fn();
        attachMedia = vi.fn();
        recoverMediaError = vi.fn();
        destroy = vi.fn();

        constructor(config: unknown) {
            this.config = config;
            state.instances.push(this);
        }

        on(event: string, listener: Listener) {
            const current = this.listeners.get(event) ?? [];
            current.push(listener);
            this.listeners.set(event, current);
        }

        emit(event: string, data?: unknown) {
            for (const listener of this.listeners.get(event) ?? []) {
                listener(event, data);
            }
        }
    }

    const state = { instances: [] as FakeHls[], FakeHls, events, errorTypes };

    return { hls: state };
});

vi.mock("hls.js", () => ({ default: hls.FakeHls }));

const SRC = "https://stream.example/live/index.m3u8";

function only() {
    const instance = hls.instances[0];
    if (!instance) {
        throw new Error("expected an hls instance to have been created");
    }

    return instance;
}

function videoOf(container: HTMLElement): HTMLVideoElement {
    const video = container.querySelector("video");
    if (!video) {
        throw new Error("expected a video element to be rendered");
    }

    return video;
}

function fatal(type: string): FatalError {
    return { fatal: true, type };
}

describe("HLSVideoPlayer", () => {
    beforeEach(() => {
        hls.instances.length = 0;
        hls.FakeHls.isSupported.mockReturnValue(true);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("renders a video that plays inline on its own with controls", () => {
        // given
        const className = "player";

        // when
        const { container } = render(<HLSVideoPlayer src={SRC} className={className} />);

        // then
        const video = videoOf(container);
        expect(video).toHaveClass(className);
        expect(video).toHaveAttribute("controls");
        expect(video.autoplay).toBe(true);
        expect(video.playsInline).toBe(true);
    });

    it("loads the playlist and attaches the player to the video", () => {
        // given
        const src = SRC;

        // when
        const { container } = render(<HLSVideoPlayer src={src} />);

        // then
        expect(only().loadSource).toHaveBeenCalledWith(src);
        expect(only().attachMedia).toHaveBeenCalledWith(videoOf(container));
    });

    it("keeps the back buffer short so a long watch does not grow without end", () => {
        // given
        const src = SRC;

        // when
        render(<HLSVideoPlayer src={src} />);

        // then
        expect(only().config).toEqual({ backBufferLength: 30 });
    });

    it("starts playing as soon as the manifest has been parsed", async () => {
        // given
        const play = vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
        render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => only().emit(hls.events.MANIFEST_PARSED));

        // then
        await waitFor(() => expect(play).toHaveBeenCalledOnce());
    });

    it("falls back to a silent play when the browser refuses to autoplay with sound", async () => {
        // given
        const play = vi
            .spyOn(HTMLMediaElement.prototype, "play")
            .mockRejectedValueOnce(new Error("blocked"))
            .mockResolvedValue(undefined);
        const { container } = render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => only().emit(hls.events.MANIFEST_PARSED));

        // then
        await waitFor(() => expect(videoOf(container).muted).toBe(true));
        expect(play).toHaveBeenCalledTimes(2);
    });

    it("gives up quietly when even a silent play is refused", async () => {
        // given
        const play = vi.spyOn(HTMLMediaElement.prototype, "play").mockRejectedValue(new Error("blocked"));
        render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => only().emit(hls.events.MANIFEST_PARSED));

        // then
        await waitFor(() => expect(play).toHaveBeenCalledTimes(2));
    });

    it("leaves a passing hiccup alone", () => {
        // given
        render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => only().emit(hls.events.ERROR, { fatal: false, type: hls.errorTypes.NETWORK_ERROR }));

        // then
        expect(only().recoverMediaError).not.toHaveBeenCalled();
        expect(only().loadSource).toHaveBeenCalledOnce();
    });

    it("recovers in place from a fatal media error", () => {
        // given
        render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => only().emit(hls.events.ERROR, fatal(hls.errorTypes.MEDIA_ERROR)));

        // then
        expect(only().recoverMediaError).toHaveBeenCalledOnce();
        expect(only().loadSource).toHaveBeenCalledOnce();
    });

    it("reloads the playlist a couple of seconds after any other fatal error", () => {
        // given
        vi.useFakeTimers();
        render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => only().emit(hls.events.ERROR, fatal(hls.errorTypes.NETWORK_ERROR)));

        // then
        expect(only().loadSource).toHaveBeenCalledOnce();
        act(() => {
            vi.advanceTimersByTime(2000);
        });
        expect(only().loadSource).toHaveBeenCalledTimes(2);
        expect(only().startLoad).toHaveBeenCalledOnce();
    });

    it("only schedules one reload however many fatal errors pile up", () => {
        // given
        vi.useFakeTimers();
        render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => {
            only().emit(hls.events.ERROR, fatal(hls.errorTypes.NETWORK_ERROR));
            only().emit(hls.events.ERROR, fatal(hls.errorTypes.NETWORK_ERROR));
            vi.advanceTimersByTime(2000);
        });

        // then
        expect(only().startLoad).toHaveBeenCalledOnce();
    });

    it("throws the player away and cancels any pending reload when it goes", () => {
        // given
        vi.useFakeTimers();
        const { unmount } = render(<HLSVideoPlayer src={SRC} />);
        act(() => only().emit(hls.events.ERROR, fatal(hls.errorTypes.NETWORK_ERROR)));

        // when
        unmount();
        act(() => {
            vi.advanceTimersByTime(5000);
        });

        // then
        expect(only().destroy).toHaveBeenCalledOnce();
        expect(only().loadSource).toHaveBeenCalledOnce();
    });

    it("builds a fresh player when the source changes", () => {
        // given
        const { rerender } = render(<HLSVideoPlayer src={SRC} />);

        // when
        rerender(<HLSVideoPlayer src="https://stream.example/other/index.m3u8" />);

        // then
        expect(hls.instances).toHaveLength(2);
        expect(hls.instances[0].destroy).toHaveBeenCalledOnce();
        expect(hls.instances[1].loadSource).toHaveBeenCalledWith("https://stream.example/other/index.m3u8");
    });

    it("hands the stream to the browser when it can play hls by itself", async () => {
        // given
        hls.FakeHls.isSupported.mockReturnValue(false);
        vi.spyOn(HTMLMediaElement.prototype, "canPlayType").mockReturnValue("maybe");
        const play = vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
        const { container } = render(<HLSVideoPlayer src={SRC} />);

        // when
        act(() => {
            videoOf(container).dispatchEvent(new Event("loadedmetadata"));
        });

        // then
        expect(hls.instances).toHaveLength(0);
        expect(videoOf(container).src).toBe(SRC);
        await waitFor(() => expect(play).toHaveBeenCalledOnce());
    });

    it("stops listening for the browser's own playback once it goes away", () => {
        // given
        hls.FakeHls.isSupported.mockReturnValue(false);
        vi.spyOn(HTMLMediaElement.prototype, "canPlayType").mockReturnValue("maybe");
        const play = vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
        const { container, unmount } = render(<HLSVideoPlayer src={SRC} />);
        const video = videoOf(container);

        // when
        unmount();
        video.dispatchEvent(new Event("loadedmetadata"));

        // then
        expect(play).not.toHaveBeenCalled();
    });

    it("does nothing when the browser can neither run hls nor play it", () => {
        // given
        hls.FakeHls.isSupported.mockReturnValue(false);
        vi.spyOn(HTMLMediaElement.prototype, "canPlayType").mockReturnValue("");

        // when
        const { container } = render(<HLSVideoPlayer src={SRC} />);

        // then
        expect(hls.instances).toHaveLength(0);
        expect(videoOf(container).getAttribute("src")).toBeNull();
    });

    it("waits for a source before setting anything up", () => {
        // given
        const src = "";

        // when
        render(<HLSVideoPlayer src={src} />);

        // then
        expect(hls.instances).toHaveLength(0);
    });

    it("silences the video when it is asked to be muted", () => {
        // given
        const muted = true;

        // when
        const { container } = render(<HLSVideoPlayer src={SRC} muted={muted} />);

        // then
        expect(videoOf(container).muted).toBe(true);
    });

    it("tries the sound again on a new stream after an earlier silent fallback", async () => {
        // given
        const play = vi
            .spyOn(HTMLMediaElement.prototype, "play")
            .mockRejectedValueOnce(new Error("blocked"))
            .mockResolvedValue(undefined);
        const { container, rerender } = render(<HLSVideoPlayer src={SRC} muted={false} />);
        act(() => only().emit(hls.events.MANIFEST_PARSED));
        await waitFor(() => expect(videoOf(container).muted).toBe(true));

        // when
        rerender(<HLSVideoPlayer src="https://stream.example/other/index.m3u8" muted={false} />);
        act(() => hls.instances[1].emit(hls.events.MANIFEST_PARSED));

        // then
        await waitFor(() => expect(play).toHaveBeenCalledTimes(3));
        expect(videoOf(container).muted).toBe(false);
    });

    it("brings the sound back when muting is turned off again", () => {
        // given
        const { container, rerender } = render(<HLSVideoPlayer src={SRC} muted />);

        // when
        rerender(<HLSVideoPlayer src={SRC} muted={false} />);

        // then
        expect(videoOf(container).muted).toBe(false);
    });
});
