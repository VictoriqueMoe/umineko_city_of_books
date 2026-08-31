import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Room } from "livekit-client";
import { makeStream as makeLiveStream, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { LiveStream, UserProfile } from "../../types/api";
import { LiveWatchPage } from "./LiveWatchPage";

const mocks = vi.hoisted(() => ({
    getStream: vi.fn(),
    getStreamViewerToken: vi.fn(),
    uploadStreamThumbnail: vi.fn(),
    connectRoom: vi.fn(),
    disconnectRoom: vi.fn(),
    reportClientError: vi.fn(),
    useIsMobile: vi.fn(),
}));

vi.mock("../../api/endpoints/stream", () => ({
    listLiveStreams: vi.fn(),
    getStream: mocks.getStream,
    getMyStream: vi.fn(),
    getStreamCredentials: vi.fn(),
    getStreamViewerToken: mocks.getStreamViewerToken,
    joinStreamChat: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
    uploadStreamThumbnail: mocks.uploadStreamThumbnail,
}));

vi.mock("../../api/livekit/connect", () => ({
    connectRoom: mocks.connectRoom,
    disconnectRoom: mocks.disconnectRoom,
}));

vi.mock("../../api/telemetry", () => ({ reportClientError: mocks.reportClientError }));

vi.mock("@livekit/components-react", () => ({
    RoomContext: { Provider: (props: { children: React.ReactNode }) => <>{props.children}</> },
    RoomAudioRenderer: (props: { volume: number }) => <div data-testid="audio-renderer">{props.volume}</div>,
    StartAudio: (props: { label: string }) => <button type="button">{props.label}</button>,
}));

vi.mock("../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

vi.mock("./StreamChatPanel", () => ({
    StreamChatPanel: (props: { streamId: string; isLive: boolean; onPopOut?: () => void }) => (
        <div data-testid="stream-chat" data-live={String(props.isLive)}>
            {props.streamId}
            {props.onPopOut && (
                <button type="button" onClick={props.onPopOut}>
                    Pop out
                </button>
            )}
        </div>
    ),
}));

vi.mock("./MobileLiveView", () => ({
    MobileLiveView: (props: { stream: LiveStream }) => <div data-testid="mobile-view">{props.stream.title}</div>,
}));

vi.mock("./streamParts", () => ({
    StreamStage: () => <div data-testid="stream-stage" />,
    StreamUptime: (props: { startedAt?: string }) => <div data-testid="uptime">{props.startedAt}</div>,
    StreamViewers: () => <div data-testid="stream-viewers" />,
    ViewerCountReporter: () => null,
}));

vi.mock("../../components/live/HLSVideoPlayer", () => ({
    HLSVideoPlayer: (props: { src: string; muted?: boolean }) => (
        <div data-testid="hls-player" data-muted={String(Boolean(props.muted))}>
            {props.src}
        </div>
    ),
}));

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return makeLiveStream({
        userId: "streamer-1",
        title: "Reading Episode 4",
        startedAt: "2026-02-01T12:00:00Z",
        streamerUsername: "beatrice",
        ...overrides,
    });
}

function renderWatch(options: { user?: UserProfile | null; streamID?: string } = {}) {
    return renderWithProviders(<LiveWatchPage />, {
        user: options.user ?? null,
        route: `/live/${options.streamID ?? "stream-1"}`,
        path: "/live/:streamID",
    });
}

beforeEach(() => {
    mocks.useIsMobile.mockReturnValue(false);
    mocks.getStream.mockResolvedValue(makeStream());
    mocks.getStreamViewerToken.mockResolvedValue({ token: "tok", url: "wss://livekit.test" });
    mocks.uploadStreamThumbnail.mockResolvedValue(undefined);
    mocks.connectRoom.mockReset();
    mocks.disconnectRoom.mockReset();
    mocks.connectRoom.mockImplementation((options: { on?: { onConnected?: (room: Room) => void } }) => {
        const room = { name: "room-1" } as unknown as Room;
        options.on?.onConnected?.(room);

        return Promise.resolve(room);
    });
});

describe("LiveWatchPage loading and lookup", () => {
    it("waits while the stream is being looked up", () => {
        // given
        mocks.getStream.mockReturnValue(new Promise<LiveStream>(() => {}));

        // when
        renderWatch();

        // then
        expect(screen.getByText("Loading stream...")).toBeInTheDocument();
    });

    it("says the stream was not found when the lookup comes back empty", async () => {
        // given
        mocks.getStream.mockResolvedValue(null as unknown as LiveStream);

        // when
        renderWatch();

        // then
        expect(await screen.findByText("Stream not found.")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Back to live streams" })).toHaveAttribute("href", "/live");
    });

    it("hands the whole page to the mobile view on a small screen", async () => {
        // given
        mocks.useIsMobile.mockReturnValue(true);

        // when
        renderWatch();

        // then
        expect(await screen.findByTestId("mobile-view")).toHaveTextContent("Reading Episode 4");
        expect(screen.queryByTestId("stream-chat")).not.toBeInTheDocument();
    });
});

describe("LiveWatchPage stage", () => {
    it("says the stream is offline when it has stopped", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ status: "offline" }));

        // when
        renderWatch();

        // then
        expect(await screen.findByText("This stream is offline.")).toBeInTheDocument();
        expect(mocks.connectRoom).not.toHaveBeenCalled();
    });

    it("plays the live room once the connection is up", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream());

        // when
        renderWatch();

        // then
        expect(await screen.findByTestId("stream-stage")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Click to enable sound" })).toBeInTheDocument();
        expect(screen.getByRole("slider", { name: "Stream volume" })).toBeInTheDocument();
    });

    it("admits when the room could not be reached", async () => {
        // given
        mocks.getStreamViewerToken.mockRejectedValue(new Error("no token for you"));

        // when
        renderWatch();

        // then
        expect(await screen.findByText("Could not connect to this stream.")).toBeInTheDocument();
    });

    it("hides a streamer's own preview to save their upload", async () => {
        // given
        const user = makeUser({ id: "streamer-1" });

        // when
        renderWatch({ user });

        // then
        expect(await screen.findByText(/preview is hidden/)).toBeInTheDocument();
        expect(screen.queryByTestId("stream-stage")).not.toBeInTheDocument();
    });

    it("shows the streamer a muted preview when they ask for one", async () => {
        // given
        const pointer = userEvent.setup();
        renderWatch({ user: makeUser({ id: "streamer-1" }) });
        await screen.findByText(/preview is hidden/);

        // when
        await pointer.click(screen.getByRole("button", { name: "Show preview (muted)" }));

        // then
        expect(await screen.findByTestId("audio-renderer")).toHaveTextContent("0");
        expect(screen.queryByRole("slider", { name: "Stream volume" })).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Hide preview" })).toBeInTheDocument();
    });

    it("plays the smooth feed when the stream prefers hls", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ defaultMode: "hls", hlsUrl: "https://edge/s.m3u8" }));

        // when
        renderWatch();

        // then
        expect(await screen.findByTestId("hls-player")).toHaveTextContent("https://edge/s.m3u8");
    });

    it("falls back to the low latency room when the stream prefers hls but has no url", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ defaultMode: "hls", hlsUrl: undefined }));

        // when
        renderWatch();

        // then
        expect(await screen.findByTestId("stream-stage")).toBeInTheDocument();
        expect(screen.queryByTestId("hls-player")).not.toBeInTheDocument();
    });
});

