import { describe, expect, it } from "vitest";
import { detectWaifuvaultMedia } from "./detect";

describe("detectWaifuvaultMedia", () => {
    it("recognises every supported image extension", () => {
        // given
        const extensions = ["jpg", "jpeg", "png", "gif", "webp", "avif"];

        // when / then
        for (const ext of extensions) {
            expect(detectWaifuvaultMedia(`https://waifuvault.moe/f/beatrice.${ext}`)).toBe("image");
        }
    });

    it("recognises every supported video extension", () => {
        // given
        const extensions = ["mp4", "webm", "mov", "m4v"];

        // when / then
        for (const ext of extensions) {
            expect(detectWaifuvaultMedia(`https://waifuvault.moe/f/beatrice.${ext}`)).toBe("video");
        }
    });

    it("ignores the case of the extension and the host", () => {
        // given
        const shouty = "https://WAIFUVAULT.MOE/f/BEATRICE.PNG";

        // when
        const kind = detectWaifuvaultMedia(shouty);

        // then
        expect(kind).toBe("image");
    });

    it("accepts any subdomain of the vault", () => {
        // given
        const urls = ["https://cdn.waifuvault.moe/f/a.png", "https://files.cdn.waifuvault.moe/f/a.mp4"];

        // when
        const kinds = urls.map(detectWaifuvaultMedia);

        // then
        expect(kinds).toEqual(["image", "video"]);
    });

    it("ignores a query string and a fragment when reading the extension", () => {
        // given
        const withQuery = "https://waifuvault.moe/f/a.png?download=1&token=beato";
        const withHash = "https://waifuvault.moe/f/a.webm#t=10";

        // when / then
        expect(detectWaifuvaultMedia(withQuery)).toBe("image");
        expect(detectWaifuvaultMedia(withHash)).toBe("video");
    });

    it("only checks the host, so a plain http url is still recognised", () => {
        // given
        const insecure = "http://waifuvault.moe/f/a.png";

        // when
        const kind = detectWaifuvaultMedia(insecure);

        // then
        expect(kind).toBe("image");
    });

    it("rejects hosts that merely look like the vault", () => {
        // given
        const impostors = [
            "https://evilwaifuvault.moe/f/a.png",
            "https://waifuvault.moe.attacker.example/f/a.png",
            "https://waifuvault.example/f/a.png",
            "https://media.giphy.com/media/abc123/giphy.gif",
        ];

        // when / then
        for (const url of impostors) {
            expect(detectWaifuvaultMedia(url)).toBeNull();
        }
    });

    it("rejects extensions it does not know how to embed", () => {
        // given
        const unsupported = ["mkv", "avi", "svg", "pdf", "mp3", "html"];

        // when / then
        for (const ext of unsupported) {
            expect(detectWaifuvaultMedia(`https://waifuvault.moe/f/a.${ext}`)).toBeNull();
        }
    });

    it("rejects a path with no extension at all", () => {
        // given
        const urls = ["https://waifuvault.moe/f/abcdef", "https://waifuvault.moe/", "https://waifuvault.moe"];

        // when / then
        for (const url of urls) {
            expect(detectWaifuvaultMedia(url)).toBeNull();
        }
    });

    it("rejects a trailing dot with nothing after it", () => {
        // given
        const url = "https://waifuvault.moe/f/beatrice.";

        // when
        const kind = detectWaifuvaultMedia(url);

        // then
        expect(kind).toBeNull();
    });

    it("rejects a dot that belongs to a directory rather than the file", () => {
        // given
        const urls = ["https://waifuvault.moe/v1.0/beatrice", "https://waifuvault.moe/f/beatrice.png/preview"];

        // when / then
        for (const url of urls) {
            expect(detectWaifuvaultMedia(url)).toBeNull();
        }
    });

    it("rejects anything that is not an absolute url", () => {
        // given
        const notURLs = ["", "   ", "not a url", "/f/beatrice.png", "waifuvault.moe/f/beatrice.png"];

        // when / then
        for (const value of notURLs) {
            expect(detectWaifuvaultMedia(value)).toBeNull();
        }
    });
});
