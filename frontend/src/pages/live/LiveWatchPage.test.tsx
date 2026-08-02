import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { LiveStream } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../../test-utils/render";
import type { UserProfile, WSMessage } from "../../types/api";
import { LiveWatchPage } from "./LiveWatchPage";

const mocks = vi.hoisted(() => {
    const created: FakeRoom[] = [];

    class FakeRoom {
        handlers = new Map<string, (() => void)[]>();
        connect = vi.fn((_url: string, _token: string, _options?: { autoSubscribe?: boolean }) => {
            const listeners = this.handlers.get("connected") ?? [];
            for (const listener of listeners) {
                listener();
            }
            return Promise.resolve();
        });
        disconnect = vi.fn(() => Promise.resolve());

        constructor() {
            created.push(this);
        }

        on(event: string, handler: () => void) {
            const listeners = this.handlers.get(event) ?? [];
            listeners.push(handler);
            this.handlers.set(event, listeners);
            return this;
        }
    }

    return {
        created,
        FakeRoom,
        getStream: vi.fn(),
        getStreamViewerToken: vi.fn(),
        uploadStreamThumbnail: vi.fn(),
        useIsMobile: vi.fn(),
    };
});

vi.mock("livekit-client", () => ({
    Room: mocks.FakeRoom,
    RoomEvent: { Connected: "connected", Disconnected: "disconnected" },
}));

vi.mock("@livekit/components-react", () => ({
    RoomContext: { Provider: (props: { children: React.ReactNode }) => <>{props.children}</> },
    RoomAudioRenderer: (props: { volume: number }) => <div data-testid="audio-renderer">{props.volume}</div>,
    StartAudio: (props: { label: string }) => <button type="button">{props.label}</button>,
}));

vi.mock("../../api/endpoints", () => ({
    getStream: mocks.getStream,
    getStreamViewerToken: mocks.getStreamViewerToken,
    uploadStreamThumbnail: mocks.uploadStreamThumbnail,
}));

vi.mock("../../hooks/useIsMobile", () => ({ useIsMobile: mocks.useIsMobile }));

vi.mock("./StreamChatPanel", () => ({
    StreamChatPanel: (props: { streamId: string; isLive: boolean }) => (
        <div data-testid="stream-chat" data-live={String(props.isLive)}>
            {props.streamId}
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
    return {
        id: "stream-1",
        userId: "streamer-1",
        title: "Reading Episode 4",
        status: "live",
        viewerCount: 3,
        startedAt: "2026-02-01T12:00:00Z",
        streamerUsername: "beatrice",
        streamerDisplayName: "Beatrice",
        streamerAvatarUrl: "",
        defaultMode: "webrtc",
        ...overrides,
    };
}

function renderWatch(options: { user?: UserProfile | null; streamID?: string } = {}) {
    const listeners: ((msg: WSMessage) => void)[] = [];
    const queryClient = createTestQueryClient();
    const result = renderWithProviders(<LiveWatchPage />, {
        user: options.user ?? null,
        queryClient,
        route: `/live/${options.streamID ?? "stream-1"}`,
        path: "/live/:streamID",
        notification: {
            addWSListener: listener => {
                listeners.push(listener);
                return () => {};
            },
        },
    });

    return { ...result, listeners, queryClient };
}

beforeEach(() => {
    mocks.created.length = 0;
    mocks.useIsMobile.mockReturnValue(false);
    mocks.getStream.mockResolvedValue(makeStream());
    mocks.getStreamViewerToken.mockResolvedValue({ token: "tok", url: "wss://livekit.test" });
    mocks.uploadStreamThumbnail.mockResolvedValue(undefined);
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

    it("asks the server for the stream named in the address", async () => {
        // given
        const streamID = "stream-77";

        // when
        renderWatch({ streamID });

        // then
        await waitFor(() => {
            expect(mocks.getStream).toHaveBeenCalledWith("stream-77");
        });
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
        expect(mocks.created).toHaveLength(0);
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

    it("subscribes to the media of somebody else's stream", async () => {
        // given
        renderWatch({ user: makeUser({ id: "viewer-1" }) });

        // when
        await screen.findByTestId("stream-stage");

        // then
        expect(mocks.created[0].connect).toHaveBeenCalledWith("wss://livekit.test", "tok", { autoSubscribe: true });
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

    it("joins the room without media while a streamer's own preview is hidden", async () => {
        // given
        const user = makeUser({ id: "streamer-1" });

        // when
        renderWatch({ user });
        await screen.findByText(/preview is hidden/);

        // then
        expect(mocks.created[0].connect).toHaveBeenCalledWith("wss://livekit.test", "tok", { autoSubscribe: false });
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

    it("opens no livekit room while the smooth feed is playing", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ defaultMode: "hls", hlsUrl: "https://edge/s.m3u8" }));

        // when
        renderWatch();
        await screen.findByTestId("hls-player");

        // then
        expect(mocks.created).toHaveLength(0);
        expect(mocks.getStreamViewerToken).not.toHaveBeenCalled();
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

describe("LiveWatchPage thumbnails", () => {
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

    it("never sends a thumbnail from somebody watching another player's stream", async () => {
        // given
        stubCanvas();
        vi.useFakeTimers();
        renderWatch({ user: makeUser({ id: "viewer-1" }) });
        await act(async () => {
            await vi.advanceTimersByTimeAsync(100);
        });
        putVideoOnStage();

        // when
        await act(async () => {
            await vi.advanceTimersByTimeAsync(60000);
        });

        // then
        expect(mocks.uploadStreamThumbnail).not.toHaveBeenCalled();
    });

    it("sends a thumbnail from the streamer's own preview", async () => {
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
    });
});

describe("LiveWatchPage live updates", () => {
    it("refetches the stream when it goes offline", async () => {
        // given
        const { listeners, queryClient } = renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-stage");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "stream_offline", data: { streamId: "stream-1" } });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["streams", "detail", "stream-1"] });
    });

    it("ignores another stream going offline", async () => {
        // given
        const { listeners, queryClient } = renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-stage");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "stream_offline", data: { streamId: "another-stream" } });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });

    it("refetches the stream when it comes back live", async () => {
        // given
        const { listeners, queryClient } = renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-stage");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "stream_live", data: makeStream({ id: "stream-1" }) });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["streams", "detail", "stream-1"] });
    });

    it("renames the stream in place when the streamer changes the title", async () => {
        // given
        const { listeners } = renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-stage");

        // when
        listeners[0]({ type: "stream_title", data: { streamId: "stream-1", title: "Now solving the epitaph" } });

        // then
        expect(await screen.findByRole("heading", { name: "Now solving the epitaph" })).toBeInTheDocument();
    });

    it("leaves the title alone when another stream is renamed", async () => {
        // given
        const { listeners } = renderWatch({ streamID: "stream-1" });
        await screen.findByTestId("stream-stage");

        // when
        listeners[0]({ type: "stream_title", data: { streamId: "another-stream", title: "Something else" } });

        // then
        expect(screen.getByRole("heading", { name: "Reading Episode 4" })).toBeInTheDocument();
    });
});
