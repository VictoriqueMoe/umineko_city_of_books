import { beforeEach, describe, expect, it, vi } from "vitest";
import {
    clearDraft,
    draftFromFields,
    draftToFields,
    DRAFT_KEY,
    loadDraft,
    resumableDraft,
    saveDraft,
    type DraftData,
} from "./draft";
import { fieldDefaults, type EditorFields } from "./form";

function makeDraft(overrides: Partial<DraftData> = {}): DraftData {
    return {
        title: "Golden Land",
        summary: "A closed room.",
        series: "Umineko",
        customSeries: "",
        rating: "M",
        language: "English",
        customLanguage: "",
        genreA: "Mystery",
        genreB: "",
        tags: ["closed room"],
        status: "in_progress",
        characters: [],
        isPairing: false,
        isOneshot: true,
        containsLemons: false,
        body: "Beatrice laughed.",
        step: 1,
        ...overrides,
    };
}

function makeFields(overrides: Partial<EditorFields> = {}): EditorFields {
    return { ...fieldDefaults(), ...overrides };
}

beforeEach(() => {
    localStorage.clear();
});

describe("loadDraft", () => {
    it("reads back what was stored", () => {
        // given
        saveDraft(makeDraft());

        // then
        expect(loadDraft()).toEqual(makeDraft());
    });

    it("has nothing to read when no draft was ever stored", () => {
        // then
        expect(loadDraft()).toBeNull();
    });

    it("survives a draft that is not valid json", () => {
        // given
        localStorage.setItem(DRAFT_KEY, "{not json");

        // then
        expect(loadDraft()).toBeNull();
    });

    it("survives storage that refuses to be read", () => {
        // given
        const getItem = vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
            throw new Error("denied");
        });

        // then
        expect(loadDraft()).toBeNull();
        getItem.mockRestore();
    });
});

describe("clearDraft", () => {
    it("throws the stored draft away", () => {
        // given
        saveDraft(makeDraft());

        // when
        clearDraft();

        // then
        expect(localStorage.getItem(DRAFT_KEY)).toBeNull();
    });
});

describe("resumableDraft", () => {
    it("offers a draft that got as far as a title", () => {
        // given
        saveDraft(makeDraft({ title: "Golden Land" }));

        // then
        expect(resumableDraft()?.title).toBe("Golden Land");
    });

    it("ignores a draft that never got a title", () => {
        // given
        saveDraft(makeDraft({ title: "", body: "something" }));

        // then
        expect(resumableDraft()).toBeNull();
    });

    it("has nothing to offer when no draft was stored", () => {
        // then
        expect(resumableDraft()).toBeNull();
    });
});

describe("draftToFields", () => {
    it("puts the stored draft back into the form", () => {
        // when
        const fields = draftToFields(makeDraft());

        // then
        expect(fields).toEqual({
            title: "Golden Land",
            summary: "A closed room.",
            series: "Umineko",
            customSeries: "",
            rating: "M",
            language: "English",
            customLanguage: "",
            genreA: "Mystery",
            genreB: "",
            tags: ["closed room"],
            status: "in_progress",
            characters: [],
            isPairing: false,
            isOneshot: true,
            containsLemons: false,
        });
    });

    it("never restores the cover preview, which was never stored", () => {
        // when
        const fields = draftToFields(makeDraft());

        // then
        expect(fields).not.toHaveProperty("coverPreview");
    });

    it("falls back to an empty tag list and an in progress status on an older draft", () => {
        // given
        const older = { title: "Golden Land", body: "", step: 1 } as unknown as DraftData;

        // when
        const fields = draftToFields(older);

        // then
        expect(fields.tags).toEqual([]);
        expect(fields.status).toBe("in_progress");
    });
});

describe("draftFromFields", () => {
    it("stores the form, the prose and the step the writer reached", () => {
        // when
        const draft = draftFromFields(makeFields({ title: "Golden Land" }), "Beatrice laughed.", 2);

        // then
        expect(draft.title).toBe("Golden Land");
        expect(draft.body).toBe("Beatrice laughed.");
        expect(draft.step).toBe(2);
    });

    it("never stores the cover preview, which is a local blob url", () => {
        // when
        const draft = draftFromFields(makeFields({ coverPreview: "blob:local" }), "", 1);

        // then
        expect(draft).not.toHaveProperty("coverPreview");
    });

    it("round trips through storage unchanged", () => {
        // given
        const draft = draftFromFields(makeFields({ title: "Golden Land", tags: ["closed room"] }), "prose", 2);

        // when
        saveDraft(draft);

        // then
        expect(loadDraft()).toEqual(draft);
    });
});
