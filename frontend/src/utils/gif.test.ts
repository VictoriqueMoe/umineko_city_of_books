import { describe, expect, it } from "vitest";
import { extractGif, extractGiphyId } from "./gif";

describe("extractGif", () => {
    it("accepts a giphy media url and hands it back trimmed", () => {
        // given
        const body = "  https://media.giphy.com/media/l0HlQ7LRal7QN0Fmg/giphy.gif  ";

        // when
        const gif = extractGif(body);

        // then
        expect(gif).toBe("https://media.giphy.com/media/l0HlQ7LRal7QN0Fmg/giphy.gif");
    });

    it("accepts every giphy host shape it supports", () => {
        // given / when / then
        expect(extractGif("https://media.giphy.com/media/abc/giphy.gif")).not.toBeNull();
        expect(extractGif("https://media0.giphy.com/media/abc/giphy.gif")).not.toBeNull();
        expect(extractGif("https://media4.giphy.com/media/abc/giphy.gif")).not.toBeNull();
        expect(extractGif("https://i.giphy.com/abc.gif")).not.toBeNull();
    });

    it("accepts webp and mp4 as well as gif, whatever the case", () => {
        // given / when / then
        expect(extractGif("https://media.giphy.com/media/abc/giphy.webp")).not.toBeNull();
        expect(extractGif("https://media.giphy.com/media/abc/giphy.mp4")).not.toBeNull();
        expect(extractGif("https://media.giphy.com/media/abc/giphy.GIF")).not.toBeNull();
    });

    it("keeps a trailing query string", () => {
        // given
        const body = "https://media.giphy.com/media/abc/giphy.gif?cid=790b7611&ct=g";

        // when
        const gif = extractGif(body);

        // then
        expect(gif).toBe("https://media.giphy.com/media/abc/giphy.gif?cid=790b7611&ct=g");
    });

    it("rejects anything that is not a bare https giphy media url", () => {
        // given / when / then
        expect(extractGif("http://media.giphy.com/media/abc/giphy.gif")).toBeNull();
        expect(extractGif("https://giphy.com/media/abc/giphy.gif")).toBeNull();
        expect(extractGif("https://media.giphy.com.evil.test/media/abc/giphy.gif")).toBeNull();
        expect(extractGif("https://tenor.com/media/abc/thing.gif")).toBeNull();
        expect(extractGif("https://media.giphy.com/media/abc/giphy.png")).toBeNull();
    });

    it("rejects a url that is only part of a longer message", () => {
        // given / when / then
        expect(extractGif("look at this https://media.giphy.com/media/abc/giphy.gif")).toBeNull();
        expect(extractGif("https://media.giphy.com/media/abc/giphy.gif lol")).toBeNull();
    });

    it("returns nothing for an empty or blank body", () => {
        // given / when / then
        expect(extractGif("")).toBeNull();
        expect(extractGif("   \n  ")).toBeNull();
    });
});

describe("extractGiphyId", () => {
    it("reads the id out of a media path", () => {
        // given
        const url = "https://media.giphy.com/media/l0HlQ7LRal7QN0Fmg/giphy.gif";

        // when
        const id = extractGiphyId(url);

        // then
        expect(id).toBe("l0HlQ7LRal7QN0Fmg");
    });

    it("skips the versioned segment giphy puts before the id", () => {
        // given
        const url = "https://media0.giphy.com/media/v1.Y2lkPTc5MGI3NjEx/l0HlQ7LRal7QN0Fmg/giphy.webp";

        // when
        const id = extractGiphyId(url);

        // then
        expect(id).toBe("l0HlQ7LRal7QN0Fmg");
    });

    it("reads the id out of the short i subdomain shape", () => {
        // given / when / then
        expect(extractGiphyId("https://i.giphy.com/l0HlQ7LRal7QN0Fmg.gif")).toBe("l0HlQ7LRal7QN0Fmg");
        expect(extractGiphyId("https://i.giphy.com/l0HlQ7LRal7QN0Fmg.mp4")).toBe("l0HlQ7LRal7QN0Fmg");
        expect(extractGiphyId("https://i.giphy.com/l0HlQ7LRal7QN0Fmg.GIF")).toBe("l0HlQ7LRal7QN0Fmg");
    });

    it("ignores a query string on either shape", () => {
        // given / when / then
        expect(extractGiphyId("https://media.giphy.com/media/abc123/giphy.gif?cid=790b7611")).toBe("abc123");
        expect(extractGiphyId("https://i.giphy.com/abc123.gif?cid=790b7611")).toBe("abc123");
    });

    it("prefers the media path when a url carries both shapes", () => {
        // given
        const url = "https://i.giphy.com/media/abc123/giphy.mp4";

        // when
        const id = extractGiphyId(url);

        // then
        expect(id).toBe("abc123");
    });

    it("returns nothing when no id can be read", () => {
        // given / when / then
        expect(extractGiphyId("")).toBeNull();
        expect(extractGiphyId("https://media.giphy.com/stickers/abc123/giphy.gif")).toBeNull();
        expect(extractGiphyId("https://i.giphy.com/abc123.png")).toBeNull();
        expect(extractGiphyId("https://example.test/cat.gif")).toBeNull();
    });

    it("returns nothing for a url on another host that copies a giphy path", () => {
        // given / when / then
        expect(extractGiphyId("https://example.test/media/abc123/thing.gif")).toBeNull();
        expect(extractGiphyId("https://media.giphy.com.evil.test/media/abc123/giphy.gif")).toBeNull();
        expect(extractGiphyId("https://evil.test/x//i.giphy.com/abc123.gif")).toBeNull();
        expect(extractGiphyId("http://media.giphy.com/media/abc123/giphy.gif")).toBeNull();
    });
});
