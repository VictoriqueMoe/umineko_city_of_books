import { act, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createRef } from "react";
import { describe, expect, it, vi } from "vitest";
import type { LiveStream, StreamDefaultMode } from "../../api/endpoints";
import { renderWithProviders } from "../../test-utils/render";
import { MobileLiveView } from "./MobileLiveView";

const mocks = vi.hoisted(() => ({ onChangeViewerCount: { current: null as ((count: number) => void) | null } }));

vi.mock("@livekit/components-react", () => ({
    RoomContext: { Provider: (props: { children: React.ReactNode }) => <>{props.children}</> },
    RoomAudioRenderer: (props: { volume: number }) => <div data-testid="audio-renderer">{props.volume}</div>,
    StartAudio: (props: { label: string }) => <button type="button">{props.label}</button>,
}));

vi.mock("./StreamChatPanel", () => ({
    StreamChatPanel: (props: { streamId: string; isLive: boolean }) => (
        <div data-testid="stream-chat" data-live={String(props.isLive)}>
            {props.streamId}
        </div>
    ),
}));

vi.mock("../../components/live/HLSVideoPlayer", () => ({
    HLSVideoPlayer: (props: { src: string; muted?: boolean }) => (
        <div data-testid="hls-player" data-muted={String(Boolean(props.muted))}>
            {props.src}
        </div>
    ),
}));

