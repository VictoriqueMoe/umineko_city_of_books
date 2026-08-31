import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { EvidenceItem, Quote, Series } from "../types/api";
import { providerWrapper } from "../test-utils/render";
import { useEvidence } from "./useEvidence";

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

interface EvidenceProps {
    initial?: EvidenceItem[];
    series?: Series;
}

function setup(props: EvidenceProps = {}) {
    return renderHook(p => useEvidence(p.initial, p.series), { initialProps: props, wrapper: providerWrapper() });
}

beforeEach(() => {
    mocks.tryGetQuoteByAudioId.mockResolvedValue(makeQuote());
    mocks.tryGetQuoteByIndex.mockResolvedValue(makeQuote());
});

describe("useEvidence", () => {
    it("starts with nothing selected and the picker closed", () => {
        // given
        const { result } = setup();

        // then
        expect(result.current.evidence).toEqual([]);
        expect(result.current.selectedKeys).toEqual([]);
        expect(result.current.pickerOpen).toBe(false);
        expect(mocks.tryGetQuoteByAudioId).not.toHaveBeenCalled();
        expect(mocks.tryGetQuoteByIndex).not.toHaveBeenCalled();
    });

    it("adds a quote in the requested language and closes the picker", () => {
        // given
        const quote = makeQuote({ audioId: "ep1_001" });
        const { result } = setup();
        act(() => {
            result.current.openPicker();
        });

        // when
        act(() => {
            result.current.addQuote(quote, "jp");
        });

        // then
        expect(result.current.evidence).toEqual([{ quote, note: "", lang: "jp" }]);
        expect(result.current.pickerOpen).toBe(false);
    });

    it("defaults a newly added quote to english", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "ep1_001" }));
        });

        // then
        expect(result.current.evidence[0].lang).toBe("en");
    });

    it("ignores a quote that has already been selected", () => {
        // given
        const quote = makeQuote({ audioId: "ep1_001" });
        const { result } = setup();

        // when
        act(() => {
            result.current.addQuote(quote);
        });
        act(() => {
            result.current.addQuote(quote, "jp");
        });

        // then
        expect(result.current.evidence).toHaveLength(1);
        expect(result.current.evidence[0].lang).toBe("en");
    });

    it("tells quotes without an audio id apart by their index", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.addQuote(makeQuote({ index: 3 }));
        });
        act(() => {
            result.current.addQuote(makeQuote({ index: 4 }));
        });
        act(() => {
            result.current.addQuote(makeQuote({ index: 3 }));
        });

        // then
        expect(result.current.selectedKeys).toEqual(["index:3", "index:4"]);
    });

    it("updates the note of the quote at a given position", () => {
        // given
        const { result } = setup();
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "a" }));
        });
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "b" }));
        });

        // when
        act(() => {
            result.current.updateNote(1, "this is the culprit");
        });

        // then
        expect(result.current.evidence[0].note).toBe("");
        expect(result.current.evidence[1].note).toBe("this is the culprit");
    });

    it("ignores a note aimed past the end of the selection", () => {
        // given
        const quote = makeQuote({ audioId: "a" });
        const { result } = setup();
        act(() => {
            result.current.addQuote(quote);
        });

        // when
        act(() => {
            result.current.updateNote(4, "nowhere");
        });

        // then
        expect(result.current.evidence).toEqual([{ quote, note: "", lang: "en" }]);
        expect(result.current.selectedKeys).toEqual(["audio:a"]);
    });

    it("ignores a note aimed at a negative position", () => {
        // given
        const { result } = setup();
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "a" }));
        });

        // when
        act(() => {
            result.current.updateNote(-1, "nowhere");
        });

        // then
        expect(result.current.evidence).toHaveLength(1);
        expect(Object.keys(result.current.evidence)).toEqual(["0"]);
        expect(result.current.evidence[0].note).toBe("");
    });

    it("removes the quote at a given position", () => {
        // given
        const { result } = setup();
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "a" }));
        });
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "b" }));
        });

        // when
        act(() => {
            result.current.removeAt(0);
        });

        // then
        expect(result.current.selectedKeys).toEqual(["audio:b"]);
    });

    it("clears every selected quote", () => {
        // given
        const { result } = setup();
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "a" }));
        });

        // when
        act(() => {
            result.current.clear();
        });

        // then
        expect(result.current.evidence).toEqual([]);
    });

    it("opens and closes the picker on demand", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.openPicker();
        });

        // then
        expect(result.current.pickerOpen).toBe(true);
        act(() => {
            result.current.closePicker();
        });
        expect(result.current.pickerOpen).toBe(false);
    });

    it("sends an audio id for voiced quotes and an index for the rest", () => {
        // given
        const { result } = setup();
        act(() => {
            result.current.addQuote(makeQuote({ audioId: "ep1_001", index: 7 }), "jp");
        });
        act(() => {
            result.current.addQuote(makeQuote({ index: 9 }));
        });
        act(() => {
            result.current.updateNote(0, "listen closely");
        });

        // when
        const input = result.current.toInput();

        // then
        expect(input).toEqual([
            { audio_id: "ep1_001", quote_index: undefined, note: "listen closely", lang: "jp" },
            { audio_id: undefined, quote_index: 9, note: "", lang: "en" },
        ]);
    });

    it("resolves initial evidence that points at an audio id", async () => {
        // given
        const quote = makeQuote({ audioId: "ep2_042" });
        mocks.tryGetQuoteByAudioId.mockResolvedValue(quote);

        // when
        const { result } = setup({ initial: [makeEvidenceItem({ audio_id: "ep2_042", note: "here", lang: "jp" })] });

        // then
        await waitFor(() => expect(result.current.evidence).toHaveLength(1));
        expect(mocks.tryGetQuoteByAudioId).toHaveBeenCalledWith("umineko", "ep2_042", "jp");
        expect(result.current.evidence[0]).toEqual({ quote, note: "here", lang: "jp" });
    });

    it("resolves initial evidence that points at a quote index", async () => {
        // given
        const quote = makeQuote({ index: 42 });
        mocks.tryGetQuoteByIndex.mockResolvedValue(quote);

        // when
        const { result } = setup({ initial: [makeEvidenceItem({ quote_index: 42, note: "index one" })] });

        // then
        await waitFor(() => expect(result.current.evidence).toHaveLength(1));
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledWith("umineko", 42, "en");
        expect(result.current.evidence[0].lang).toBe("en");
    });

    it("hands a comma separated audio id over whole", async () => {
        // given
        const items = [makeEvidenceItem({ audio_id: " ep3_001 , ep3_002 ", lang: "en" })];

        // when
        const { result } = setup({ initial: items });

        // then
        await waitFor(() => expect(result.current.evidence).toHaveLength(1));
        expect(mocks.tryGetQuoteByAudioId).toHaveBeenCalledWith("umineko", " ep3_001 , ep3_002 ", "en");
    });

    it("requests quotes from the series it was given", async () => {
        // given
        const items = [makeEvidenceItem({ quote_index: 5 })];

        // when
        const { result } = setup({ initial: items, series: "higurashi" });

        // then
        await waitFor(() => expect(result.current.evidence).toHaveLength(1));
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledWith("higurashi", 5, "en");
    });

    it("drops initial evidence whose quote cannot be fetched", async () => {
        // given
        const good = makeQuote({ audioId: "ep1_good" });
        mocks.tryGetQuoteByAudioId.mockImplementation((_series: Series, audioId: string) => {
            if (audioId.includes("ep1_bad")) {
                return Promise.resolve(null);
            }
            return Promise.resolve(good);
        });

        // when
        const { result } = setup({
            initial: [
                makeEvidenceItem({ id: 1, audio_id: "ep1_bad" }),
                makeEvidenceItem({ id: 2, audio_id: "ep1_good" }),
            ],
        });

        // then
        await waitFor(() => expect(result.current.evidence).toHaveLength(1));
        expect(result.current.selectedKeys).toEqual(["audio:ep1_good"]);
    });

    it("resolves initial evidence once even when the array identity changes", async () => {
        // given
        const { result, rerender } = setup({ initial: [makeEvidenceItem({ quote_index: 1 })] });
        await waitFor(() => expect(result.current.evidence).toHaveLength(1));

        // when
        rerender({ initial: [makeEvidenceItem({ quote_index: 1 })] });

        // then
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledOnce();
    });

    it("resolves evidence that only arrives after the first render", async () => {
        // given
        const { result, rerender } = setup({ initial: [] });
        expect(mocks.tryGetQuoteByIndex).not.toHaveBeenCalled();

        // when
        rerender({ initial: [makeEvidenceItem({ quote_index: 8 })] });

        // then
        await waitFor(() => expect(result.current.evidence).toHaveLength(1));
        expect(mocks.tryGetQuoteByIndex).toHaveBeenCalledWith("umineko", 8, "en");
    });
});
