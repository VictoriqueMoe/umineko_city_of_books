import { describe, expect, it } from "vitest";
import {
    CONTENT_RULES,
    contentPermissions,
    isContentOwner,
    type ContentFamily,
    type ContentViewer,
} from "./contentPermissions";

const author: ContentViewer = { id: "author", role: undefined };
const stranger: ContentViewer = { id: "stranger", role: undefined };
const moderator: ContentViewer = { id: "moderator", role: "moderator" };
const admin: ContentViewer = { id: "admin", role: "admin" };

function subject(family: ContentFamily) {
    return { family, authorId: "author" };
}

describe("isContentOwner", () => {
    it("matches the viewer against the author id", () => {
        // given
        const item = subject("post");

        // when
        const owner = isContentOwner(author, item);

        // then
        expect(owner).toBe(true);
    });

    it("treats a signed out viewer as nobody even when the author id is missing", () => {
        // given
        const item = { family: "post" as const, authorId: undefined };

        // when
        const owner = isContentOwner(null, item);

        // then
        expect(owner).toBe(false);
    });
});

describe("contentPermissions", () => {
    it("gives the author both actions on every family that offers them", () => {
        // given
        const families = Object.keys(CONTENT_RULES) as ContentFamily[];

        // when
        const results = families.map(family => ({ family, ...contentPermissions(author, subject(family)) }));

        // then
        for (const result of results) {
            const rules = CONTENT_RULES[result.family];
            expect(result.canEdit).toBe(rules.edit !== "no_surface");
            expect(result.canDelete).toBe(rules.delete !== "no_surface");
        }
    });

    it("gives a signed out viewer nothing anywhere", () => {
        // given
        const families = Object.keys(CONTENT_RULES) as ContentFamily[];

        // when
        const results = families.map(family => contentPermissions(undefined, subject(family)));

        // then
        expect(results.every(r => !r.canEdit && !r.canDelete)).toBe(true);
    });

    it("gives a stranger with no role nothing anywhere", () => {
        // given
        const families = Object.keys(CONTENT_RULES) as ContentFamily[];

        // when
        const results = families.map(family => contentPermissions(stranger, subject(family)));

        // then
        expect(results.every(r => !r.canEdit && !r.canDelete)).toBe(true);
    });

    const staffOverrides: { family: ContentFamily; canEdit: boolean; canDelete: boolean }[] = [
        { family: "post", canEdit: true, canDelete: true },
        { family: "fanfic", canEdit: true, canDelete: true },
        { family: "art", canEdit: true, canDelete: true },
        { family: "ship", canEdit: true, canDelete: true },
        { family: "theory", canEdit: true, canDelete: true },
        { family: "mystery", canEdit: true, canDelete: true },
        { family: "mystery_attempt", canEdit: false, canDelete: true },
        { family: "journal", canEdit: true, canDelete: true },
        { family: "journal_entry", canEdit: true, canDelete: true },
        { family: "comment", canEdit: true, canDelete: true },
        { family: "response", canEdit: false, canDelete: true },
        { family: "oc", canEdit: false, canDelete: false },
        { family: "gallery", canEdit: false, canDelete: false },
    ];

    for (const testCase of staffOverrides) {
        it(`resolves the moderator override on ${testCase.family}`, () => {
            // given
            const item = subject(testCase.family);

            // when
            const permissions = contentPermissions(moderator, item);

            // then
            expect(permissions).toEqual({ canEdit: testCase.canEdit, canDelete: testCase.canDelete });
        });
    }

    it("does not let an admin edit or delete someone else's original character", () => {
        // given
        const item = subject("oc");

        // when
        const permissions = contentPermissions(admin, item);

        // then
        expect(permissions).toEqual({ canEdit: false, canDelete: false });
    });

    it("does not let an admin edit or delete someone else's gallery", () => {
        // given
        const item = subject("gallery");

        // when
        const permissions = contentPermissions(admin, item);

        // then
        expect(permissions).toEqual({ canEdit: false, canDelete: false });
    });

    it("gates a mystery attempt on the comment permission rather than a theory one", () => {
        // given
        const commentOnly: ContentViewer = { id: "staff", permissions: ["delete_any_comment"] };

        // when
        const permissions = contentPermissions(commentOnly, subject("mystery_attempt"));

        // then
        expect(permissions).toEqual({ canEdit: false, canDelete: true });
    });

    it("reads an explicit permission list in preference to the role", () => {
        // given
        const scoped: ContentViewer = { id: "scoped", role: "moderator", permissions: ["edit_any_post"] };

        // when
        const permissions = contentPermissions(scoped, subject("post"));

        // then
        expect(permissions).toEqual({ canEdit: true, canDelete: false });
    });
});
