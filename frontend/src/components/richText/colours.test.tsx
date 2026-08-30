import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { COLOUR_CLASS, colourRegex, renderColours } from "./colours";

function makeRenderer() {
    return vi.fn((text: string, _keyPrefix: string): ReactNode[] => [text]);
}

describe("COLOUR_CLASS", () => {
    it("maps every supported colour tag to its truth class", () => {
        // given / when / then
        expect(COLOUR_CLASS).toEqual({
            red: "red-truth",
            blue: "blue-truth",
            gold: "gold-truth",
            purple: "purple-truth",
            green: "green-truth",
            pink: "pink-truth",
        });
    });
});

describe("colourRegex", () => {
    it("captures the colour name and the wrapped text", () => {
        // given
        const text = "[gold]a golden claim[/gold]";

        // when
        const match = colourRegex().exec(text);

        // then
        expect(match?.[1]).toBe("gold");
        expect(match?.[2]).toBe("a golden claim");
    });

    it("hands back a fresh global regex so callers never share a cursor", () => {
        // given
        const first = colourRegex();
        first.exec("[red]a[/red]");

        // when
        const second = colourRegex();

        // then
        expect(first.lastIndex).toBeGreaterThan(0);
        expect(second.lastIndex).toBe(0);
        expect(second.global).toBe(true);
    });

    it("refuses to match when the closing tag is a different colour", () => {
        // given
        const text = "[red]mismatched[/blue]";

        // when
        const match = colourRegex().exec(text);

        // then
        expect(match).toBeNull();
    });

    it("matches across newlines inside a tag", () => {
        // given
        const text = "[blue]line one\nline two[/blue]";

        // when
        const match = colourRegex().exec(text);

        // then
        expect(match?.[2]).toBe("line one\nline two");
    });

    it("stops at the first closing tag rather than the last", () => {
        // given
        const text = "[red]one[/red] and [red]two[/red]";

        // when
        const match = colourRegex().exec(text);

        // then
        expect(match?.[2]).toBe("one");
    });
});

describe("renderColours", () => {
    it("returns nothing for empty text and never calls the inner renderer", () => {
        // given
        const renderInner = makeRenderer();

        // when
        const nodes = renderColours("", renderInner, "k");

        // then
        expect(nodes).toEqual([]);
        expect(renderInner).not.toHaveBeenCalled();
    });

    it("passes untagged text straight through to the inner renderer", () => {
        // given
        const renderInner = makeRenderer();

        // when
        const nodes = renderColours("nothing to colour here", renderInner, "k");

        // then
        expect(renderInner).toHaveBeenCalledTimes(1);
        expect(renderInner.mock.calls[0][0]).toBe("nothing to colour here");
        expect(nodes).toEqual(["nothing to colour here"]);
    });

    it("wraps a tagged run in a span carrying the colour class", () => {
        // given
        const renderInner = makeRenderer();

        // when
        render(<>{renderColours("[red]the truth[/red]", renderInner, "k")}</>);

        // then
        expect(screen.getByText("the truth")).toHaveClass("red-truth");
    });

    it("keeps the text before, inside and after a tag in order", () => {
        // given
        const renderInner = makeRenderer();

        // when
        const { container } = render(<>{renderColours("before [blue]middle[/blue] after", renderInner, "k")}</>);

        // then
        expect(container.textContent).toBe("before middle after");
        expect(renderInner.mock.calls.map(call => call[0])).toEqual(["before ", "middle", " after"]);
    });

    it("colours each tagged run separately when several colours appear", () => {
        // given
        const renderInner = makeRenderer();

        // when
        render(<>{renderColours("[red]one[/red][green]two[/green][purple]three[/purple]", renderInner, "k")}</>);

        // then
        expect(screen.getByText("one")).toHaveClass("red-truth");
        expect(screen.getByText("two")).toHaveClass("green-truth");
        expect(screen.getByText("three")).toHaveClass("purple-truth");
    });

    it("leaves a mismatched tag pair as ordinary text", () => {
        // given
        const renderInner = makeRenderer();

        // when
        const { container } = render(<>{renderColours("[red]mismatched[/blue]", renderInner, "k")}</>);

        // then
        expect(container.querySelector("span")).toBeNull();
        expect(container.textContent).toBe("[red]mismatched[/blue]");
    });

    it("renders an empty tag pair as a span with nothing in it", () => {
        // given
        const renderInner = makeRenderer();

        // when
        const { container } = render(<>{renderColours("[pink][/pink]", renderInner, "k")}</>);

        // then
        expect(container.querySelector(".pink-truth")).not.toBeNull();
        expect(container.textContent).toBe("");
    });

    it("gives every segment a distinct key prefix", () => {
        // given
        const renderInner = makeRenderer();

        // when
        renderColours("a [red]b[/red] c [gold]d[/gold] e", renderInner, "k");

        // then
        const prefixes = renderInner.mock.calls.map(call => call[1]);
        expect(prefixes).toHaveLength(5);
        expect(new Set(prefixes).size).toBe(prefixes.length);
        for (const prefix of prefixes) {
            expect(prefix.startsWith("k-")).toBe(true);
        }
    });
});
