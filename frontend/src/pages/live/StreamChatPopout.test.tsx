import { screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { WSMessage } from "../../types/api";
import { STREAM_CHAT_POPOUT_CLOSED } from "../../utils/streamChatPopout";
import { StreamChatPopout } from "./StreamChatPopout";

const mocks = vi.hoisted(() => ({
    getStream: vi.fn(),
}));

vi.mock("../../api/endpoints", () => ({ getStream: mocks.getStream }));

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

interface StreamOptions {
    id?: string;
    title?: string;
    status?: string;
}

function makeStream(options: StreamOptions = {}) {
    return {
        id: options.id ?? "stream-1",
        title: options.title ?? "Tea party",
        status: options.status ?? "live",
        userId: "streamer-1",
        streamerUsername: "beatrice",
        streamerDisplayName: "Beatrice",
    };
}

function renderPopout(streamId = "stream-1") {
    const listeners: ((msg: WSMessage) => void)[] = [];
    const result = renderWithProviders(<StreamChatPopout />, {
        route: `/live/${streamId}/chat`,
        path: "/live/:streamID/chat",
        notification: {
            addWSListener: listener => {
                listeners.push(listener);
                return () => {};
            },
        },
    });

    return { ...result, listeners };
}

beforeEach(() => {
    mocks.getStream.mockResolvedValue(makeStream());
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
            expect(mocks.getStream).toHaveBeenCalledWith("stream-42");
        });
    });

    it("hands the chat panel the stream it is chatting about", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ id: "stream-9", status: "live" }));

        // when
        renderPopout("stream-9");

        // then
        const panel = await screen.findByTestId("panel");
        expect(panel).toHaveAttribute("data-stream", "stream-9");
        expect(panel).toHaveAttribute("data-live", "true");
    });

    it("never offers a second pop out control inside the popped out window", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream());

        // when
        renderPopout();

        // then
        expect(await screen.findByTestId("panel")).toHaveAttribute("data-has-popout", "false");
    });

    it("closes the chat off when the stream is no longer live", async () => {
        // given
        mocks.getStream.mockResolvedValue(makeStream({ status: "offline" }));

        // when
        renderPopout();

        // then
        expect(await screen.findByTestId("panel")).toHaveAttribute("data-live", "false");
    });

    it("says so when the stream cannot be found", async () => {
        // given
        mocks.getStream.mockRejectedValue(new Error("gone"));

        // when
        renderPopout();

        // then
        expect(await screen.findByText("Stream not found.")).toBeInTheDocument();
    });

    it("re-reads the stream when it is told the stream went offline", async () => {
        // given
        const { listeners } = renderPopout("stream-1");
        await screen.findByTestId("panel");
        mocks.getStream.mockClear();

        // when
        for (const listener of listeners) {
            listener({ type: "stream_offline", data: { streamId: "stream-1" } } as WSMessage);
        }

        // then
        await waitFor(() => {
            expect(mocks.getStream).toHaveBeenCalled();
        });
    });

    it("ignores news about a stream it is not showing", async () => {
        // given
        const { listeners } = renderPopout("stream-1");
        await screen.findByTestId("panel");
        mocks.getStream.mockClear();

        // when
        for (const listener of listeners) {
            listener({ type: "stream_offline", data: { streamId: "some-other-stream" } } as WSMessage);
        }

        // then
        expect(mocks.getStream).not.toHaveBeenCalled();
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
});
