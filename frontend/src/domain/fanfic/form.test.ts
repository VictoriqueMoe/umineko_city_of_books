import { describe, expect, it } from "vitest";
import type { FanficDetail, User } from "../../types/api";
import {
    collapseGenres,
    fieldDefaults,
    mergeDefined,
    OTHER_VALUE,
    resolveLanguage,
    resolveSeries,
    seedFromFanfic,
    seriesOptions,
    toPayload,
    validateDetails,
    withCharacter,
    withoutCharacter,
    withoutTag,
    withTag,
    type EditorFields,
} from "./form";

const author: User = {
    id: "author-1",
    username: "beatrice",
    display_name: "Beatrice",
    avatar_url: "",
};

function makeFanfic(overrides: Partial<FanficDetail> = {}): FanficDetail {
    return {
        id: "fanfic-1",
        author,
        title: "Golden Land",
        summary: "A closed room on Rokkenjima.",
        series: "Umineko",
        rating: "T",
        language: "English",
        status: "in_progress",
        is_oneshot: false,
        contains_lemons: false,
        genres: ["Mystery"],
        tags: ["closed room"],
        characters: [],
        is_pairing: false,
        word_count: 2500,
        chapter_count: 2,
        favourite_count: 4,
        view_count: 90,
        comment_count: 0,
        user_favourited: false,
        published_at: "2026-01-01T00:00:00Z",
        created_at: "2026-01-01T00:00:00Z",
        chapters: [],
        comments: [],
        reading_progress: 0,
        viewer_blocked: false,
        ...overrides,
    };
}

function makeFields(overrides: Partial<EditorFields> = {}): EditorFields {
    return { ...fieldDefaults(), ...overrides };
}

describe("fieldDefaults", () => {
    it("starts a fanfic as a one-shot in Umineko in English", () => {
        // when
        const fields = fieldDefaults();

        // then
        expect(fields.series).toBe("Umineko");
        expect(fields.language).toBe("English");
        expect(fields.rating).toBe("K");
        expect(fields.status).toBe("in_progress");
        expect(fields.isOneshot).toBe(true);
    });

    it("hands out a fresh object every time", () => {
        // when
        const first = fieldDefaults();
        const second = fieldDefaults();

        // then
        first.tags.push("closed room");
        expect(second.tags).toEqual([]);
    });
});

describe("mergeDefined", () => {
    it("lets a later override win over an earlier one", () => {
        // when
        const merged = mergeDefined(fieldDefaults(), { title: "Seed" }, { title: "Typed" });

        // then
        expect(merged.title).toBe("Typed");
    });

    it("ignores an undefined value rather than blanking the field", () => {
        // when
        const merged = mergeDefined(fieldDefaults(), { title: "Seed" }, { title: undefined });

        // then
        expect(merged.title).toBe("Seed");
    });

    it("ignores a missing override entirely", () => {
        // when
        const merged = mergeDefined(fieldDefaults(), null, undefined);

        // then
        expect(merged).toEqual(fieldDefaults());
    });

    it("keeps a deliberately emptied field empty", () => {
        // when
        const merged = mergeDefined(fieldDefaults(), { title: "Seed" }, { title: "" });

        // then
        expect(merged.title).toBe("");
    });

    it("ignores a key the base does not have", () => {
        // when
        const merged = mergeDefined(fieldDefaults(), { chapterCount: 3 } as unknown as Partial<EditorFields>);

        // then
        expect(merged).toEqual(fieldDefaults());
    });
});

