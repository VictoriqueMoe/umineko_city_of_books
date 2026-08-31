import { describe, expect, it } from "vitest";
import { replyPreview } from "./replyPreview";

interface PreviewCase {
    name: string;
    body: string;
    want: string;
}

describe("replyPreview", () => {
    const cases: PreviewCase[] = [
        { name: "keeps a short body whole", body: "a short claim", want: "a short claim" },
        { name: "keeps an empty body empty", body: "", want: "" },
        { name: "keeps a body of exactly the limit whole", body: "x".repeat(80), want: "x".repeat(80) },
        { name: "truncates the first body over the limit", body: "x".repeat(81), want: `${"x".repeat(80)}...` },
        { name: "truncates a long body and marks the cut", body: "x".repeat(120), want: `${"x".repeat(80)}...` },
    ];

    for (const testCase of cases) {
        it(testCase.name, () => {
            // given
            const body = testCase.body;

            // when
            const preview = replyPreview(body);

            // then
            expect(preview).toBe(testCase.want);
        });
    }

    it("never grows the body it was given beyond the ellipsis", () => {
        // given
        const body = "x".repeat(500);

        // when
        const preview = replyPreview(body);

        // then
        expect(preview).toHaveLength(83);
    });
});
