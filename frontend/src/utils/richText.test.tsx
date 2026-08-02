import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders, type ProviderOptions } from "../test-utils/render";
import { renderRich } from "./richText";

function renderRichText(text: string, options: ProviderOptions = {}) {
    return renderWithProviders(<div>{renderRich(text)}</div>, options);
}

describe("renderRich", () => {
    it("renders nothing for empty text", () => {
        // given
        const text = "";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.textContent).toBe("");
    });

    it("renders plain text as it is", () => {
        // given
        const text = "the golden witch laughed";

        // when
        renderRichText(text);

        // then
        expect(screen.getByText("the golden witch laughed")).toBeInTheDocument();
    });

    it("renders double asterisks as bold", () => {
        // given
        const text = "**without love it cannot be seen**";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("strong")?.textContent).toBe("without love it cannot be seen");
    });

    it("renders single asterisks and single underscores as italics", () => {
        // given
        const text = "*one* and _two_";

        // when
        const { container } = renderRichText(text);

        // then
        const italics = container.querySelectorAll("em");
        expect(italics).toHaveLength(2);
        expect(italics[0].textContent).toBe("one");
        expect(italics[1].textContent).toBe("two");
    });

    it("renders double underscores as underline and double tildes as strikethrough", () => {
        // given
        const text = "__underlined__ and ~~struck~~";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("u")?.textContent).toBe("underlined");
        expect(container.querySelector("s")?.textContent).toBe("struck");
    });

    it("renders triple asterisks as bold italics", () => {
        // given
        const text = "***emphatic***";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("strong em")?.textContent).toBe("emphatic");
    });

    it("nests marks inside one another", () => {
        // given
        const text = "**bold with *italic* inside**";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("strong")?.textContent).toBe("bold with italic inside");
        expect(container.querySelector("strong em")?.textContent).toBe("italic");
    });

    it("renders backticks as inline code and leaves markdown inside them alone", () => {
        // given
        const text = "`const x = 1;` and `**stars**`";

        // when
        const { container } = renderRichText(text);

        // then
        const codes = container.querySelectorAll("code");
        expect(codes).toHaveLength(2);
        expect(codes[0].textContent).toBe("const x = 1;");
        expect(codes[1].textContent).toBe("**stars**");
        expect(container.querySelector("strong")).toBeNull();
    });

    it("leaves an unclosed marker as literal text", () => {
        // given
        const text = "~~struck";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.textContent).toBe("~~struck");
        expect(container.querySelector("s")).toBeNull();
    });

    it("leaves an unclosed doubled marker as literal text instead of swallowing it", () => {
        // given
        const text = "**bold";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.textContent).toBe("**bold");
        expect(container.querySelector("strong")).toBeNull();
        expect(container.querySelector("em")).toBeNull();
    });

    it("leaves an unclosed double underscore as literal text", () => {
        // given
        const text = "__underlined";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.textContent).toBe("__underlined");
        expect(container.querySelector("u")).toBeNull();
        expect(container.querySelector("em")).toBeNull();
    });

    it("does not emit an empty mark for a marker with nothing between its halves", () => {
        // given
        const text = "****";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.textContent).toBe("****");
        expect(container.querySelector("strong")).toBeNull();
        expect(container.querySelector("em")).toBeNull();
    });

    it("hides text between double pipes behind a spoiler", () => {
        // given
        const text = "the culprit is ||Beatrice|| really";

        // when
        const { container } = renderRichText(text);

        // then
        expect(screen.getByTitle("Hover to reveal").textContent).toBe("Beatrice");
        expect(container.textContent).toBe("the culprit is Beatrice really");
    });

    it("leaves an unclosed spoiler as literal text", () => {
        // given
        const text = "||secret";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.textContent).toBe("||secret");
        expect(screen.queryByTitle("Hover to reveal")).not.toBeInTheDocument();
    });

    it("wraps colour tags in the matching truth colour", () => {
        // given
        const text = "[red]the truth[/red] and [blue]a guess[/blue]";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector(".red-truth")?.textContent).toBe("the truth");
        expect(container.querySelector(".blue-truth")?.textContent).toBe("a guess");
    });

    it("still applies inline marks inside a colour tag", () => {
        // given
        const text = "[gold]**absolute**[/gold]";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector(".gold-truth strong")?.textContent).toBe("absolute");
    });

    it("renders a fenced block as a code block, keeping the source text", () => {
        // given
        const text = "```js\nconst answer = 42;\n```";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("pre code")?.textContent).toBe("const answer = 42;");
    });

    it("closes an unterminated fence at the end of the text", () => {
        // given
        const text = "```\nnever closed";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("pre code")?.textContent).toBe("never closed");
    });

    it("does not parse markdown or html inside a fenced block", () => {
        // given
        const text = "```\n<b>**not bold**</b>\n```";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("pre code")?.textContent).toBe("<b>**not bold**</b>");
        expect(container.querySelector("pre b")).toBeNull();
        expect(container.querySelector("strong")).toBeNull();
    });

    it("escapes html written in plain text instead of rendering it", () => {
        // given
        const text = '<script>alert("xss")</script><img src=x onerror=boom>';

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("script")).toBeNull();
        expect(container.querySelector("img")).toBeNull();
        expect(container.textContent).toBe('<script>alert("xss")</script><img src=x onerror=boom>');
    });

    it("renders a leading angle bracket as a quote that ends at a blank line", () => {
        // given
        const text = "> first line\n> second line\n\nafter the quote";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("blockquote")?.textContent).toBe("first line\nsecond line");
        expect(container.textContent).toContain("after the quote");
    });

    it("links an external url so it opens in a new tab", () => {
        // given
        const text = "see https://example.test/page here";

        // when
        renderRichText(text);

        // then
        const link = screen.getByRole("link", { name: "https://example.test/page" });
        expect(link).toHaveAttribute("href", "https://example.test/page");
        expect(link).toHaveAttribute("target", "_blank");
        expect(link).toHaveAttribute("rel", "noopener noreferrer");
    });

    it("links an internal url through the router as a relative path", () => {
        // given
        const text = "https://whentheycry.social/theories/1?sort=new";

        // when
        renderRichText(text);

        // then
        const link = screen.getByRole("link", { name: "https://whentheycry.social/theories/1?sort=new" });
        expect(link).toHaveAttribute("href", "/theories/1?sort=new");
        expect(link).not.toHaveAttribute("target");
    });

    it("does not treat underscores inside a url as italics", () => {
        // given
        const text = "https://example.test/a_b_c";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("em")).toBeNull();
        expect(screen.getByRole("link", { name: "https://example.test/a_b_c" })).toHaveAttribute(
            "href",
            "https://example.test/a_b_c",
        );
    });

    it("renders a mention as plain text until the resolver confirms the user", () => {
        // given
        const request = vi.fn();

        // when
        const { container } = renderRichText("hello @beatrice", {
            mentionResolver: { isKnown: () => undefined, request },
        });

        // then
        expect(container.textContent).toBe("hello @beatrice");
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
        expect(request).toHaveBeenCalledWith("beatrice");
    });

    it("links a mention once the resolver knows the user", () => {
        // given
        const resolver = { isKnown: () => true, request: vi.fn() };

        // when
        const { container } = renderRichText("hello @beatrice, are you there", { mentionResolver: resolver });

        // then
        expect(screen.getByRole("link", { name: "@beatrice" })).toHaveAttribute("href", "/user/beatrice");
        expect(container.textContent).toBe("hello @beatrice, are you there");
    });

    it("does not treat underscores inside a mention as italics", () => {
        // given
        const text = "@ao_no_kanata said so";

        // when
        const { container } = renderRichText(text);

        // then
        expect(container.querySelector("em")).toBeNull();
        expect(container.textContent).toBe("@ao_no_kanata said so");
    });
});
