import { describe, expect, it } from "vitest";
import {
    CACHING_TOKEN_THRESHOLD,
    CHARS_PER_TOKEN,
    channelLabel,
    channelRows,
    channelTokens,
    estimateTokens,
    meetsCachingThreshold,
    tokensToCachingThreshold,
} from "./chatbotUsage";
import type { ChatbotChannelUsage } from "../types/api";

function makeChannel(overrides: Partial<ChatbotChannelUsage> = {}): ChatbotChannelUsage {
    return {
        channel: "group",
        invocations: 0,
        prompt_tokens: 0,
        cached_prompt_tokens: 0,
        cache_write_tokens: 0,
        completion_tokens: 0,
        reasoning_tokens: 0,
        ...overrides,
    };
}

describe("estimateTokens", () => {
    it("charges four characters to a token", () => {
        // given
        const text = "a".repeat(CHARS_PER_TOKEN * 500);

        // when
        const tokens = estimateTokens(text);

        // then
        expect(tokens).toBe(500);
    });

    it("rounds a part token up rather than down, so the estimate never flatters the bill", () => {
        // given
        const texts = ["", "a", "abcd", "abcde", "abcdefg"];

        // when
        const tokens = texts.map(estimateTokens);

        // then
        expect(tokens).toEqual([0, 1, 1, 2, 2]);
    });
});

describe("meetsCachingThreshold", () => {
    it("counts a prompt sitting exactly on the threshold as eligible", () => {
        // given
        const tokens = CACHING_TOKEN_THRESHOLD;

        // when
        const eligible = meetsCachingThreshold(tokens);

        // then
        expect(eligible).toBe(true);
    });

    it("holds a prompt one token short back", () => {
        // given
        const tokens = CACHING_TOKEN_THRESHOLD - 1;

        // when
        const eligible = meetsCachingThreshold(tokens);

        // then
        expect(eligible).toBe(false);
    });

    it("keeps a prompt past the threshold eligible", () => {
        // given
        const tokens = CACHING_TOKEN_THRESHOLD + 1;

        // when
        const eligible = meetsCachingThreshold(tokens);

        // then
        expect(eligible).toBe(true);
    });
});

describe("tokensToCachingThreshold", () => {
    it("counts the tokens still needed to reach caching", () => {
        // given
        const tokens = 250;

        // when
        const remaining = tokensToCachingThreshold(tokens);

        // then
        expect(remaining).toBe(CACHING_TOKEN_THRESHOLD - 250);
    });

    it("reaches zero exactly where the prompt becomes eligible", () => {
        // given
        const tokens = CACHING_TOKEN_THRESHOLD;

        // when
        const remaining = tokensToCachingThreshold(tokens);

        // then
        expect(remaining).toBe(0);
    });
});

describe("channelLabel", () => {
    it("names the four channels a bot can be reached through", () => {
        // given
        const channels = ["group", "dm", "post", "post_comment"];

        // when
        const labels = channels.map(channelLabel);

        // then
        expect(labels).toEqual(["Group chats", "DMs", "Posts", "Post comments"]);
    });

    it("shows a channel the server invented under its own name", () => {
        // given
        const channel = "watch_party";

        // when
        const label = channelLabel(channel);

        // then
        expect(label).toBe("watch_party");
    });
});

describe("channelTokens", () => {
    it("bills input and output only, leaving cached, cache write and reasoning tokens out", () => {
        // given
        const channel = makeChannel({
            prompt_tokens: 700000,
            completion_tokens: 12000,
            cached_prompt_tokens: 480000,
            cache_write_tokens: 1200,
            reasoning_tokens: 900,
        });

        // when
        const tokens = channelTokens(channel);

        // then
        expect(tokens).toBe(712000);
    });

    it("counts nothing for a channel that was never used", () => {
        // given
        const channel = makeChannel();

        // when
        const tokens = channelTokens(channel);

        // then
        expect(tokens).toBe(0);
    });
});

describe("channelRows", () => {
    it("lists the known channels in the same order whatever order the server sent them", () => {
        // given
        const channels = [
            makeChannel({ channel: "post_comment", invocations: 90 }),
            makeChannel({ channel: "dm", invocations: 40 }),
            makeChannel({ channel: "post", invocations: 18 }),
            makeChannel({ channel: "group", invocations: 180 }),
        ];

        // when
        const rows = channelRows(channels);

        // then
        expect(rows.map(row => row.channel)).toEqual(["group", "dm", "post", "post_comment"]);
        expect(rows.map(row => row.invocations)).toEqual([180, 40, 18, 90]);
    });

    it("fills a channel the server did not report with an all zero row", () => {
        // given
        const channels = [makeChannel({ channel: "dm", invocations: 40 })];

        // when
        const rows = channelRows(channels);

        // then
        expect(rows.map(row => row.channel)).toEqual(["group", "dm", "post", "post_comment"]);
        expect(rows[0]).toEqual(makeChannel({ channel: "group" }));
        expect(rows[2]).toEqual(makeChannel({ channel: "post" }));
        expect(rows[3]).toEqual(makeChannel({ channel: "post_comment" }));
    });

    it("returns the four known rows even when the server reported nothing", () => {
        // given
        const channels: ChatbotChannelUsage[] = [];

        // when
        const rows = channelRows(channels);

        // then
        expect(rows.map(row => row.channel)).toEqual(["group", "dm", "post", "post_comment"]);
        expect(rows.every(row => row.invocations === 0)).toBe(true);
    });

    it("appends a channel it does not know about after the known four", () => {
        // given
        const channels = [
            makeChannel({ channel: "watch_party", invocations: 7 }),
            makeChannel({ channel: "stream", invocations: 3 }),
            makeChannel({ channel: "dm", invocations: 40 }),
        ];

        // when
        const rows = channelRows(channels);

        // then
        expect(rows.map(row => row.channel)).toEqual(["group", "dm", "post", "post_comment", "watch_party", "stream"]);
        expect(rows[4].invocations).toBe(7);
        expect(rows[5].invocations).toBe(3);
    });

    it("keeps the last entry when the server repeats a channel", () => {
        // given
        const channels = [
            makeChannel({ channel: "dm", invocations: 40 }),
            makeChannel({ channel: "dm", invocations: 41 }),
        ];

        // when
        const rows = channelRows(channels);

        // then
        expect(rows).toHaveLength(4);
        expect(rows[1].invocations).toBe(41);
    });

    it("does not mutate the channels it was given", () => {
        // given
        const channels = [makeChannel({ channel: "dm", invocations: 40 })];

        // when
        channelRows(channels);

        // then
        expect(channels).toEqual([makeChannel({ channel: "dm", invocations: 40 })]);
    });
});