describe("LiveWatchPage controls", () => {
    it("offers no quality choice when the stream has no smooth feed", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ hlsUrl: undefined }));

        // when
        renderWatch();

        // then
        await screen.findByTestId("stream-stage");
        expect(screen.queryByRole("button", { name: "Smooth" })).not.toBeInTheDocument();
    });

    it("lets the viewer switch to the smooth feed", async () => {
        // given
        const pointer = userEvent.setup();
        mocks.getStream.mockResolvedValue(makeStream({ hlsUrl: "https://edge/s.m3u8" }));
        renderWatch();
        await screen.findByTestId("stream-stage");

        // when
        await pointer.click(screen.getByRole("button", { name: "Smooth" }));

        // then
        expect(await screen.findByTestId("hls-player")).toBeInTheDocument();
    });

    it("lets the viewer switch back to the low latency feed", async () => {
        // given
        const pointer = userEvent.setup();
        mocks.getStream.mockResolvedValue(makeStream({ defaultMode: "hls", hlsUrl: "https://edge/s.m3u8" }));
        renderWatch();
        await screen.findByTestId("hls-player");

        // when
        await pointer.click(screen.getByRole("button", { name: "Low latency" }));

        // then
        expect(await screen.findByTestId("stream-stage")).toBeInTheDocument();
    });

    it("asks the browser for fullscreen on the stage", async () => {
        // given
        const pointer = userEvent.setup();
        const requestFullscreen = vi.fn(() => Promise.resolve());
        Object.defineProperty(Element.prototype, "requestFullscreen", {
            configurable: true,
            writable: true,
            value: requestFullscreen,
        });
        renderWatch();
        await screen.findByTestId("stream-stage");

        // when
        await pointer.click(screen.getByRole("button", { name: "Toggle fullscreen" }));

        // then
        expect(requestFullscreen).toHaveBeenCalledTimes(1);
    });

    it("leaves fullscreen again when the browser is already showing it", async () => {
        // given
        const pointer = userEvent.setup();
        const exitFullscreen = vi.fn(() => Promise.resolve());
        Object.defineProperty(document, "fullscreenElement", { configurable: true, value: document.body });
        Object.defineProperty(document, "exitFullscreen", { configurable: true, value: exitFullscreen });
        renderWatch();
        await screen.findByTestId("stream-stage");

        // when
        await pointer.click(screen.getByRole("button", { name: "Toggle fullscreen" }));

        // then
        expect(exitFullscreen).toHaveBeenCalledTimes(1);
        Object.defineProperty(document, "fullscreenElement", { configurable: true, value: null });
    });
});

