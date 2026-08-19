import { describe, expect, it } from "vitest";
import { previewableURLs } from "./urls";

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

    it("caps the number of previews", () => {
        const body = ["a", "b", "c", "d", "e", "f", "g"].map(h => `https://${h}.example`).join(" ");
        expect(previewableURLs(body)).toHaveLength(5);
    });

    it("returns nothing when the limit is zero or negative", () => {
        expect(previewableURLs("https://a.example", 0)).toEqual([]);
        expect(previewableURLs("https://a.example", -1)).toEqual([]);
    });

    it("does not carry regex state between calls", () => {
        const long = "padding padding padding https://first.example/a/long/path";
        previewableURLs(long);

        expect(previewableURLs("https://b.example")).toEqual(["https://b.example"]);
    });
});
