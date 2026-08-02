import { describe, expect, it } from "vitest";
import { auditActionLabel, auditTargetLabel, parseAuditDetails, shortId } from "./auditLog";

describe("auditActionLabel", () => {
    it("maps known actions to plain english", () => {
        expect(auditActionLabel("chat_word_filter_kick")).toBe("Kicked by the word filter");
        expect(auditActionLabel("watch_party.kick")).toBe("Kicked from a watch party");
    });

    it("falls back to a readable form for unknown actions", () => {
        expect(auditActionLabel("some_new_action")).toBe("some new action");
        expect(auditActionLabel("thing.happened")).toBe("thing happened");
    });
});

describe("auditTargetLabel", () => {
    it("maps internal target types to plain english", () => {
        expect(auditTargetLabel("chat_watch_party_session")).toBe("Watch party");
    });

    it("falls back for unknown types", () => {
        expect(auditTargetLabel("brand_new_thing")).toBe("brand new thing");
    });
});

describe("parseAuditDetails", () => {
    it("returns nothing for empty details", () => {
        expect(parseAuditDetails("")).toEqual([]);
        expect(parseAuditDetails("   ")).toEqual([]);
    });

    it("splits quoted key=value pairs and strips the quotes", () => {
        expect(parseAuditDetails('pattern="badword" match="a badword here"')).toEqual([
            { key: "pattern", value: "badword" },
            { key: "match", value: "a badword here" },
        ]);
    });

    it("keeps a trailing unquoted value whole, spaces included", () => {
        expect(parseAuditDetails("reason=being rude in the parlour")).toEqual([
            { key: "reason", value: "being rude in the parlour" },
        ]);
    });

    it("treats a bare value with no key as a single part", () => {
        expect(parseAuditDetails("admin")).toEqual([{ key: "", value: "admin" }]);
        expect(parseAuditDetails("old@example.com -> new@example.com")).toEqual([
            { key: "", value: "old@example.com -> new@example.com" },
        ]);
    });

    it("expands a json object into parts", () => {
        expect(parseAuditDetails('{"room_id":"r1","target_user_id":"u1"}')).toEqual([
            { key: "room_id", value: "r1" },
            { key: "target_user_id", value: "u1" },
        ]);
    });

    it("falls back to the raw string for malformed json", () => {
        expect(parseAuditDetails("{not json")).toEqual([{ key: "", value: "{not json" }]);
    });

    it("drops keys whose value is empty", () => {
        expect(parseAuditDetails("reason=")).toEqual([]);
    });

    it("handles escaped quotes inside a quoted value", () => {
        expect(parseAuditDetails('match="he said \\"no\\""')).toEqual([{ key: "match", value: 'he said "no"' }]);
    });
});

describe("shortId", () => {
    it("truncates long ids", () => {
        expect(shortId("35511476-c463-4d40-a028-8b6528779c98")).toBe("35511476...");
    });

    it("leaves short ids alone", () => {
        expect(shortId("abc")).toBe("abc");
    });
});
