import { describe, expect, it } from "vitest";
import type { SearchEntityType } from "../../../types/api";
import {
    SEARCH_FILTER_OPTIONS,
    SEARCH_GROUP_LABEL,
    SEARCH_GROUP_ORDER,
    SEARCH_TYPE_META,
    type SearchTypeGroup,
} from "./searchTypeMeta";

interface MetaCase {
    type: SearchEntityType;
    label: string;
    short: string;
    colour: string;
    group: SearchTypeGroup;
}

const metaCases: MetaCase[] = [
    { type: "theory", label: "Theory", short: "Theory", colour: "#a78bfa", group: "theories" },
    { type: "response", label: "Theory reply", short: "Reply", colour: "#a78bfa", group: "theories" },
    { type: "post", label: "Game Board post", short: "Post", colour: "#38bdf8", group: "posts" },
    { type: "post_comment", label: "Game Board comment", short: "Comment", colour: "#38bdf8", group: "posts" },
    { type: "art", label: "Artwork", short: "Art", colour: "#f472b6", group: "art" },
    { type: "art_comment", label: "Art comment", short: "Comment", colour: "#f472b6", group: "art" },
    { type: "mystery", label: "Mystery", short: "Mystery", colour: "#fb923c", group: "mysteries" },
    { type: "mystery_attempt", label: "Mystery solution", short: "Solution", colour: "#fb923c", group: "mysteries" },
    { type: "mystery_comment", label: "Mystery comment", short: "Comment", colour: "#fb923c", group: "mysteries" },
    { type: "ship", label: "Ship", short: "Ship", colour: "#fb7185", group: "ships" },
    { type: "ship_comment", label: "Ship comment", short: "Comment", colour: "#fb7185", group: "ships" },
    { type: "oc", label: "OC", short: "OC", colour: "#c084fc", group: "ocs" },
    { type: "oc_comment", label: "OC comment", short: "Comment", colour: "#c084fc", group: "ocs" },
    { type: "announcement", label: "Announcement", short: "News", colour: "#facc15", group: "announcements" },
    {
        type: "announcement_comment",
        label: "Announcement comment",
        short: "Comment",
        colour: "#facc15",
        group: "announcements",
    },
    { type: "fanfic", label: "Fanfiction", short: "Fanfic", colour: "#34d399", group: "fanfics" },
    { type: "fanfic_comment", label: "Fanfic comment", short: "Comment", colour: "#34d399", group: "fanfics" },
    { type: "journal", label: "Journal", short: "Journal", colour: "#60a5fa", group: "journals" },
    { type: "journal_entry", label: "Journal entry", short: "Entry", colour: "#60a5fa", group: "journals" },
    { type: "journal_comment", label: "Journal comment", short: "Comment", colour: "#60a5fa", group: "journals" },
    { type: "chat_message", label: "Chat message", short: "Chat", colour: "#22d3ee", group: "chats" },
    { type: "user", label: "User", short: "User", colour: "#e89ec0", group: "users" },
];

const everyEntityType: Record<SearchEntityType, true> = {
    theory: true,
    response: true,
    post: true,
    post_comment: true,
    art: true,
    art_comment: true,
    mystery: true,
    mystery_attempt: true,
    mystery_comment: true,
    ship: true,
    ship_comment: true,
    oc: true,
    oc_comment: true,
    announcement: true,
    announcement_comment: true,
    fanfic: true,
    fanfic_comment: true,
    journal: true,
    journal_entry: true,
    journal_comment: true,
    chat_message: true,
    user: true,
};

