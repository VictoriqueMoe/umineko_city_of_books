import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { EvidenceItem, Quote } from "../../../types/api";
import { useResolveQuotes } from "../../../hooks/useResolveQuotes";
import { EvidenceList } from "./EvidenceList";

vi.mock("../../../hooks/useResolveQuotes", () => ({
    useResolveQuotes: vi.fn(() => new Map<string, Quote | null>()),
}));

const resolveQuotes = vi.mocked(useResolveQuotes);

function makeQuote(overrides: Partial<Quote> = {}): Quote {
    return {
        text: "Without love, it cannot be seen.",
        textHtml: "<p>Without love, it cannot be seen.</p>",
        characterId: "beatrice",
        character: "Beatrice",
        audioId: "ep1_0001",
        episode: 1,
        contentType: "dialogue",
        hasRedTruth: false,
        hasBlueTruth: false,
        hasGoldTruth: false,
        hasPurpleTruth: false,
        index: 12,
        ...overrides,
    };
}

function makeEvidence(overrides: Partial<EvidenceItem> = {}): EvidenceItem {
    return {
        id: 1,
        audio_id: "ep1_0001",
        note: "",
        lang: "en",
        sort_order: 0,
        ...overrides,
    };
}

function resolvesTo(entries: [string, Quote][]) {
    resolveQuotes.mockReturnValue(new Map<string, Quote | null>(entries));
}

describe("EvidenceList", () => {
    it("renders nothing at all when a response cites no evidence", () => {
        // given
        resolvesTo([]);

        // when
        const { container } = renderWithProviders(<EvidenceList evidence={[]} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows the quote and the note explaining why it matters", () => {
        // given
        const quote = makeQuote();
        resolvesTo([["audio:ep1_0001", quote]]);

        // when
        renderWithProviders(<EvidenceList evidence={[makeEvidence({ note: "This denies the closed room." })]} />);

        // then
        expect(screen.getByText("Evidence")).toBeInTheDocument();
        expect(screen.getByText(quote.text)).toBeInTheDocument();
        expect(screen.getByText("This denies the closed room.")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("shows a placeholder for a quote that has not come back yet", () => {
        // given
        resolvesTo([]);

        // when
        renderWithProviders(<EvidenceList evidence={[makeEvidence()]} />);

        // then
        expect(screen.getByText("Loading quote...")).toBeInTheDocument();
    });

    it("looks up audio backed evidence by audio id and the rest by quote index", () => {
        // given
        const spoken = makeQuote({ text: "The witch smiles." });
        const narrated = makeQuote({ audioId: "", index: 7, text: "The seagulls cry." });
        resolvesTo([
            ["audio:ep1_0001", spoken],
            ["index:7", narrated],
        ]);

        // when
        renderWithProviders(
            <EvidenceList
                evidence={[makeEvidence(), makeEvidence({ id: 2, audio_id: undefined, quote_index: 7, sort_order: 1 })]}
            />,
        );

        // then
        expect(screen.getByText("The witch smiles.")).toBeInTheDocument();
        expect(screen.getByText("The seagulls cry.")).toBeInTheDocument();
        expect(screen.queryByText("Loading quote...")).not.toBeInTheDocument();
    });

    it("still shows a placeholder for the one quote that failed while showing the others", () => {
        // given
        const resolved = makeQuote({ text: "The witch smiles." });
        resolvesTo([["audio:ep1_0001", resolved]]);

        // when
        renderWithProviders(
            <EvidenceList
                evidence={[makeEvidence(), makeEvidence({ id: 2, audio_id: undefined, quote_index: 7, sort_order: 1 })]}
            />,
        );

        // then
        expect(screen.getByText("The witch smiles.")).toBeInTheDocument();
        expect(screen.getByText("Loading quote...")).toBeInTheDocument();
    });

    it("resolves the quotes against the series it was given", () => {
        // given
        resolvesTo([]);
        const evidence = [makeEvidence()];

        // when
        renderWithProviders(<EvidenceList evidence={evidence} series="higurashi" />);

        // then
        expect(resolveQuotes).toHaveBeenCalledWith(evidence, "higurashi");
    });

    it("resolves the quotes against umineko when no series is given", () => {
        // given
        resolvesTo([]);
        const evidence = [makeEvidence()];

        // when
        renderWithProviders(<EvidenceList evidence={evidence} />);

        // then
        expect(resolveQuotes).toHaveBeenCalledWith(evidence, "umineko");
    });
});
