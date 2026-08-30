import { afterEach, describe, expect, it, vi } from "vitest";
import {
    STREAM_CHAT_POPOUT_CLOSED,
    openStreamChatPopout,
    streamChatPopoutName,
    streamChatPopoutPath,
} from "./streamChatPopout";

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("streamChatPopoutPath", () => {
    it("points at the bare chat page for the stream", () => {
        // given
        const streamId = "stream-7";

        // when
        const path = streamChatPopoutPath(streamId);

        // then
        expect(path).toBe("/live/stream-7/chat");
    });
});

describe("streamChatPopoutName", () => {
    it("names the window after the stream so a second click reuses it", () => {
        // given
        const streamId = "stream-7";

        // when
        const name = streamChatPopoutName(streamId);

        // then
        expect(name).toBe("stream-chat-stream-7");
    });

    it("gives two different streams two different windows", () => {
        // given / when / then
        expect(streamChatPopoutName("a")).not.toBe(streamChatPopoutName("b"));
    });
});

describe("openStreamChatPopout", () => {
    it("opens the chat page in a resizable popup window", () => {
        // given
        const open = vi.fn(() => ({}) as Window);
        vi.stubGlobal("open", open);

        // when
        openStreamChatPopout("stream-7");

        // then
        const [url, name, features] = open.mock.calls[0] as unknown as [string, string, string];
        expect(url).toBe("/live/stream-7/chat");
        expect(name).toBe("stream-chat-stream-7");
        expect(features).toContain("popup=yes");
        expect(features).toContain("resizable=yes");
    });

    it("passes back nothing when the browser refuses to open the window", () => {
        // given
        vi.stubGlobal(
            "open",
            vi.fn(() => null),
        );

        // when
        const opened = openStreamChatPopout("stream-7");

        // then
        expect(opened).toBeNull();
    });
});

describe("STREAM_CHAT_POPOUT_CLOSED", () => {
    it("keeps a stable name that both windows agree on", () => {
        // given / when / then
        expect(STREAM_CHAT_POPOUT_CLOSED).toBe("stream-chat-popout-closed");
    });
});
