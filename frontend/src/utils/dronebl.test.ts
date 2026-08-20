import { describe, expect, it } from "vitest";
import { DRONEBL_CLASSES, parseIgnoredClasses, serialiseIgnoredClasses, toggleIgnoredClass } from "./dronebl";

describe("parseIgnoredClasses", () => {
    it("reads a comma separated setting", () => {
        // given the value the backend stores
        const parsed = parseIgnoredClasses("8,9,17");

        // then
        expect([...parsed].sort((a, b) => a - b)).toEqual([8, 9, 17]);
    });

    it("tolerates spacing, because a human typed it before the checkboxes existed", () => {
        // given
        const parsed = parseIgnoredClasses(" 13 , 18 ");

        // then
        expect([...parsed].sort((a, b) => a - b)).toEqual([13, 18]);
    });

    it("drops junk and out of range values rather than widening the block", () => {
        // given 1 is the RFC test entry and never a real listing
        const parsed = parseIgnoredClasses("nonsense, 0, 1, 256, 9");

        // then
        expect([...parsed]).toEqual([9]);
    });

    it("treats an empty setting as ignoring nothing", () => {
        // given the default
        expect(parseIgnoredClasses("")).toEqual(new Set());
        expect(parseIgnoredClasses("   ")).toEqual(new Set());
    });
});

describe("toggleIgnoredClass", () => {
    it("adds a class without disturbing the others", () => {
        // when
        const next = toggleIgnoredClass("8,9", 17, true);

        // then
        expect(next).toBe("8,9,17");
    });

    it("removes a class", () => {
        // when
        const next = toggleIgnoredClass("8,9,17", 9, false);

        // then
        expect(next).toBe("8,17");
    });

    it("keeps the value sorted so the stored setting is stable", () => {
        // given a value written in any order
        const next = toggleIgnoredClass("17,8", 9, true);

        // then saving twice must not churn the setting
        expect(next).toBe("8,9,17");
    });

    it("preserves a class the checkbox list does not know about", () => {
        // given somebody typed a class this build has no checkbox for
        const next = toggleIgnoredClass("4,9", 17, true);

        // then it survives, rather than being silently dropped on save
        expect(next).toBe("4,9,17");
    });

    it("is a no-op when the class is already in the wanted state", () => {
        expect(toggleIgnoredClass("8,9", 9, true)).toBe("8,9");
        expect(toggleIgnoredClass("8,9", 17, false)).toBe("8,9");
    });

    it("goes back to empty when the last class is unticked", () => {
        // then the backend reads empty as blocking every listing
        expect(toggleIgnoredClass("17", 17, false)).toBe("");
    });
});

describe("DRONEBL_CLASSES", () => {
    it("covers every class DroneBL documents", () => {
        // given the published list, which omits 1 and 4
        const ids = DRONEBL_CLASSES.map(cls => cls.id);

        expect(ids).toEqual([2, 3, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 255]);
    });

    it("round trips through the stored format", () => {
        // given every class ticked
        const all = new Set(DRONEBL_CLASSES.map(cls => cls.id));

        // when
        const stored = serialiseIgnoredClasses(all);

        // then nothing is lost on the way back
        expect(parseIgnoredClasses(stored)).toEqual(all);
    });

    it("labels each class in English rather than by number", () => {
        for (const cls of DRONEBL_CLASSES) {
            expect(cls.label).not.toMatch(/^\d+$/);
            expect(cls.label.length).toBeGreaterThan(3);
        }
    });
});