describe("SEARCH_TYPE_META", () => {
    for (const metaCase of metaCases) {
        it(`describes the ${metaCase.type} entity type`, () => {
            // given
            const type = metaCase.type;

            // when
            const meta = SEARCH_TYPE_META[type];

            // then
            expect(meta).toEqual({
                type,
                label: metaCase.label,
                short: metaCase.short,
                color: metaCase.colour,
                group: metaCase.group,
            });
        });
    }

    it("covers every entity type the search API can return", () => {
        // given
        const known = Object.keys(everyEntityType).sort();

        // when
        const described = Object.keys(SEARCH_TYPE_META).sort();

        // then
        expect(described).toEqual(known);
        expect(metaCases.map(c => c.type).sort()).toEqual(known);
    });

    it("keys every entry by its own type", () => {
        // given
        const entries = Object.entries(SEARCH_TYPE_META);

        // when
        const mismatched = entries.filter(([key, meta]) => key !== meta.type);

        // then
        expect(mismatched).toEqual([]);
    });

    it("gives every type in a group the same colour", () => {
        // given
        const coloursByGroup = new Map<SearchTypeGroup, Set<string>>();

        // when
        for (const meta of Object.values(SEARCH_TYPE_META)) {
            const colours = coloursByGroup.get(meta.group) ?? new Set<string>();
            colours.add(meta.color);
            coloursByGroup.set(meta.group, colours);
        }

        // then
        for (const [group, colours] of coloursByGroup) {
            expect({ group, count: colours.size }).toEqual({ group, count: 1 });
        }
    });

    it("only ever points at a group that is part of the display order", () => {
        // given
        const order = new Set(SEARCH_GROUP_ORDER);

        // when
        const strays = Object.values(SEARCH_TYPE_META).filter(meta => !order.has(meta.group));

        // then
        expect(strays).toEqual([]);
    });
});

describe("SEARCH_GROUP_ORDER", () => {
    it("lists the groups in the order the dropdown shows them", () => {
        // given
        const expected: SearchTypeGroup[] = [
            "theories",
            "posts",
            "art",
            "mysteries",
            "ships",
            "ocs",
            "fanfics",
            "journals",
            "announcements",
            "chats",
            "users",
        ];

        // when
        const order = SEARCH_GROUP_ORDER;

        // then
        expect(order).toEqual(expected);
    });

    it("mentions each group exactly once", () => {
        // given
        const order = SEARCH_GROUP_ORDER;

        // when
        const unique = new Set(order);

        // then
        expect(unique.size).toBe(order.length);
    });
});

describe("SEARCH_GROUP_LABEL", () => {
    it("gives every group a human readable heading", () => {
        // given
        const order = SEARCH_GROUP_ORDER;

        // when
        const labels = order.map(group => SEARCH_GROUP_LABEL[group]);

        // then
        expect(labels).toEqual([
            "Theories",
            "Game Board",
            "Art",
            "Mysteries",
            "Ships",
            "OCs",
            "Fanfiction",
            "Journals",
            "Announcements",
            "Chats",
            "Users",
        ]);
    });

    it("labels nothing that is not a group", () => {
        // given
        const order = new Set<string>(SEARCH_GROUP_ORDER);

        // when
        const labelled = Object.keys(SEARCH_GROUP_LABEL);

        // then
        expect(labelled.filter(key => !order.has(key))).toEqual([]);
    });
});

describe("SEARCH_FILTER_OPTIONS", () => {
    it("opens with an unfiltered option", () => {
        // given
        const options = SEARCH_FILTER_OPTIONS;

        // when
        const first = options[0];

        // then
        expect(first).toEqual({ value: "", label: "All" });
    });

    it("closes with a comments only option", () => {
        // given
        const options = SEARCH_FILTER_OPTIONS;

        // when
        const last = options[options.length - 1];

        // then
        expect(last).toEqual({ value: "comments", label: "Comments only" });
    });

    it("offers one option per group between the two special options", () => {
        // given
        const options = SEARCH_FILTER_OPTIONS;

        // when
        const groupOptions = options.slice(1, -1);

        // then
        expect(groupOptions.map(option => option.label)).toEqual(
            SEARCH_GROUP_ORDER.map(group => SEARCH_GROUP_LABEL[group]),
        );
    });

    it("packs every entity type of a group into that group's filter value", () => {
        // given
        const groupOptions = SEARCH_FILTER_OPTIONS.slice(1, -1);

        // when
        const values = groupOptions.map(option => option.value);

        // then
        expect(values).toEqual([
            "theory,response",
            "post,post_comment",
            "art,art_comment",
            "mystery,mystery_attempt,mystery_comment",
            "ship,ship_comment",
            "oc,oc_comment",
            "fanfic,fanfic_comment",
            "journal,journal_entry,journal_comment",
            "announcement,announcement_comment",
            "chat_message",
            "user",
        ]);
    });

    it("mentions every entity type across the group filters", () => {
        // given
        const groupOptions = SEARCH_FILTER_OPTIONS.slice(1, -1);

        // when
        const mentioned = groupOptions.flatMap(option => option.value.split(",")).sort();

        // then
        expect(mentioned).toEqual(Object.keys(everyEntityType).sort());
    });
});
