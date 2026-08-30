import { describe, expect, it } from "vitest";
import { isLightTheme } from "./themes";

describe("isLightTheme", () => {
    const cases: { name: string; theme: string | undefined | null; want: boolean }[] = [
        { name: "virgilia is the light theme", theme: "virgilia", want: true },
        { name: "featherine is dark", theme: "featherine", want: false },
        { name: "beatrice is dark", theme: "beatrice", want: false },
        { name: "an unknown theme is treated as dark", theme: "nonsense", want: false },
        { name: "an empty theme is treated as dark", theme: "", want: false },
        { name: "an absent theme is treated as dark", theme: undefined, want: false },
        { name: "a null theme is treated as dark", theme: null, want: false },
    ];

    for (const tc of cases) {
        it(tc.name, () => {
            // given the theme from the table

            // when
            const got = isLightTheme(tc.theme);

            // then
            expect(got).toBe(tc.want);
        });
    }
});
