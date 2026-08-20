import { describe, expect, it } from "vitest";
import {
    KNOWN_CRAWLER_FEEDS,
    customFeeds,
    isKnownFeedEnabled,
    parseCrawlerFeeds,
    replaceCustomFeeds,
    serialiseCrawlerFeeds,
    toggleKnownFeed,
} from "./crawlerFeeds";

const BING = KNOWN_CRAWLER_FEEDS[0];
const BING_LINE = `bing=${BING.url}`;

describe("parseCrawlerFeeds", () => {
    it("reads one entry per line", () => {
        // given the format the backend stores
        const feeds = parseCrawlerFeeds(`${BING_LINE}\nmine=https://example.com/r.json`);

        // then
        expect(feeds).toEqual([
            { name: "bing", url: BING.url },
            { name: "mine", url: "https://example.com/r.json" },
        ]);
    });

    it("keeps a comma inside a url, because query strings contain them", () => {
        // given the case that used to truncate the url at the comma
        const feeds = parseCrawlerFeeds("mine=https://example.com/r.json?ids=1,2");

        // then
        expect(feeds).toEqual([{ name: "mine", url: "https://example.com/r.json?ids=1,2" }]);
    });

    it("splits on the first equals only, so a query string survives", () => {
        // given
        const feeds = parseCrawlerFeeds("mine=https://example.com/r.json?a=b&c=d");

        // then
        expect(feeds).toEqual([{ name: "mine", url: "https://example.com/r.json?a=b&c=d" }]);
    });

    it("drops plain http, matching what the backend accepts", () => {
        // given an entry the Go parser silently ignores
        const feeds = parseCrawlerFeeds("mine=http://example.com/r.json");

        // then the UI must not show something the server will drop
        expect(feeds).toEqual([]);
    });

    it("drops entries with no name, no url, or no equals", () => {
        expect(parseCrawlerFeeds("=https://example.com/r.json")).toEqual([]);
        expect(parseCrawlerFeeds("mine=")).toEqual([]);
        expect(parseCrawlerFeeds("mine")).toEqual([]);
        expect(parseCrawlerFeeds("")).toEqual([]);
    });
});

describe("serialiseCrawlerFeeds", () => {
    it("refuses an entry that would reparse differently", () => {
        // given a name containing an equals, which would rename itself on reparse
        const serialised = serialiseCrawlerFeeds([{ name: "a=b", url: "https://example.com/r.json" }]);

        // then
        expect(serialised).toBe("");
    });

    it("refuses a half typed row rather than writing a broken line", () => {
        expect(serialiseCrawlerFeeds([{ name: "mine", url: "" }])).toBe("");
        expect(serialiseCrawlerFeeds([{ name: "", url: "https://example.com/r.json" }])).toBe("");
        expect(serialiseCrawlerFeeds([{ name: "mine", url: "http://example.com/r.json" }])).toBe("");
    });

    it("round trips a valid set", () => {
        // given
        const feeds = [
            { name: "bing", url: BING.url },
            { name: "mine", url: "https://example.com/r.json?ids=1,2" },
        ];

        // then
        expect(parseCrawlerFeeds(serialiseCrawlerFeeds(feeds))).toEqual(feeds);
    });
});

describe("toggleKnownFeed", () => {
    it("adds a known feed with its published url", () => {
        expect(toggleKnownFeed("", BING, true)).toBe(BING_LINE);
    });

    it("removes it again", () => {
        expect(toggleKnownFeed(BING_LINE, BING, false)).toBe("");
    });

    it("leaves custom entries alone", () => {
        // given
        const raw = `mine=https://example.com/r.json\n${BING_LINE}`;

        // when
        const next = toggleKnownFeed(raw, BING, false);

        // then
        expect(parseCrawlerFeeds(next)).toEqual([{ name: "mine", url: "https://example.com/r.json" }]);
    });
});

describe("customFeeds", () => {
    it("excludes the known feeds", () => {
        // given
        const raw = `${BING_LINE}\nmine=https://example.com/r.json`;

        // then
        expect(customFeeds(raw)).toEqual([{ name: "mine", url: "https://example.com/r.json" }]);
    });
});

describe("replaceCustomFeeds", () => {
    it("keeps the known feeds while swapping the custom ones", () => {
        // given
        const raw = `${BING_LINE}\nold=https://old.example/r.json`;

        // when
        const next = replaceCustomFeeds(raw, [{ name: "new", url: "https://new.example/r.json" }]);

        // then
        expect(parseCrawlerFeeds(next)).toEqual([
            { name: "bing", url: BING.url },
            { name: "new", url: "https://new.example/r.json" },
        ]);
    });

    it("refuses a custom entry that would shadow a known feed", () => {
        // given somebody naming their own feed bing, which would be invisible and clobbered
        const next = replaceCustomFeeds(BING_LINE, [{ name: "bing", url: "https://mirror.example/r.json" }]);

        // then the published url wins and the mirror is not silently swallowed
        expect(parseCrawlerFeeds(next)).toEqual([{ name: "bing", url: BING.url }]);
    });

    it("drops blank rows the admin has not finished typing", () => {
        // given
        const next = replaceCustomFeeds("", [
            { name: "", url: "" },
            { name: "mine", url: "https://example.com/r.json" },
        ]);

        // then
        expect(next).toBe("mine=https://example.com/r.json");
    });
});

describe("isKnownFeedEnabled", () => {
    it("reports each known feed independently", () => {
        expect(isKnownFeedEnabled(BING_LINE, BING)).toBe(true);
        expect(isKnownFeedEnabled(BING_LINE, KNOWN_CRAWLER_FEEDS[1])).toBe(false);
    });

    it("is false for an entry the backend would drop", () => {
        // given a bing row over plain http
        expect(isKnownFeedEnabled("bing=http://www.bing.com/toolbox/bingbot.json", BING)).toBe(false);
    });
});
