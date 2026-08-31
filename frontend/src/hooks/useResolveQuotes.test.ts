import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { EvidenceItem, Quote, Series } from "../types/api";
import { providerWrapper } from "../test-utils/render";
import { useResolveQuotes } from "./useResolveQuotes";

const mocks = vi.hoisted(() => ({
    tryGetQuoteByAudioId: vi.fn(),
    tryGetQuoteByIndex: vi.fn(),
    searchQuotes: vi.fn(),
    browseQuotes: vi.fn(),
    getCharacters: vi.fn(),
    getCharacterGroups: vi.fn(),
}));

vi.mock("../api/endpoints/quote", () => mocks);

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

interface ResolveProps {
    evidence: EvidenceItem[];
    series?: Series;
}

function setup(props: ResolveProps) {
    return renderHook(p => useResolveQuotes(p.evidence, p.series), {
        initialProps: props,
        wrapper: providerWrapper(),
    });
}

beforeEach(() => {
    mocks.tryGetQuoteByAudioId.mockResolvedValue(makeQuote());
    mocks.tryGetQuoteByIndex.mockResolvedValue(makeQuote());
});

describe("useResolveQuotes", () => {
    it("resolves nothing when there is no evidence", () => {
        // given
        const { result } = setup({ evidence: [] });

        // then
        expect(result.current.size).toBe(0);
        expect(mocks.tryGetQuoteByAudioId).not.toHaveBeenCalled();
        expect(mocks.tryGetQuoteByIndex).not.toHaveBeenCalled();
    });

    it("keys a voiced quote by its audio id", async () => {
        // given
        const quote = makeQuote({ audioId: "ep2_042" });
        mocks.tryGetQuoteByAudioId.mockResolvedValue(quote);

        // when
        const { result } = setup({ evidence: [makeEvidenceItem({ audio_id: "ep2_042", lang: "en" })] });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(result.current.get("audio:ep2_042")).toEqual(quote);
        expect(mocks.tryGetQuoteByAudioId).toHaveBeenCalledWith("umineko", "ep2_042", "en");
    });

    it("keys an unvoiced quote by its index", async () => {
        // given
        const quote = makeQuote({ index: 17 });
        mocks.tryGetQuoteByIndex.mockResolvedValue(quote);

        // when
        const { result } = setup({ evidence: [makeEvidenceItem({ quote_index: 17, lang: "jp" })] });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(result.current.get("index:17")).toEqual(quote);
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledWith("umineko", 17, "jp");
    });

    it("asks for no language at all when the evidence has none", async () => {
        // given
        const evidence = [makeEvidenceItem({ quote_index: 3, lang: "" })];

        // when
        const { result } = setup({ evidence });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledWith("umineko", 3, undefined);
    });

    it("hands a comma separated audio id over whole and keys it by the whole string", async () => {
        // given
        const evidence = [makeEvidenceItem({ audio_id: " ep3_001 , ep3_002 " })];

        // when
        const { result } = setup({ evidence });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(mocks.tryGetQuoteByAudioId).toHaveBeenCalledWith("umineko", " ep3_001 , ep3_002 ", undefined);
        expect(result.current.has("audio: ep3_001 , ep3_002 ")).toBe(true);
    });

    it("requests quotes from the series it was given", async () => {
        // given
        const evidence = [makeEvidenceItem({ quote_index: 5, lang: "en" })];

        // when
        const { result } = setup({ evidence, series: "ciconia" });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledWith("ciconia", 5, "en");
    });

    it("records a null for a quote the api will not return", async () => {
        // given
        mocks.tryGetQuoteByAudioId.mockResolvedValue(null);

        // when
        const { result } = setup({ evidence: [makeEvidenceItem({ audio_id: "missing" })] });

        // then
        await waitFor(() => expect(result.current.has("audio:missing")).toBe(true));
        expect(result.current.get("audio:missing")).toBeNull();
    });

    it("skips evidence that names neither an audio id nor an index", async () => {
        // given
        const evidence = [makeEvidenceItem({ id: 1 }), makeEvidenceItem({ id: 2, quote_index: 6 })];

        // when
        const { result } = setup({ evidence });

        // then
        await waitFor(() => expect(result.current.size).toBe(1));
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledOnce();
        expect(result.current.has("")).toBe(false);
    });

    it("fetches each key only once across rerenders", async () => {
        // given
        const { result, rerender } = setup({ evidence: [makeEvidenceItem({ quote_index: 1 })] });
        await waitFor(() => expect(result.current.size).toBe(1));

        // when
        rerender({ evidence: [makeEvidenceItem({ quote_index: 1 })] });

        // then
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledOnce();
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
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledTimes(2);
        expect(mocks.tryGetQuoteByIndex).toHaveBeenLastCalledWith("umineko", 2, undefined);
    });
});
