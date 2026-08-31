import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./quote";

const globalFetch = vi.fn();

beforeEach(() => {
    globalFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(""),
    });
    vi.stubGlobal("fetch", globalFetch);
});

describe("the quote API", () => {
    it("searchQuotes defaults to umineko and a page of thirty", async () => {
        // given
        const params = { query: "gold" };

        // when
        await api.searchQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/umineko/search?q=gold&limit=30");
    });

    it("searchQuotes keeps the language filter for umineko", async () => {
        // given
        const params = { query: "gold", lang: "jp" };

        // when
        await api.searchQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith(
            "https://quotes.auaurora.moe/api/v1/umineko/search?q=gold&lang=jp&limit=30",
        );
    });

    it("searchQuotes drops the language filter for the other series", async () => {
        // given
        const params = { query: "gold", lang: "jp", series: "higurashi" as const };

        // when
        await api.searchQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/higurashi/search?q=gold&limit=30");
    });

    it("browseQuotes pages through a character's lines", async () => {
        // given
        const params = { character: "Beatrice", episode: 4, limit: 10, offset: 20 };

        // when
        await api.browseQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith(
            "https://quotes.auaurora.moe/api/v1/umineko/browse?character=Beatrice&episode=4&limit=10&offset=20",
        );
    });

    it("searchQuotes reports the status when the quote API fails", async () => {
        // given
        globalFetch.mockResolvedValue({ ok: false, status: 503 });

        // when
        const attempt = api.searchQuotes({ query: "gold" });

        // then
        await expect(attempt).rejects.toThrow("Quote API error: 503");
    });

    it("tryGetQuoteByAudioId asks for the quote by its audio id and language", async () => {
        // given
        const audioId = "ep2_042";

        // when
        await api.tryGetQuoteByAudioId("umineko", audioId, "jp");

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/umineko/quote/ep2_042?lang=jp");
    });

    it("tryGetQuoteByAudioId uses only the first id of a comma separated audio id", async () => {
        // given
        const audioId = " ep3_001 , ep3_002 ";

        // when
        await api.tryGetQuoteByAudioId("umineko", audioId, "en");

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/umineko/quote/ep3_001?lang=en");
    });

    it("tryGetQuoteByAudioId leaves the language off when it is not given", async () => {
        // given
        const audioId = "ep3_001";

        // when
        await api.tryGetQuoteByAudioId("umineko", audioId);

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/umineko/quote/ep3_001");
    });

    it("tryGetQuoteByAudioId asks nobody when the audio id is empty", async () => {
        // given
        const audioId = " , ";

        // when
        const result = await api.tryGetQuoteByAudioId("umineko", audioId);

        // then
        expect(result).toBeNull();
        expect(globalFetch).not.toHaveBeenCalled();
    });

    it("tryGetQuoteByAudioId answers null when the quote API has no such quote", async () => {
        // given
        globalFetch.mockResolvedValue({ ok: false, status: 404 });

        // when
        const result = await api.tryGetQuoteByAudioId("umineko", "missing");

        // then
        expect(result).toBeNull();
    });

    it("tryGetQuoteByIndex asks for the quote by its index and language", async () => {
        // given
        const index = 42;

        // when
        await api.tryGetQuoteByIndex("umineko", index, "en");

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/umineko/quote/index/42?lang=en");
    });

    it("tryGetQuoteByIndex leaves the language off when it is not given", async () => {
        // given
        const index = 3;

        // when
        await api.tryGetQuoteByIndex("umineko", index);

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/umineko/quote/index/3");
    });

    it("tryGetQuoteByIndex asks the series it was given", async () => {
        // given
        const series = "higurashi" as const;

        // when
        await api.tryGetQuoteByIndex(series, 5, "en");

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/higurashi/quote/index/5?lang=en");
    });

    it("tryGetQuoteByIndex answers null when the request itself fails", async () => {
        // given
        globalFetch.mockRejectedValue(new Error("network is out"));

        // when
        const result = await api.tryGetQuoteByIndex("umineko", 2);

        // then
        expect(result).toBeNull();
    });

    it("getCharacters merges the main and additional casts", async () => {
        // given
        globalFetch.mockResolvedValue({
            ok: true,
            json: () => Promise.resolve({ characters: { bea: "Beatrice" }, additional: { kanon: "Kanon" } }),
        });

        // when
        const result = await api.getCharacters("higurashi");

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/higurashi/characters");
        expect(result).toEqual({ bea: "Beatrice", kanon: "Kanon" });
    });

    it("getCharacterGroups falls back to empty groups when the payload is bare", async () => {
        // given
        globalFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });

        // when
        const result = await api.getCharacterGroups();

        // then
        expect(result).toEqual({ main: {}, additional: {} });
    });
});