describe("seedFromFanfic", () => {
    it("has nothing to seed without a fanfic", () => {
        // then
        expect(seedFromFanfic(null, ["English"])).toBeNull();
    });

    it("copies the fanfic as it stands into the form", () => {
        // when
        const seed = seedFromFanfic(makeFanfic(), ["English"]);

        // then
        expect(seed).toMatchObject({
            title: "Golden Land",
            summary: "A closed room on Rokkenjima.",
            rating: "T",
            isOneshot: false,
            containsLemons: false,
            isPairing: false,
            status: "in_progress",
            genreA: "Mystery",
            genreB: "",
            tags: ["closed room"],
            series: "Umineko",
            customSeries: "",
            language: "English",
            customLanguage: "",
            coverPreview: "",
            chapterCount: 2,
        });
    });

    it("moves a series the archive does not pin into the custom field", () => {
        // when
        const seed = seedFromFanfic(makeFanfic({ series: "Higanbana" }), ["English"]);

        // then
        expect(seed?.series).toBe(OTHER_VALUE);
        expect(seed?.customSeries).toBe("Higanbana");
    });

    it("moves a language the archive does not know into the custom field", () => {
        // when
        const seed = seedFromFanfic(makeFanfic({ language: "Welsh" }), ["English", "Japanese"]);

        // then
        expect(seed?.language).toBe(OTHER_VALUE);
        expect(seed?.customLanguage).toBe("Welsh");
    });

    it("leaves the language alone while the archive has not said which it knows", () => {
        // when
        const seed = seedFromFanfic(makeFanfic({ language: "Welsh" }), []);

        // then
        expect(seed?.language).toBe("Welsh");
        expect(seed?.customLanguage).toBe("");
    });

    it("renumbers the characters from the top", () => {
        // given
        const fanfic = makeFanfic({
            characters: [
                { series: "umineko", character_id: "c2", character_name: "Kanon", sort_order: 7 },
                { series: "umineko", character_id: "c1", character_name: "Shannon", sort_order: 4 },
            ],
        });

        // when
        const seed = seedFromFanfic(fanfic, ["English"]);

        // then
        expect(seed?.characters).toEqual([
            { series: "umineko", character_id: "c2", character_name: "Kanon", sort_order: 0 },
            { series: "umineko", character_id: "c1", character_name: "Shannon", sort_order: 1 },
        ]);
    });

    it("takes both genres when the fanfic has two", () => {
        // when
        const seed = seedFromFanfic(makeFanfic({ genres: ["Mystery", "Horror"] }), ["English"]);

        // then
        expect(seed?.genreA).toBe("Mystery");
        expect(seed?.genreB).toBe("Horror");
    });

    it("seeds the cover preview from the cover the fanfic already has", () => {
        // when
        const seed = seedFromFanfic(makeFanfic({ cover_image_url: "/covers/1.png" }), ["English"]);

        // then
        expect(seed?.coverPreview).toBe("/covers/1.png");
    });
});

describe("seriesOptions", () => {
    it("pins the three works ahead of whatever the archive holds", () => {
        // when
        const options = seriesOptions(["Rose Guns Days"]);

        // then
        expect(options).toEqual(["Umineko", "Higurashi", "Ciconia", "Rose Guns Days"]);
    });

    it("never offers a pinned work twice", () => {
        // when
        const options = seriesOptions(["Umineko", "Higurashi", "Rose Guns Days"]);

        // then
        expect(options).toEqual(["Umineko", "Higurashi", "Ciconia", "Rose Guns Days"]);
    });
});

describe("resolveSeries", () => {
    it("takes the chosen series when it is not the custom one", () => {
        // then
        expect(resolveSeries({ series: "Higurashi", customSeries: "ignored" })).toBe("Higurashi");
    });

    it("trims the typed series when the writer chose Other", () => {
        // then
        expect(resolveSeries({ series: OTHER_VALUE, customSeries: "  Higanbana  " })).toBe("Higanbana");
    });
});

describe("resolveLanguage", () => {
    it("takes the chosen language when it is not the custom one", () => {
        // then
        expect(resolveLanguage({ language: "Japanese", customLanguage: "ignored" })).toBe("Japanese");
    });

    it("trims the typed language when the writer chose Other", () => {
        // then
        expect(resolveLanguage({ language: OTHER_VALUE, customLanguage: "  Welsh  " })).toBe("Welsh");
    });
});

describe("collapseGenres", () => {
    it("keeps nothing when neither genre was chosen", () => {
        // then
        expect(collapseGenres("", "")).toEqual([]);
    });

    it("keeps the first genre alone", () => {
        // then
        expect(collapseGenres("Mystery", "")).toEqual(["Mystery"]);
    });

    it("keeps both genres in the order they were chosen", () => {
        // then
        expect(collapseGenres("Mystery", "Horror")).toEqual(["Mystery", "Horror"]);
    });

    it("never repeats the same genre twice", () => {
        // then
        expect(collapseGenres("Mystery", "Mystery")).toEqual(["Mystery"]);
    });

    it("keeps a second genre chosen without a first", () => {
        // then
        expect(collapseGenres("", "Horror")).toEqual(["Horror"]);
    });
});

describe("validateDetails", () => {
    it("passes a fanfic with a title, a series and a language", () => {
        // then
        expect(validateDetails(makeFields({ title: "Golden Land" }))).toBe("");
    });

    it("asks for a title before anything else", () => {
        // then
        expect(validateDetails(makeFields({ title: "   " }))).toBe("Title is required");
    });

    it("names the series when the custom one was left empty", () => {
        // then
        expect(validateDetails(makeFields({ title: "Golden Land", series: OTHER_VALUE, customSeries: "  " }))).toBe(
            "Series is required",
        );
    });

    it("names the language when the custom one was left empty", () => {
        // then
        expect(validateDetails(makeFields({ title: "Golden Land", language: OTHER_VALUE, customLanguage: "  " }))).toBe(
            "Language is required",
        );
    });

    it("names the series first when both custom fields were left empty", () => {
        // given
        const fields = makeFields({
            title: "Golden Land",
            series: OTHER_VALUE,
            customSeries: "",
            language: OTHER_VALUE,
            customLanguage: "",
        });

        // then
        expect(validateDetails(fields)).toBe("Series is required");
    });
});

