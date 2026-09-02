import { describe, expect, it } from "vitest";
import { charCount, clampChars, ellipsise } from "./text";

describe("charCount", () => {
    it("counts an emoji as one character rather than two code units", () => {
        // given
        const value = "🎲🎲";

        // when
        const count = charCount(value);

        // then
        expect(value.length).toBe(4);
        expect(count).toBe(2);
    });
});

describe("clampChars", () => {
    it("returns the value untouched when it is short enough", () => {
        // given
        const value = "Beatrice";

        // when
        const clipped = clampChars(value, 20);

        // then
        expect(clipped).toBe("Beatrice");
    });

    it("never splits an emoji in half", () => {
        // given
        const value = "ab🎲cd";

        // when
        const clipped = clampChars(value, 3);

        // then
        expect(clipped).toBe("ab🎲");
        expect(clipped).not.toContain("�");
    });

    it("counts Japanese characters one at a time", () => {
        // given
        const value = "雛見沢村";

        // when
        const clipped = clampChars(value, 2);

        // then
        expect(clipped).toBe("雛見");
    });
});

describe("ellipsise", () => {
    it("appends an ellipsis only when it actually truncated", () => {
        // given
        const short = "hi";
        const long = "ab🎲cd";

        // when
        const keptShort = ellipsise(short, 10);
        const cutLong = ellipsise(long, 3);

        // then
        expect(keptShort).toBe("hi");
        expect(cutLong).toBe("ab🎲...");
    });
});
