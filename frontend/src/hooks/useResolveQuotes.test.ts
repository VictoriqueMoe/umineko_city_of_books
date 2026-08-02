import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Series } from "../api/endpoints";
import type { EvidenceItem, Quote } from "../types/api";
import { useResolveQuotes } from "./useResolveQuotes";

const QUOTE_API = "https://quotes.auaurora.moe/api/v1";

const fetchMock = vi.fn();

function makeQuote(overrides: Partial<Quote> = {}): Quote {
    return {
        text: "Without love, it cannot be seen.",
        textHtml: "<p>Without love, it cannot be seen.</p>",
        characterId: "beatrice",
        character: "Beatrice",
        audioId: "",
        episode: 1,
        contentType: "dialogue",
        hasRedTruth: false,
        hasBlueTruth: false,
        hasGoldTruth: false,
        hasPurpleTruth: false,
        index: 0,
        ...overrides,
    };
}

function makeEvidenceItem(overrides: Partial<EvidenceItem> = {}): EvidenceItem {
    return {
        id: 1,
        note: "",
        lang: "",
        sort_order: 0,
        ...overrides,
    };
}

function respondWith(body: unknown) {
    return Promise.resolve({ ok: true, json: () => Promise.resolve(body) });
}

function respondNotFound() {
    return Promise.resolve({ ok: false, json: () => Promise.resolve(null) });
}

interface ResolveProps {
    evidence: EvidenceItem[];
    series?: Series;
}

function setup(props: ResolveProps) {
    return renderHook(p => useResolveQuotes(p.evidence, p.series), { initialProps: props });
}

beforeEach(() => {
    fetchMock.mockImplementation(() => respondWith(makeQuote()));
    vi.stubGlobal("fetch", fetchMock);
});

describe("useResolveQuotes", () => {
    it("resolves nothing when there is no evidence", () => {
        // given
        const { result } = setup({ evidence: [] });

        // then
        expect(result.current.size).toBe(0);
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("keys a voiced quote by its audio id", async () => {
        // given
        const quote = makeQuote({ audioId: "ep2_042" });
        fetchMock.mockImplementation(() => respondWith(quote));

        // when
        const { result } = setup({ evidence: [makeEvidenceItem({ audio_id: "ep2_042", lang: "en" })] });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(result.current.get("audio:ep2_042")).toEqual(quote);
        expect(fetchMock).toHaveBeenCalledWith(`${QUOTE_API}/umineko/quote/ep2_042?lang=en`);
    });

    it("keys an unvoiced quote by its index", async () => {
        // given
        const quote = makeQuote({ index: 17 });
        fetchMock.mockImplementation(() => respondWith(quote));

        // when
        const { result } = setup({ evidence: [makeEvidenceItem({ quote_index: 17, lang: "jp" })] });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(result.current.get("index:17")).toEqual(quote);
        expect(fetchMock).toHaveBeenCalledWith(`${QUOTE_API}/umineko/quote/index/17?lang=jp`);
    });

    it("omits the language query when the evidence has none", async () => {
        // given
        const evidence = [makeEvidenceItem({ quote_index: 3, lang: "" })];

        // when
        const { result } = setup({ evidence });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(fetchMock).toHaveBeenCalledWith(`${QUOTE_API}/umineko/quote/index/3`);
    });

    it("uses only the first id of a comma separated audio id", async () => {
        // given
        const evidence = [makeEvidenceItem({ audio_id: " ep3_001 , ep3_002 " })];

        // when
        const { result } = setup({ evidence });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(fetchMock).toHaveBeenCalledWith(`${QUOTE_API}/umineko/quote/ep3_001`);
        expect(result.current.has("audio: ep3_001 , ep3_002 ")).toBe(true);
    });

    it("requests quotes from the series it was given", async () => {
        // given
        const evidence = [makeEvidenceItem({ quote_index: 5, lang: "en" })];

        // when
        const { result } = setup({ evidence, series: "ciconia" });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(fetchMock).toHaveBeenCalledWith(`${QUOTE_API}/ciconia/quote/index/5?lang=en`);
    });

    it("records a null for a quote the api will not return", async () => {
        // given
        fetchMock.mockImplementation(() => respondNotFound());

        // when
        const { result } = setup({ evidence: [makeEvidenceItem({ audio_id: "missing" })] });

        // then
        await waitFor(() => expect(result.current.has("audio:missing")).toBe(true));
        expect(result.current.get("audio:missing")).toBeNull();
    });

    it("records a null when the request itself fails", async () => {
        // given
        fetchMock.mockImplementation(() => Promise.reject(new Error("network is out")));

        // when
        const { result } = setup({ evidence: [makeEvidenceItem({ quote_index: 2 })] });

        // then
        await waitFor(() => expect(result.current.has("index:2")).toBe(true));
        expect(result.current.get("index:2")).toBeNull();
    });

    it("skips evidence that names neither an audio id nor an index", async () => {
        // given
        const evidence = [makeEvidenceItem({ id: 1 }), makeEvidenceItem({ id: 2, quote_index: 6 })];

        // when
        const { result } = setup({ evidence });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(fetchMock).toHaveBeenCalledOnce();
        expect(result.current.has("")).toBe(false);
    });

    it("fetches each key only once across rerenders", async () => {
        // given
        const { result, rerender } = setup({ evidence: [makeEvidenceItem({ quote_index: 1 })] });
        await waitFor(() => expect(result.current.size).toBe(1));

        // when
        rerender({ evidence: [makeEvidenceItem({ quote_index: 1 })] });

        // then
        expect(fetchMock).toHaveBeenCalledOnce();
    });

    it("fetches only the newly added evidence and keeps what it already resolved", async () => {
        // given
        const first = makeEvidenceItem({ id: 1, quote_index: 1 });
        const { result, rerender } = setup({ evidence: [first] });
        await waitFor(() => expect(result.current.size).toBe(1));

        // when
        rerender({ evidence: [first, makeEvidenceItem({ id: 2, quote_index: 2 })] });

        // then
        await waitFor(() => expect(result.current.size).toBe(2));
        expect(fetchMock).toHaveBeenCalledTimes(2);
        expect(fetchMock).toHaveBeenLastCalledWith(`${QUOTE_API}/umineko/quote/index/2`);
    });
});
