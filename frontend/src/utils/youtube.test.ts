import { describe, expect, it } from "vitest";
import { extractYouTubeIDs } from "./youtube";

describe("extractYouTubeIDs", () => {
    it("returns nothing for text without a video link", () => {
        // given / when / then
        expect(extractYouTubeIDs("")).toEqual([]);
        expect(extractYouTubeIDs("no links here at all")).toEqual([]);
    });

    it("reads the id out of every supported link shape", () => {
        // given / when / then
        expect(extractYouTubeIDs("https://www.youtube.com/watch?v=dQw4w9WgXcQ")).toEqual(["dQw4w9WgXcQ"]);
        expect(extractYouTubeIDs("https://youtube.com/watch?v=dQw4w9WgXcQ")).toEqual(["dQw4w9WgXcQ"]);
        expect(extractYouTubeIDs("https://m.youtube.com/watch?v=dQw4w9WgXcQ")).toEqual(["dQw4w9WgXcQ"]);
        expect(extractYouTubeIDs("https://www.youtube.com/embed/dQw4w9WgXcQ")).toEqual(["dQw4w9WgXcQ"]);
        expect(extractYouTubeIDs("https://www.youtube.com/shorts/dQw4w9WgXcQ")).toEqual(["dQw4w9WgXcQ"]);
        expect(extractYouTubeIDs("https://youtu.be/dQw4w9WgXcQ")).toEqual(["dQw4w9WgXcQ"]);
        expect(extractYouTubeIDs("http://youtu.be/dQw4w9WgXcQ")).toEqual(["dQw4w9WgXcQ"]);
    });

    it("finds a link embedded in a sentence and ignores trailing query parameters", () => {
        // given
        const text = "watch this https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=42s before bed";

        // when
        const ids = extractYouTubeIDs(text);

        // then
        expect(ids).toEqual(["dQw4w9WgXcQ"]);
    });

    it("accepts ids containing hyphens and underscores", () => {
        // given
        const text = "https://youtu.be/a_b-c123456";

        // when
        const ids = extractYouTubeIDs(text);

        // then
        expect(ids).toEqual(["a_b-c123456"]);
    });

    it("ignores links that are not youtube or whose id is too short", () => {
        // given / when / then
        expect(extractYouTubeIDs("https://vimeo.com/watch?v=dQw4w9WgXcQ")).toEqual([]);
        expect(extractYouTubeIDs("https://notyoutube.com/watch?v=dQw4w9WgXcQ")).toEqual([]);
        expect(extractYouTubeIDs("https://youtube.com.evil.test/watch?v=dQw4w9WgXcQ")).toEqual([]);
        expect(extractYouTubeIDs("https://www.youtube.com/watch?v=tooshort")).toEqual([]);
        expect(extractYouTubeIDs("https://www.youtube.com/playlist?list=PLdQw4w9WgXcQ")).toEqual([]);
        expect(extractYouTubeIDs("www.youtube.com/watch?v=dQw4w9WgXcQ")).toEqual([]);
    });

    it("takes only the first eleven characters when the token runs on", () => {
        // given
        const text = "https://youtu.be/dQw4w9WgXcQextrastuff";

        // when
        const ids = extractYouTubeIDs(text);

        // then
        expect(ids).toEqual(["dQw4w9WgXcQ"]);
    });

    it("keeps the first appearance order and drops duplicates", () => {
        // given
        const text = [
            "https://youtu.be/aaaaaaaaaaa",
            "https://www.youtube.com/watch?v=bbbbbbbbbbb",
            "https://www.youtube.com/embed/aaaaaaaaaaa",
        ].join(" ");

        // when
        const ids = extractYouTubeIDs(text, 5);

        // then
        expect(ids).toEqual(["aaaaaaaaaaa", "bbbbbbbbbbb"]);
    });

    it("stops after two videos by default", () => {
        // given
        const text = [
            "https://youtu.be/aaaaaaaaaaa",
            "https://youtu.be/bbbbbbbbbbb",
            "https://youtu.be/ccccccccccc",
        ].join(" ");

        // when
        const ids = extractYouTubeIDs(text);

        // then
        expect(ids).toEqual(["aaaaaaaaaaa", "bbbbbbbbbbb"]);
    });

    it("honours a custom limit", () => {
        // given
        const text = [
            "https://youtu.be/aaaaaaaaaaa",
            "https://youtu.be/bbbbbbbbbbb",
            "https://youtu.be/ccccccccccc",
        ].join(" ");

        // when / then
        expect(extractYouTubeIDs(text, 1)).toEqual(["aaaaaaaaaaa"]);
        expect(extractYouTubeIDs(text, 3)).toEqual(["aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"]);
        expect(extractYouTubeIDs(text, 10)).toEqual(["aaaaaaaaaaa", "bbbbbbbbbbb", "ccccccccccc"]);
    });

    it("returns nothing when the limit leaves no room for a video", () => {
        // given
        const text = "https://youtu.be/aaaaaaaaaaa https://youtu.be/bbbbbbbbbbb";

        // when / then
        expect(extractYouTubeIDs(text, 0)).toEqual([]);
        expect(extractYouTubeIDs(text, -1)).toEqual([]);
    });

    it("does not leak scanning position between calls when an earlier call hit its limit", () => {
        // given
        const text = "https://youtu.be/aaaaaaaaaaa https://youtu.be/bbbbbbbbbbb https://youtu.be/ccccccccccc";
        const first = extractYouTubeIDs(text, 1);

        // when
        const second = extractYouTubeIDs(text, 1);
        const third = extractYouTubeIDs("https://youtu.be/ddddddddddd", 1);

        // then
        expect(first).toEqual(["aaaaaaaaaaa"]);
        expect(second).toEqual(["aaaaaaaaaaa"]);
        expect(third).toEqual(["ddddddddddd"]);
    });
});