vi.mock("./streamParts", () => ({
    StreamStage: () => <div data-testid="stream-stage" />,
    StreamUptime: (props: { startedAt?: string }) => <div data-testid="uptime">{props.startedAt}</div>,
    StreamViewers: () => <div data-testid="stream-viewers" />,
    ViewerCountReporter: (props: { onChange: (count: number) => void }) => {
        mocks.onChangeViewerCount.current = props.onChange;
        return null;
    },
}));

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return {
        id: "stream-1",
        userId: "user-1",
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

interface ViewOptions {
    stream?: Partial<LiveStream>;
    room?: object | null;
    isLive?: boolean;
    error?: string | null;
    volume?: number;
    mode?: StreamDefaultMode;
    isOwnStream?: boolean;
    showOwnPreview?: boolean;
}

function renderView(options: ViewOptions = {}) {
    const onVolumeChange = vi.fn();
    const onToggleFullscreen = vi.fn();
    const onModeChange = vi.fn();
    const onToggleOwnPreview = vi.fn();
    const stageRef = createRef<HTMLDivElement>();

    const result = renderWithProviders(
        <MobileLiveView
            stream={makeStream(options.stream)}
            room={(options.room === undefined ? {} : options.room) as never}
            isLive={options.isLive ?? true}
            error={options.error ?? null}
            volume={options.volume ?? 0.5}
            onVolumeChange={onVolumeChange}
            stageRef={stageRef}
            onToggleFullscreen={onToggleFullscreen}
            mode={options.mode ?? "webrtc"}
            onModeChange={onModeChange}
            isOwnStream={options.isOwnStream ?? false}
            showOwnPreview={options.showOwnPreview ?? false}
            onToggleOwnPreview={onToggleOwnPreview}
        />,
    );

    return { ...result, onVolumeChange, onToggleFullscreen, onModeChange, onToggleOwnPreview, stageRef };
}

describe("MobileLiveView stage", () => {
    it("says the stream is offline when it is not live", () => {
        // given
        const isLive = false;

        // when
        renderView({ isLive });

        // then
        expect(screen.getByText("This stream is offline.")).toBeInTheDocument();
        expect(screen.queryByTestId("stream-stage")).not.toBeInTheDocument();
    });

    it("prefers the connection error over the plain offline message", () => {
        // given
        const error = "Could not connect to this stream.";

        // when
        renderView({ isLive: false, error });

        // then
        expect(screen.getByText("Could not connect to this stream.")).toBeInTheDocument();
    });

    it("waits while the room is still connecting", () => {
        // given
        const room = null;

        // when
        renderView({ room });

        // then
        expect(screen.getByText("Connecting...")).toBeInTheDocument();
    });

    it("shows the connection error instead of connecting once it has failed", () => {
        // given
        const error = "Could not connect to this stream.";

        // when
        renderView({ room: null, error });

        // then
        expect(screen.getByText("Could not connect to this stream.")).toBeInTheDocument();
        expect(screen.queryByText("Connecting...")).not.toBeInTheDocument();
    });

    it("plays the live room with sound and a volume control for a visitor", () => {
        // given
        const volume = 0.4;

        // when
        renderView({ volume });

        // then
        expect(screen.getByTestId("stream-stage")).toBeInTheDocument();
        expect(screen.getByTestId("audio-renderer")).toHaveTextContent("0.4");
        expect(screen.getByRole("slider", { name: "Stream volume" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Click to enable sound" })).toBeInTheDocument();
    });

    it("hides a streamer's own preview so they do not download their own video", () => {
        // given
        const isOwnStream = true;

        // when
        renderView({ isOwnStream, showOwnPreview: false });

        // then
        expect(screen.getByText(/preview is hidden/)).toBeInTheDocument();
        expect(screen.queryByTestId("stream-stage")).not.toBeInTheDocument();
    });

    it("lets the streamer ask for a muted preview of themselves", async () => {
        // given
        const user = userEvent.setup();
        const { onToggleOwnPreview } = renderView({ isOwnStream: true, showOwnPreview: false });

        // when
        await user.click(screen.getByRole("button", { name: "Show preview (muted)" }));

        // then
        expect(onToggleOwnPreview).toHaveBeenCalledWith(true);
    });

    it("silences the stream and drops the volume control for the streamer's own preview", () => {
        // given
        const isOwnStream = true;

        // when
        renderView({ isOwnStream, showOwnPreview: true });

        // then
        expect(screen.getByTestId("audio-renderer")).toHaveTextContent("0");
        expect(screen.queryByRole("slider", { name: "Stream volume" })).not.toBeInTheDocument();
    });

    it("lets the streamer hide their preview again", async () => {
        // given
        const user = userEvent.setup();
        const { onToggleOwnPreview } = renderView({ isOwnStream: true, showOwnPreview: true });

        // when
        await user.click(screen.getByRole("button", { name: "Hide preview" }));

        // then
        expect(onToggleOwnPreview).toHaveBeenCalledWith(false);
    });

    it("plays the smooth feed when the viewer is in hls mode", () => {
        // given
        const mode: StreamDefaultMode = "hls";

        // when
        renderView({ mode, stream: { hlsUrl: "https://edge/stream.m3u8" } });

        // then
        expect(screen.getByTestId("hls-player")).toHaveTextContent("https://edge/stream.m3u8");
        expect(screen.queryByTestId("stream-stage")).not.toBeInTheDocument();
    });

    it("falls back to the live room when hls is chosen but no url exists", () => {
        // given
        const mode: StreamDefaultMode = "hls";

        // when
        renderView({ mode, stream: { hlsUrl: undefined } });

        // then
        expect(screen.getByTestId("stream-stage")).toBeInTheDocument();
        expect(screen.queryByTestId("hls-player")).not.toBeInTheDocument();
    });

    it("mutes the smooth feed when the streamer previews their own stream", () => {
        // given
        const isOwnStream = true;

        // when
        renderView({ isOwnStream, showOwnPreview: true, mode: "hls", stream: { hlsUrl: "https://edge/s.m3u8" } });

        // then
        expect(screen.getByTestId("hls-player")).toHaveAttribute("data-muted", "true");
    });
});

describe("MobileLiveView controls", () => {
    it("offers no quality choice when the stream has no smooth feed", () => {
        // given
        const stream = { hlsUrl: undefined };

        // when
        renderView({ stream });

        // then
        expect(screen.queryByRole("button", { name: "Smooth" })).not.toBeInTheDocument();
    });

    it("switches to the smooth feed when asked", async () => {
        // given
        const user = userEvent.setup();
        const { onModeChange } = renderView({ stream: { hlsUrl: "https://edge/s.m3u8" } });

        // when
        await user.click(screen.getByRole("button", { name: "Smooth" }));

        // then
        expect(onModeChange).toHaveBeenCalledWith("hls");
    });

    it("switches back to the low latency feed when asked", async () => {
        // given
        const user = userEvent.setup();
        const { onModeChange } = renderView({ mode: "hls", stream: { hlsUrl: "https://edge/s.m3u8" } });

        // when
        await user.click(screen.getByRole("button", { name: "Low latency" }));

        // then
        expect(onModeChange).toHaveBeenCalledWith("webrtc");
    });

    it("asks its parent to go fullscreen", async () => {
        // given
        const user = userEvent.setup();
        const { onToggleFullscreen } = renderView();

        // when
        await user.click(screen.getByRole("button", { name: "Toggle fullscreen" }));

        // then
        expect(onToggleFullscreen).toHaveBeenCalledTimes(1);
    });

    it("hides the quality choice and uptime while the stream is offline", () => {
        // given
        const isLive = false;

        // when
        renderView({ isLive, stream: { hlsUrl: "https://edge/s.m3u8" } });

        // then
        expect(screen.queryByRole("button", { name: "Smooth" })).not.toBeInTheDocument();
        expect(screen.queryByTestId("uptime")).not.toBeInTheDocument();
    });

    it("resizes the stage when the handle is dragged", () => {
        // given
        const { container } = renderView();
        const handle = screen.getByRole("separator", { name: "Drag to resize the video" });
        const stage = container.querySelector("div") as HTMLDivElement;
        handle.setPointerCapture = vi.fn();
        handle.hasPointerCapture = vi.fn(() => false);

        // when
        fireEvent.pointerDown(handle, { clientY: 100, pointerId: 1 });
        fireEvent.pointerMove(handle, { clientY: 400, pointerId: 1 });
        fireEvent.pointerUp(handle, { clientY: 400, pointerId: 1 });

        // then
        expect(stage).toBeInTheDocument();
        expect(container.querySelector("[style*='height']")).not.toBeNull();
    });

    it("never shrinks the stage below its minimum height", () => {
        // given
        const { container } = renderView();
        const handle = screen.getByRole("separator", { name: "Drag to resize the video" });
        handle.setPointerCapture = vi.fn();
        handle.hasPointerCapture = vi.fn(() => false);

        // when
        fireEvent.pointerDown(handle, { clientY: 500, pointerId: 1 });
        fireEvent.pointerMove(handle, { clientY: 0, pointerId: 1 });

        // then
        const resized = container.querySelector("[style*='height']") as HTMLElement;
        expect(resized.style.height).toBe("96px");
    });

    it("ignores a drag that never started", () => {
        // given
        const { container } = renderView();
        const handle = screen.getByRole("separator", { name: "Drag to resize the video" });

        // when
        fireEvent.pointerMove(handle, { clientY: 400, pointerId: 1 });

        // then
        expect(container.querySelector("[style*='height']")).toBeNull();
    });
});

describe("MobileLiveView meta and tabs", () => {
    it("names the stream and links back to the streamer and directory", () => {
        // given
        const stream = { title: "Ciconia blind run", streamerUsername: "beatrice" };

        // when
        renderView({ stream });

        // then
        expect(screen.getByText("Ciconia blind run")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/user/beatrice");
        expect(screen.getByRole("link", { name: "Back to live streams" })).toHaveAttribute("href", "/live");
    });

    it("falls back to the username when the streamer has no display name", () => {
        // given
        const stream = { streamerDisplayName: "" };

        // when
        renderView({ stream });

        // then
        expect(screen.getByRole("link", { name: /beatrice/ })).toBeInTheDocument();
    });

    it("opens on the chat tab", () => {
        // given
        const stream = { id: "stream-8" };

        // when
        renderView({ stream });

        // then
        expect(screen.getByTestId("stream-chat")).toHaveTextContent("stream-8");
        expect(screen.getByRole("button", { name: "Chat" })).toBeInTheDocument();
    });

    it("switches to the viewers tab when asked", async () => {
        // given
        const user = userEvent.setup();
        renderView();
        expect(screen.getByTestId("stream-chat").parentElement?.className).not.toMatch(/mobilePaneHidden/);

        // when
        await user.click(screen.getByRole("button", { name: /Viewers/ }));

        // then
        expect(screen.getByTestId("stream-chat").parentElement?.className).toMatch(/mobilePaneHidden/);
        expect(screen.getByRole("button", { name: /Viewers/ }).className).toMatch(/mobileTabActive/);
    });

    it("says there are no viewers while the stream is offline", () => {
        // given
        const isLive = false;

        // when
        renderView({ isLive });

        // then
        expect(screen.getByText("No viewers while the stream is offline.")).toBeInTheDocument();
        expect(screen.queryByTestId("stream-viewers")).not.toBeInTheDocument();
    });

    it("shows the viewer count the room reports", () => {
        // given
        renderView();

        // when
        act(() => {
            mocks.onChangeViewerCount.current?.(11);
        });

        // then
        expect(screen.getByRole("button", { name: "Viewers (11)" })).toBeInTheDocument();
    });

    it("leaves the viewer count off the tab while the stream is offline", () => {
        // given
        const isLive = false;

        // when
        renderView({ isLive });

        // then
        expect(screen.getByRole("button", { name: "Viewers" })).toBeInTheDocument();
    });
});
