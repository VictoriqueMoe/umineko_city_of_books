import { screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeStream as makeLiveStream } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { LiveStream } from "../../types/api";
import { STREAM_CHAT_POPOUT_CLOSED } from "../../platform/streamChatPopout";
import { StreamChatPopout } from "./StreamChatPopout";

const mocks = vi.hoisted(() => ({
    useStreamDetail: vi.fn(),
    useLiveStream: vi.fn(),
}));

vi.mock("../../hooks/useLiveStream", () => ({
    useStreamDetail: mocks.useStreamDetail,
    useLiveStream: mocks.useLiveStream,
}));

vi.mock("./StreamChatPanel", () => ({
    StreamChatPanel: (props: { streamId: string; isLive: boolean; onPopOut?: () => void }) => (
        <div
            data-testid="panel"
            data-stream={props.streamId}
            data-live={String(props.isLive)}
            data-has-popout={String(!!props.onPopOut)}
        />
    ),
}));

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return makeLiveStream({
        userId: "streamer-1",
        title: "Tea party",
        viewerCount: 0,
        streamerUsername: "beatrice",
        ...overrides,
    });
}

function stubStream(stream: LiveStream | null = makeStream()) {
    mocks.useStreamDetail.mockReturnValue({ stream, loading: false, error: "" });
}

function renderPopout(streamId = "stream-1") {
    return renderWithProviders(<StreamChatPopout />, {
        route: `/live/${streamId}/chat`,
        path: "/live/:streamID/chat",
    });
}

beforeEach(() => {
    stubStream();
});

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("StreamChatPopout", () => {
    it("fetches the stream named in the address", async () => {
        // given
        const streamId = "stream-42";

        // when
        renderPopout(streamId);

        // then
        await waitFor(() => {
            expect(mocks.useStreamDetail).toHaveBeenCalledWith("stream-42");
        });
    });

    it("hands the chat panel the stream it is chatting about", async () => {
        // given
        stubStream(makeStream({ id: "stream-9", status: "live" }));

        // when
        renderPopout("stream-9");

        // then
        const panel = await screen.findByTestId("panel");
        expect(panel).toHaveAttribute("data-stream", "stream-9");
        expect(panel).toHaveAttribute("data-live", "true");
    });

    it("never offers a second pop out control inside the popped out window", async () => {
        // given
        stubStream();

        // when
        renderPopout();

        // then
        expect(await screen.findByTestId("panel")).toHaveAttribute("data-has-popout", "false");
    });

    it("closes the chat off when the stream is no longer live", async () => {
        // given
        stubStream(makeStream({ status: "offline" }));

        // when
        renderPopout();

        // then
        expect(await screen.findByTestId("panel")).toHaveAttribute("data-live", "false");
    });

    it("says so when the stream cannot be found", async () => {
        // given
        stubStream(null);

        // when
        renderPopout();

        // then
        expect(await screen.findByText("Stream not found.")).toBeInTheDocument();
    });

    it("tells the window it came from when it is closed", async () => {
        // given
        const postMessage = vi.fn();
        vi.stubGlobal("opener", { closed: false, postMessage });
        renderPopout("stream-1");
        await screen.findByTestId("panel");

        // when
        window.dispatchEvent(new Event("pagehide"));

        // then
        expect(postMessage).toHaveBeenCalledWith(
            { type: STREAM_CHAT_POPOUT_CLOSED, streamId: "stream-1" },
            window.location.origin,
        );
    });

    it("stays quiet when the window it came from has already gone", async () => {
        // given
        const postMessage = vi.fn();
        vi.stubGlobal("opener", { closed: true, postMessage });
        renderPopout("stream-1");
        await screen.findByTestId("panel");

        // when
        window.dispatchEvent(new Event("pagehide"));

        // then
        expect(postMessage).not.toHaveBeenCalled();
    });

    it("survives being opened directly rather than popped out", async () => {
        // given
        vi.stubGlobal("opener", null);

        // when
        renderPopout("stream-1");

        // then
        expect(await screen.findByTestId("panel")).toBeInTheDocument();
    });

    it("never joins the stream's livekit room, it only carries the chat", async () => {
        // given
        renderPopout("stream-1");

        // when
        await screen.findByTestId("panel");

        // then
        expect(mocks.useLiveStream).not.toHaveBeenCalled();
    });
});
