import { describe, expect, it } from "vitest";
import { previewableURLs, trimTrailingPunctuation } from "./links";

describe("previewableURLs", () => {
    it("returns nothing when the body has no links", () => {
        expect(previewableURLs("just some words")).toEqual([]);
    });

    it("extracts every http and https url", () => {
        expect(previewableURLs("see http://a.example and https://b.example now")).toEqual([
            "http://a.example",
            "https://b.example",
        ]);
    });

    it("keeps only the first occurrence of a repeated url", () => {
        expect(previewableURLs("https://a.example https://a.example https://b.example")).toEqual([
            "https://a.example",
            "https://b.example",
        ]);
    });

    it("skips waifuvault media because linkify renders it inline", () => {
        expect(previewableURLs("https://waifuvault.moe/f/1/cat.png and https://b.example")).toEqual([
            "https://b.example",
        ]);
    });

    it("keeps waifuvault media when the caller asks for inline media", () => {
        expect(
            previewableURLs("https://waifuvault.moe/f/1/cat.png and https://b.example", { keepInlineMedia: true }),
        ).toEqual(["https://waifuvault.moe/f/1/cat.png", "https://b.example"]);
    });

    it("caps the number of previews", () => {
        const body = ["a", "b", "c", "d", "e", "f", "g"].map(h => `https://${h}.example`).join(" ");
        expect(previewableURLs(body)).toHaveLength(5);
    });

    it("returns nothing when the limit is zero or negative", () => {
        expect(previewableURLs("https://a.example", 0)).toEqual([]);
        expect(previewableURLs("https://a.example", -1)).toEqual([]);
        expect(previewableURLs("https://a.example", { limit: 0 })).toEqual([]);
    });

    it("does not carry regex state between calls", () => {
        const long = "padding padding padding https://first.example/a/long/path";
        previewableURLs(long);

        expect(previewableURLs("https://b.example")).toEqual(["https://b.example"]);
    });

    it("drops a full stop that ends the sentence rather than the link", () => {
        expect(previewableURLs("have a look at https://example.com/page.")).toEqual(["https://example.com/page"]);
    });

    it("treats a link written twice with and without trailing punctuation as one link", () => {
        expect(previewableURLs("https://a.example/x. and https://a.example/x")).toEqual(["https://a.example/x"]);
    });
});

describe("trimTrailingPunctuation", () => {
    it("leaves a plain url alone", () => {
        expect(trimTrailingPunctuation("https://example.com/page")).toBe("https://example.com/page");
    });

    it("strips sentence punctuation", () => {
        expect(trimTrailingPunctuation("https://example.com/page.")).toBe("https://example.com/page");
        expect(trimTrailingPunctuation("https://example.com/page,")).toBe("https://example.com/page");
        expect(trimTrailingPunctuation("https://example.com/page?!")).toBe("https://example.com/page");
    });

    it("keeps a closing bracket that the url opened itself", () => {
        expect(trimTrailingPunctuation("https://en.wikipedia.org/wiki/Umineko_(series)")).toBe(
            "https://en.wikipedia.org/wiki/Umineko_(series)",
        );
    });

    it("drops a closing bracket that belongs to the sentence", () => {
        expect(trimTrailingPunctuation("https://example.com/page)")).toBe("https://example.com/page");
    });

    it("drops a bracket and the punctuation behind it", () => {
        expect(trimTrailingPunctuation("https://example.com/page).")).toBe("https://example.com/page");
    });
});
