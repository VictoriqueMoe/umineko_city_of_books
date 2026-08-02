import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { SearchResult } from "../../../types/api";
import { renderWithProviders } from "../../../test-utils/render";
import { SearchResultRow } from "./SearchResultRow";

function makeResult(overrides: Partial<SearchResult> = {}): SearchResult {
    return {
        type: "theory",
        id: "theory-1",
        parent_id: null,
        parent_title: null,
        title: "The golden truth",
        snippet: "",
        url: "/theory/theory-1",
        author: {
            id: "user-1",
            username: "beatrice",
            display_name: "Beatrice",
            avatar_url: "",
        },
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

describe("SearchResultRow", () => {
    it("links to the destination the search gave for the result", () => {
        // given
        const result = makeResult({ url: "/theory/theory-1" });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByRole("link")).toHaveAttribute("href", "/theory/theory-1");
    });

    it("still renders a row when the result has no destination", () => {
        // given
        const result = makeResult({ url: "" });

        // when
        renderWithProviders(<SearchResultRow result={result} />, { route: "/search" });

        // then
        expect(screen.getByText("The golden truth")).toBeInTheDocument();
    });

    it("offers nothing to click when the result has no destination", () => {
        // given
        const result = makeResult({ url: "" });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />, { route: "/search" });

        // then
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
        expect(container.querySelector("a")).toBeNull();
    });

    it("never sends the visitor back to the page they are already on", () => {
        // given
        const result = makeResult({ url: "" });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />, { route: "/theories" });

        // then
        expect(container.querySelector("[href]")).toBeNull();
    });

    it("announces itself as an option of the results list when the dropdown asks it to", () => {
        // given
        const result = makeResult();

        // when
        renderWithProviders(<SearchResultRow result={result} optionId="row-0" active />);

        // then
        const option = screen.getByRole("option");
        expect(option).toHaveAttribute("id", "row-0");
        expect(option).toHaveAttribute("aria-selected", "true");
    });

    it("says it is not the chosen option while another row is highlighted", () => {
        // given
        const result = makeResult();

        // when
        renderWithProviders(<SearchResultRow result={result} optionId="row-0" />);

        // then
        expect(screen.getByRole("option")).toHaveAttribute("aria-selected", "false");
    });

    it("stays an ordinary link on the full search page", () => {
        // given
        const result = makeResult();

        // when
        renderWithProviders(<SearchResultRow result={result} variant="page" />);

        // then
        expect(screen.getByRole("link")).toHaveAttribute("href", "/theory/theory-1");
        expect(screen.queryByRole("option")).not.toBeInTheDocument();
    });

    it("badges the row with the short name of its type", () => {
        // given
        const result = makeResult({ type: "mystery_attempt" });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("Solution")).toBeInTheDocument();
    });

    it("paints the badge with the colour of the type family", () => {
        // given
        const result = makeResult({ type: "fanfic_comment" });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("Comment")).toHaveStyle({ background: "#34d399" });
    });

    it("falls back to a placeholder title for an untitled result", () => {
        // given
        const result = makeResult({ title: "" });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("(untitled)")).toBeInTheDocument();
    });

    it("shows the author avatar when the author has one", () => {
        // given
        const result = makeResult({
            author: { id: "user-1", username: "beatrice", display_name: "Beatrice", avatar_url: "/avatars/bea.png" },
        });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(container.querySelector("img")).toHaveAttribute("src", "/avatars/bea.png");
        expect(screen.queryByText("B")).not.toBeInTheDocument();
    });

    it("falls back to the first letter of the display name when there is no avatar", () => {
        // given
        const result = makeResult({
            author: { id: "user-1", username: "beatrice", display_name: "ronove", avatar_url: "" },
        });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("R")).toBeInTheDocument();
        expect(container.querySelector("img")).toBeNull();
    });

    it("falls back to the username when the author has no display name", () => {
        // given
        const result = makeResult({
            author: { id: "user-1", username: "kanon", display_name: "", avatar_url: "" },
        });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("K")).toBeInTheDocument();
    });

    it("falls back to a question mark when the author has no name at all", () => {
        // given
        const result = makeResult({
            author: { id: null, username: "", display_name: "", avatar_url: "" },
        });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("?")).toBeInTheDocument();
    });

    it("credits the author of a content result", () => {
        // given
        const result = makeResult({ type: "post" });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText(/^by/)).toBeInTheDocument();
    });

    it("credits the author by username when they have no display name", () => {
        // given
        const result = makeResult({
            author: { id: "user-1", username: "kanon", display_name: "", avatar_url: "" },
        });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.getByText("kanon")).toBeInTheDocument();
    });

    it("leaves off the author line for a user result", () => {
        // given
        const result = makeResult({ type: "user", title: "Beatrice" });

        // when
        renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(screen.queryByText(/^by/)).not.toBeInTheDocument();
    });

    it("keeps the highlight markup of a snippet", () => {
        // given
        const result = makeResult({ snippet: "the <mark>golden</mark> truth" });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(container.querySelector("mark")).toHaveTextContent("golden");
    });

    it("strips scripts out of a snippet", () => {
        // given
        const result = makeResult({ snippet: '<mark>truth</mark><script>alert("witch")</script>' });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(container.querySelector("script")).toBeNull();
        expect(container.querySelector("mark")).toHaveTextContent("truth");
    });

    it("strips event handlers and images out of a snippet", () => {
        // given
        const result = makeResult({ snippet: '<img src="x" onerror="alert(1)"><b onclick="alert(2)">red</b>' });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(container.querySelector("img")).toBeNull();
        expect(container.querySelector("b")).toBeNull();
        expect(screen.getByRole("link")).toHaveTextContent("red");
    });

    it("shows no snippet when the result has nothing to quote", () => {
        // given
        const result = makeResult({ snippet: "" });

        // when
        const { container } = renderWithProviders(<SearchResultRow result={result} />);

        // then
        expect(container.querySelector("mark")).toBeNull();
        expect(screen.getByRole("link")).toHaveTextContent("The golden truth");
    });

    it("tells the parent when the row is chosen", async () => {
        // given
        const onSelect = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<SearchResultRow result={makeResult()} onSelect={onSelect} />);

        // when
        await user.click(screen.getByRole("link"));

        // then
        expect(onSelect).toHaveBeenCalledOnce();
    });

    it("survives being chosen when the parent wants no callback", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<SearchResultRow result={makeResult()} />);

        // when
        await user.click(screen.getByRole("link"));

        // then
        expect(screen.getByText("The golden truth")).toBeInTheDocument();
    });
});
