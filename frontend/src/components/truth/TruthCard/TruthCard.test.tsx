import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { Quote } from "../../../types/api";
import { TruthCard } from "./TruthCard";

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

describe("TruthCard", () => {
    it("renders the quote markup rather than escaping it", () => {
        // given
        const quote = makeQuote();

        // when
        renderWithProviders(<TruthCard quote={quote} />);

        // then
        expect(screen.getByText("Without love, it cannot be seen.").tagName).toBe("EM");
    });

    it("shows the speaker and the episode number", () => {
        // given
        const quote = makeQuote({ character: "Battler", episode: 5 });

        // when
        renderWithProviders(<TruthCard quote={quote} />);

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.getByText("Episode 5")).toBeInTheDocument();
    });

    it("prefers the arc name over the episode number when the quote has one", () => {
        // given
        const quote = makeQuote({ arc: "Onikakushi", episode: 2 });

        // when
        renderWithProviders(<TruthCard quote={quote} />);

        // then
        expect(screen.getByText("Onikakushi")).toBeInTheDocument();
        expect(screen.queryByText("Episode 2")).not.toBeInTheDocument();
    });

    it("shows the Japanese markup when Japanese is the chosen language", () => {
        // given
        const quote = makeQuote({ textJp: "愛がなければ", textJpHtml: "<b>愛の真実</b>" });

        // when
        renderWithProviders(<TruthCard quote={quote} lang="ja" />);

        // then
        expect(screen.getByText("愛の真実")).toBeInTheDocument();
        expect(screen.queryByText("Without love, it cannot be seen.")).not.toBeInTheDocument();
    });

    it("falls back to the plain Japanese text when there is no Japanese markup", () => {
        // given
        const quote = makeQuote({ textJp: "愛がなければ" });

        // when
        renderWithProviders(<TruthCard quote={quote} lang="ja" />);

        // then
        expect(screen.getByText("愛がなければ")).toBeInTheDocument();
    });

    it("keeps the default text when Japanese is asked for but the quote has none", () => {
        // given
        const quote = makeQuote();

        // when
        renderWithProviders(<TruthCard quote={quote} lang="ja" />);

        // then
        expect(screen.getByText("Without love, it cannot be seen.")).toBeInTheDocument();
    });

    it("keeps the default text for any other language", () => {
        // given
        const quote = makeQuote({ textJpHtml: "<b>愛の真実</b>" });

        // when
        renderWithProviders(<TruthCard quote={quote} lang="ru" />);

        // then
        expect(screen.getByText("Without love, it cannot be seen.")).toBeInTheDocument();
    });

    it("is not interactive when no click handler is given", () => {
        // given
        const quote = makeQuote();

        // when
        renderWithProviders(<TruthCard quote={quote} />);

        // then
        expect(screen.queryByRole("button")).not.toBeInTheDocument();
    });

    it("behaves as a button when a click handler is given", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<TruthCard quote={makeQuote()} onClick={onClick} />);

        // when
        await user.click(screen.getByRole("button"));

        // then
        expect(onClick).toHaveBeenCalledOnce();
    });

    it("activates on enter when it has keyboard focus", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<TruthCard quote={makeQuote()} onClick={onClick} />);

        // when
        await user.tab();
        await user.keyboard("{Enter}");

        // then
        expect(screen.getByRole("button")).toHaveFocus();
        expect(onClick).toHaveBeenCalledOnce();
    });

    it("activates on the space bar when it has keyboard focus", async () => {
        // given
        const onClick = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<TruthCard quote={makeQuote()} onClick={onClick} />);

        // when
        await user.tab();
        await user.keyboard(" ");

        // then
        expect(onClick).toHaveBeenCalledOnce();
    });

    it("ignores keys that are not enter or space", () => {
        // given
        const onClick = vi.fn();
        renderWithProviders(<TruthCard quote={makeQuote()} onClick={onClick} />);

        // when
        fireEvent.keyDown(screen.getByRole("button"), { key: "a" });

        // then
        expect(onClick).not.toHaveBeenCalled();
    });
});