describe("LiveWatchPage meta", () => {
    it("names the stream and links to the streamer and the directory", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ title: "Ciconia blind run" }));

        // when
        renderWatch();

        // then
        expect(await screen.findByRole("heading", { name: "Ciconia blind run" })).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/user/beatrice");
        expect(screen.getByRole("link", { name: /All live streams/ })).toHaveAttribute("href", "/live");
    });

    it("falls back to the username when the streamer has no display name", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ streamerDisplayName: "" }));

        // when
        renderWatch();

        // then
        expect(await screen.findByRole("link", { name: /beatrice/ })).toBeInTheDocument();
    });

    it("puts the stream's chat in the sidebar", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ id: "stream-42" }));

        // when
        renderWatch({ streamID: "stream-42" });

        // then
        const panel = await screen.findByTestId("stream-chat");
        expect(panel).toHaveTextContent("stream-42");
        expect(panel).toHaveAttribute("data-live", "true");
    });

    it("lists the viewers only once the room is connected", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ status: "offline" }));

        // when
        renderWatch();

        // then
        await screen.findByText("This stream is offline.");
        expect(screen.queryByTestId("stream-viewers")).not.toBeInTheDocument();
    });

    it("shows the uptime while the stream is live", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ startedAt: "2026-02-01T12:00:00Z" }));

        // when
        renderWatch();

        // then
        expect(await screen.findByTestId("uptime")).toHaveTextContent("2026-02-01T12:00:00Z");
    });
});