describe("toPayload", () => {
    it("trims the title and the summary", () => {
        // when
        const payload = toPayload(makeFields({ title: "  Golden Land  ", summary: "  A closed room.  " }));

        // then
        expect(payload.title).toBe("Golden Land");
        expect(payload.summary).toBe("A closed room.");
    });

    it("sends the resolved custom series and language rather than the placeholder", () => {
        // given
        const fields = makeFields({
            title: "Golden Land",
            series: OTHER_VALUE,
            customSeries: "  Higanbana  ",
            language: OTHER_VALUE,
            customLanguage: "  Welsh  ",
        });

        // when
        const payload = toPayload(fields);

        // then
        expect(payload.series).toBe("Higanbana");
        expect(payload.language).toBe("Welsh");
    });

    it("collapses the two genre fields into the genre list", () => {
        // when
        const payload = toPayload(makeFields({ genreA: "Mystery", genreB: "Mystery" }));

        // then
        expect(payload.genres).toEqual(["Mystery"]);
    });

    it("sends the whole form under the names the archive expects", () => {
        // given
        const fields = makeFields({
            title: "Golden Land",
            summary: "A closed room.",
            rating: "M",
            status: "complete",
            isOneshot: false,
            containsLemons: true,
            isPairing: true,
            tags: ["closed room"],
            characters: [{ series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 }],
        });

        // when
        const payload = toPayload(fields);

        // then
        expect(payload).toEqual({
            title: "Golden Land",
            summary: "A closed room.",
            series: "Umineko",
            rating: "M",
            language: "English",
            status: "complete",
            is_oneshot: false,
            contains_lemons: true,
            genres: [],
            tags: ["closed room"],
            characters: [{ series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 }],
            is_pairing: true,
        });
    });

    it("never carries the cover preview to the archive", () => {
        // when
        const payload = toPayload(makeFields({ coverPreview: "blob:local" }));

        // then
        expect(payload).not.toHaveProperty("coverPreview");
    });
});

describe("withCharacter", () => {
    it("adds a character at the end and numbers it", () => {
        // given
        const existing = [{ series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 }];

        // when
        const next = withCharacter(existing, {
            series: "umineko",
            character_id: "c2",
            character_name: "Shannon",
            sort_order: 0,
        });

        // then
        expect(next).toEqual([
            { series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 },
            { series: "umineko", character_id: "c2", character_name: "Shannon", sort_order: 1 },
        ]);
    });
});

describe("withoutCharacter", () => {
    it("drops the chosen character and renumbers the rest", () => {
        // given
        const existing = [
            { series: "umineko", character_id: "c1", character_name: "Kanon", sort_order: 0 },
            { series: "umineko", character_id: "c2", character_name: "Shannon", sort_order: 1 },
            { series: "umineko", character_id: "c3", character_name: "Jessica", sort_order: 2 },
        ];

        // when
        const next = withoutCharacter(existing, 0);

        // then
        expect(next).toEqual([
            { series: "umineko", character_id: "c2", character_name: "Shannon", sort_order: 0 },
            { series: "umineko", character_id: "c3", character_name: "Jessica", sort_order: 1 },
        ]);
    });
});

describe("withTag", () => {
    it("adds a trimmed tag", () => {
        // then
        expect(withTag([], "  closed room  ")).toEqual(["closed room"]);
    });

    it("ignores an empty tag", () => {
        // then
        expect(withTag(["closed room"], "   ")).toEqual(["closed room"]);
    });

    it("refuses the same tag twice whatever the casing", () => {
        // then
        expect(withTag(["closed room"], "Closed Room")).toEqual(["closed room"]);
    });

    it("stops at ten tags", () => {
        // given
        const tags = ["a", "b", "c", "d", "e", "f", "g", "h", "i", "j"];

        // then
        expect(withTag(tags, "k")).toEqual(tags);
    });

    it("cuts a tag off at thirty characters", () => {
        // then
        expect(withTag([], "x".repeat(40))).toEqual(["x".repeat(30)]);
    });
});

describe("withoutTag", () => {
    it("drops the tag at the given position", () => {
        // then
        expect(withoutTag(["closed room", "locked door"], 0)).toEqual(["locked door"]);
    });
});
