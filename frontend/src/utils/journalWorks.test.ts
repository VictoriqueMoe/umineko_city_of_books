import { describe, expect, it } from "vitest";
import type { JournalWork } from "../types/api";
import { JOURNAL_WORKS, workLabel } from "./journalWorks";

describe("JOURNAL_WORKS", () => {
    it("lists every selectable work with general first", () => {
        // given / when / then
        expect(JOURNAL_WORKS.map(work => work.id)).toEqual([
            "general",
            "umineko",
            "higurashi",
            "ciconia",
            "higanbana",
            "roseguns",
        ]);
    });

    it("spells out the display name for works whose id is abbreviated", () => {
        // given
        const roseGuns = JOURNAL_WORKS.find(work => work.id === "roseguns");

        // when
        const label = roseGuns?.label;

        // then
        expect(label).toBe("Rose Guns Days");
    });

    it("gives every entry a non empty label", () => {
        // given / when / then
        for (const work of JOURNAL_WORKS) {
            expect(work.label.length).toBeGreaterThan(0);
        }
    });
});

describe("workLabel", () => {
    it("returns the display label for each known work", () => {
        // given / when / then
        expect(workLabel("general")).toBe("General");
        expect(workLabel("umineko")).toBe("Umineko");
        expect(workLabel("higurashi")).toBe("Higurashi");
        expect(workLabel("ciconia")).toBe("Ciconia");
        expect(workLabel("higanbana")).toBe("Higanbana");
        expect(workLabel("roseguns")).toBe("Rose Guns Days");
    });

    it("falls back to general for a work it does not recognise", () => {
        // given
        const unknown = "kanon" as JournalWork;

        // when
        const label = workLabel(unknown);

        // then
        expect(label).toBe("General");
    });

    it("falls back to general for an empty work id", () => {
        // given
        const blank = "" as JournalWork;

        // when
        const label = workLabel(blank);

        // then
        expect(label).toBe("General");
    });

    it("is case sensitive about the work id", () => {
        // given
        const shouty = "UMINEKO" as JournalWork;

        // when
        const label = workLabel(shouty);

        // then
        expect(label).toBe("General");
    });
});
