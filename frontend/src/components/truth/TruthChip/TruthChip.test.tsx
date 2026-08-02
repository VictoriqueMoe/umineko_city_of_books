import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { Quote } from "../../../types/api";
import { TruthChip } from "./TruthChip";

const longText = "The one who reaches the truth of the epitaph shall be crowned as the head of the family. ".repeat(3);

function makeQuote(overrides: Partial<Quote> = {}): Quote {
    return {
        text: "Without love, it cannot be seen.",
        textHtml: "<em>Without love, it cannot be seen.</em>",
        characterId: "beatrice",
        character: "Beatrice",
        audioId: "ep1_0001",
        episode: 4,
        contentType: "dialogue",
        hasRedTruth: false,
        hasBlueTruth: false,
        hasGoldTruth: false,
        hasPurpleTruth: false,
        index: 12,
        ...overrides,
    };
}

describe("TruthChip", () => {
    it("shows a short quote in full with no expand control", () => {
        // given
        const quote = makeQuote();

        // when
        renderWithProviders(<TruthChip quote={quote} />);

        // then
        expect(screen.getByText("Without love, it cannot be seen.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "show more" })).not.toBeInTheDocument();
    });

    it("renders the quote as plain text rather than as markup", () => {
        // given
        const quote = makeQuote();

        // when
        const { container } = renderWithProviders(<TruthChip quote={quote} />);

        // then
        expect(container.querySelector("em")).toBeNull();
        expect(screen.getByText("Without love, it cannot be seen.")).toBeInTheDocument();
    });

    it("leaves a quote of exactly a hundred characters untruncated", () => {
        // given
        const quote = makeQuote({ text: "z".repeat(100) });

        // when
        renderWithProviders(<TruthChip quote={quote} />);

        // then
        expect(screen.queryByRole("button", { name: "show more" })).not.toBeInTheDocument();
    });

    it("truncates a long quote and offers to show more", () => {
        // given
        const quote = makeQuote({ text: longText });

        // when
        const { container } = renderWithProviders(<TruthChip quote={quote} />);

        // then
        expect(container.textContent).toContain(`${longText.slice(0, 100)}...`);
        expect(container.textContent).not.toContain(longText);
        expect(screen.getByRole("button", { name: "show more" })).toBeInTheDocument();
    });

    it("reveals the whole quote once show more is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<TruthChip quote={makeQuote({ text: longText })} />);

        // when
        await user.click(screen.getByRole("button", { name: "show more" }));

        // then
        expect(container.textContent).toContain(longText);
        expect(screen.getByRole("button", { name: "show less" })).toBeInTheDocument();
    });

    it("collapses the quote again once show less is pressed", async () => {
        // given
        const user = userEvent.setup();
        const { container } = renderWithProviders(<TruthChip quote={makeQuote({ text: longText })} />);
        await user.click(screen.getByRole("button", { name: "show more" }));

        // when
        await user.click(screen.getByRole("button", { name: "show less" }));

        // then
        expect(container.textContent).not.toContain(longText);
        expect(screen.getByRole("button", { name: "show more" })).toBeInTheDocument();
    });

    it("shows the Japanese text for the language value the picker hands back", () => {
        // given
        const quote = makeQuote({ textJp: "愛がなければ、視えない" });

        // when
        renderWithProviders(<TruthChip quote={quote} lang="ja" />);

        // then
        expect(screen.getByText("愛がなければ、視えない")).toBeInTheDocument();
        expect(screen.queryByText("Without love, it cannot be seen.")).not.toBeInTheDocument();
    });

    it("falls back to the default text when the quote has no Japanese version", () => {
        // given
        const quote = makeQuote();

        // when
        renderWithProviders(<TruthChip quote={quote} lang="ja" />);

        // then
        expect(screen.getByText("Without love, it cannot be seen.")).toBeInTheDocument();
    });

    it("keeps the default text for any language other than Japanese", () => {
        // given
        const quote = makeQuote({ textJp: "愛がなければ、視えない" });

        // when
        renderWithProviders(<TruthChip quote={quote} lang="ru" />);

        // then
        expect(screen.getByText("Without love, it cannot be seen.")).toBeInTheDocument();
        expect(screen.queryByText("愛がなければ、視えない")).not.toBeInTheDocument();
    });

    it("shows the speaker and a short episode label", () => {
        // given
        const quote = makeQuote({ character: "Battler", episode: 6 });

        // when
        renderWithProviders(<TruthChip quote={quote} />);

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("EP6")).toBeInTheDocument();
    });

    it("prefers the arc name over the episode label", () => {
        // given
        const quote = makeQuote({ arc: "Watanagashi", episode: 2 });

        // when
        renderWithProviders(<TruthChip quote={quote} />);

        // then
        expect(screen.getByText("Watanagashi")).toBeInTheDocument();
        expect(screen.queryByText("EP2")).not.toBeInTheDocument();
    });

    it("shows the note that was attached to the evidence", () => {
        // given
        const note = "This contradicts the first twilight.";

        // when
        renderWithProviders(<TruthChip quote={makeQuote()} note={note} />);

        // then
        expect(screen.getByText(note)).toBeInTheDocument();
    });

    it("offers no remove control unless a remove handler is given", () => {
        // given
        const quote = makeQuote();

        // when
        renderWithProviders(<TruthChip quote={quote} />);

        // then
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("calls the remove handler when the remove control is pressed", async () => {
        // given
        const onRemove = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<TruthChip quote={makeQuote()} onRemove={onRemove} />);

        // when
        await user.click(screen.getByRole("button", { name: "✕" }));

        // then
        expect(onRemove).toHaveBeenCalledOnce();
    });
});