describe("LiveWatchPage popping the chat out", () => {
    const restoreLabel = "Chat is in its own window. Bring it back";

    function stubWindowOpen(opened: Partial<Window> | null) {
        const open = vi.fn(() => opened as Window | null);
        vi.stubGlobal("open", open);

        return open;
    }

    function makePopout() {
        return { focus: vi.fn(), close: vi.fn() };
    }

    afterEach(() => {
        vi.unstubAllGlobals();
    });

    it("opens the chat in a window of its own", async () => {
        // given
        const open = stubWindowOpen(makePopout());
        renderWatch({ streamID: "stream-7" });
        await screen.findByTestId("stream-chat");

        // when
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // then
        expect(open).toHaveBeenCalledWith(
            "/live/stream-7/chat",
            "stream-chat-stream-7",
            expect.stringContaining("popup=yes"),
        );
    });

    it("brings the popped out window to the front", async () => {
        // given
        const popout = makePopout();
        stubWindowOpen(popout);
        renderWatch();
        await screen.findByTestId("stream-chat");

        // when
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // then
        expect(popout.focus).toHaveBeenCalled();
    });

    it("gives the sidebar space back to the video once the chat is popped out", async () => {
        // given
        stubWindowOpen(makePopout());
        renderWatch();
        await screen.findByTestId("stream-chat");

        // when
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // then
        expect(screen.queryByTestId("stream-chat")).not.toBeInTheDocument();
        expect(screen.getByRole("button", { name: restoreLabel })).toBeInTheDocument();
    });

    it("leaves the chat where it is when the browser blocks the popup", async () => {
        // given
        stubWindowOpen(null);
        renderWatch();
        await screen.findByTestId("stream-chat");

        // when
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // then
        expect(screen.getByTestId("stream-chat")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: restoreLabel })).not.toBeInTheDocument();
    });

    it("closes the popped out window and takes the chat back", async () => {
        // given
        const popout = makePopout();
        stubWindowOpen(popout);
        renderWatch();
        await screen.findByTestId("stream-chat");
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // when
        await userEvent.click(screen.getByRole("button", { name: restoreLabel }));

        // then
        expect(popout.close).toHaveBeenCalled();
        expect(await screen.findByTestId("stream-chat")).toBeInTheDocument();
    });

    it("takes the chat back when the popped out window says it has closed", async () => {
        // given
        stubWindowOpen(makePopout());
        renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-chat");
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // when
        act(() => {
            window.dispatchEvent(
                new MessageEvent("message", {
                    data: { type: "stream-chat-popout-closed", streamId: "stream-1" },
                    origin: window.location.origin,
                }),
            );
        });

        // then
        expect(await screen.findByTestId("stream-chat")).toBeInTheDocument();
    });

    it("ignores a close message about a different stream", async () => {
        // given
        stubWindowOpen(makePopout());
        renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-chat");
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // when
        act(() => {
            window.dispatchEvent(
                new MessageEvent("message", {
                    data: { type: "stream-chat-popout-closed", streamId: "stream-2" },
                    origin: window.location.origin,
                }),
            );
        });

        // then
        expect(screen.queryByTestId("stream-chat")).not.toBeInTheDocument();
    });

    it("ignores a close message sent from another site", async () => {
        // given
        stubWindowOpen(makePopout());
        renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-chat");
        await userEvent.click(screen.getByRole("button", { name: "Pop out" }));

        // when
        act(() => {
            window.dispatchEvent(
                new MessageEvent("message", {
                    data: { type: "stream-chat-popout-closed", streamId: "stream-1" },
                    origin: "https://evil.example",
                }),
            );
        });

        // then
        expect(screen.queryByTestId("stream-chat")).not.toBeInTheDocument();
    });
});

describe("LiveWatchPage thumbnail trouble", () => {
    function stubCanvas() {
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

    function putVideoOnStage() {
        const video = document.createElement("video");
        Object.defineProperty(video, "videoWidth", { configurable: true, value: 1920 });
        Object.defineProperty(video, "videoHeight", { configurable: true, value: 1080 });
        screen.getByTestId("stream-stage").parentElement?.appendChild(video);
    }

    afterEach(() => {
        vi.useRealTimers();
    });

    it("puts the thumbnail warning on the page when the capture keeps failing", async () => {
        // given
        stubCanvas();
        mocks.uploadStreamThumbnail.mockRejectedValue(new Error("the ingress refused it"));
        vi.useFakeTimers();
        renderWatch({ user: makeUser({ id: "streamer-1" }) });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(100);
        });
        fireEvent.click(screen.getByRole("button", { name: "Show preview (muted)" }));
        await act(async () => {
            await vi.advanceTimersByTimeAsync(100);
        });
        putVideoOnStage();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(9000);
        });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(10);
        });

        // then
        expect(screen.getByRole("status")).toHaveTextContent("Your stream thumbnail is not updating.");
    });

    it("leaves the page clear of warnings while the thumbnails are going through", async () => {
        // given
        stubCanvas();
        vi.useFakeTimers();
        renderWatch({ user: makeUser({ id: "streamer-1" }) });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(100);
        });
        fireEvent.click(screen.getByRole("button", { name: "Show preview (muted)" }));
        await act(async () => {
            await vi.advanceTimersByTimeAsync(100);
        });
        putVideoOnStage();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(9000);
        });

        // then
        expect(mocks.uploadStreamThumbnail).toHaveBeenCalledWith("stream-1", expect.any(Blob));
        expect(screen.queryByRole("status")).not.toBeInTheDocument();
    });
});
