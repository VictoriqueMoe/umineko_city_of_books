import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { MentionLink } from "./MentionLink";
import { MentionResolverContext, type MentionResolverContextValue } from "../../context/mentionResolverContextValue";

function renderMention(known: boolean | undefined, withProvider = true): string {
    const resolver: MentionResolverContextValue = {
        isKnown: () => known,
        request: () => {},
    };
    const mention = <MentionLink username="Featherines_other_half" label="@Featherines_other_half" />;
    return renderToStaticMarkup(
        <MemoryRouter>
            {withProvider ? (
                <MentionResolverContext.Provider value={resolver}>{mention}</MentionResolverContext.Provider>
            ) : (
                mention
            )}
        </MemoryRouter>,
    );
}

describe("MentionLink", () => {
    it("links a mention once the user is known to exist", () => {
        // given
        const known = true;

        // when
        const html = renderMention(known);

        // then
        expect(html).toContain('href="/user/Featherines_other_half"');
        expect(html).toContain("@Featherines_other_half");
    });

    it("renders plain text for a user that does not exist", () => {
        // given
        const known = false;

        // when
        const html = renderMention(known);

        // then
        expect(html).not.toContain("<a");
        expect(html).toBe("@Featherines_other_half");
    });

    it("renders plain text while resolution is still pending", () => {
        // given
        const known = undefined;

        // when
        const html = renderMention(known);

        // then
        expect(html).not.toContain("<a");
        expect(html).toBe("@Featherines_other_half");
    });

    it("renders plain text when no resolver is mounted", () => {
        // given
        const withProvider = false;

        // when
        const html = renderMention(true, withProvider);

        // then
        expect(html).not.toContain("<a");
        expect(html).toBe("@Featherines_other_half");
    });
});
