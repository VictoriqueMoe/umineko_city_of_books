import { describe, expect, it } from "vitest";
import { isStreamChatPopoutPath, isStreamWatchPath, streamChatPopoutPath, streamWatchPath } from "./streamUrl";

describe("streamWatchPath", () => {
    it("builds a streamer's permanent address", () => {
        // given a username

        // when
        const path = streamWatchPath("Featherine");

        // then
        expect(path).toBe("/Featherine/live");
    });
});

describe("isStreamWatchPath", () => {
    const cases = [
        { name: "a streamer's page", path: "/Featherine/live", want: true },
        { name: "casing does not matter", path: "/featherine/live", want: true },
        { name: "the live games page is not a streamer", path: "/games/live", want: false },
        { name: "the live directory is not a streamer", path: "/live", want: false },
        { name: "a nested live path is not a streamer", path: "/live/abc", want: false },
        { name: "another site route is not a streamer", path: "/theory/live", want: false },
        { name: "a static mount is not a streamer", path: "/uploads/live", want: false },
        { name: "the popout is not the watch page", path: "/beatrice/live/chat", want: false },
        { name: "a profile page is not a stream", path: "/user/beatrice", want: false },
    ];

    for (const tc of cases) {
        it(tc.name, () => {
            // given a pathname

            // when
            const got = isStreamWatchPath(tc.path);

            // then
            expect(got).toBe(tc.want);
        });
    }
});

describe("streamChatPopoutPath", () => {
    it("hangs the chat window off the streamer's own page", () => {
        // given a username

        // when
        const path = streamChatPopoutPath("Featherine");

        // then
        expect(path).toBe("/Featherine/live/chat");
    });
});

describe("isStreamChatPopoutPath", () => {
    const cases = [
        { name: "a streamer's chat window", path: "/beatrice/live/chat", want: true },
        { name: "casing does not matter", path: "/Beatrice/live/chat", want: true },
        { name: "the watch page itself is not the popout", path: "/beatrice/live", want: false },
        { name: "a reserved segment never opens a bare shell", path: "/games/live/chat", want: false },
        { name: "another reserved segment stays out", path: "/uploads/live/chat", want: false },
        { name: "the retired address is gone", path: "/live/abc/chat", want: false },
    ];

    for (const tc of cases) {
        it(tc.name, () => {
            // given a pathname

            // when
            const got = isStreamChatPopoutPath(tc.path);

            // then
            expect(got).toBe(tc.want);
        });
    }
});
