import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./admin";
import {
    deleteMock,
    deleteWithBodyMock,
    fetchMock,
    postFormDataMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

vi.mock("../../platform/capabilities", () => ({
    isNativeApp: () => false,
    clientPlatform: () => "web",
}));

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

describe("admin user management", () => {
    const id = "u-1";
    const cases: RequestCase[] = [
        {
            name: "getAdminUser reads a single user",
            call: () => api.getAdminUser(id),
            transport: fetchMock,
            request: [`/admin/users/${id}`],
        },
        {
            name: "setUserRole posts the role",
            call: () => api.setUserRole(id, "moderator"),
            transport: postMock,
            request: [`/admin/users/${id}/role`, { role: "moderator" }],
        },
        {
            name: "removeUserRole deletes with the role in the body",
            call: () => api.removeUserRole(id, "moderator"),
            transport: deleteWithBodyMock,
            request: [`/admin/users/${id}/role`, { role: "moderator" }],
        },
        {
            name: "updateDetectiveScore puts the desired mystery score",
            call: () => api.updateDetectiveScore(id, 42),
            transport: putMock,
            request: [`/admin/users/${id}/mystery-score`, { desired_score: 42 }],
        },
        {
            name: "updateGMScore puts the desired gm score",
            call: () => api.updateGMScore(id, 7),
            transport: putMock,
            request: [`/admin/users/${id}/gm-score`, { desired_score: 7 }],
        },
        {
            name: "banUser sends the reason",
            call: () => api.banUser(id, "being rude in the parlour"),
            transport: postMock,
            request: [`/admin/users/${id}/ban`, { reason: "being rude in the parlour" }],
        },
        {
            name: "unbanUser sends no body",
            call: () => api.unbanUser(id),
            transport: postMock,
            request: [`/admin/users/${id}/unban`, undefined],
        },
        {
            name: "lockUser sends the reason",
            call: () => api.lockUser(id, "suspicious logins"),
            transport: postMock,
            request: [`/admin/users/${id}/lock`, { reason: "suspicious logins" }],
        },
        {
            name: "unlockUser sends no body",
            call: () => api.unlockUser(id),
            transport: postMock,
            request: [`/admin/users/${id}/unlock`, undefined],
        },
        {
            name: "adminDeleteUser deletes the user",
            call: () => api.adminDeleteUser(id),
            transport: deleteMock,
            request: [`/admin/users/${id}`],
        },
        {
            name: "resetUserPassword posts with no body",
            call: () => api.resetUserPassword(id),
            transport: postMock,
            request: [`/admin/users/${id}/reset-password`, undefined],
        },
        {
            name: "setUserEmail puts the new address",
            call: () => api.setUserEmail(id, "kujo@example.com"),
            transport: putMock,
            request: [`/admin/users/${id}/email`, { email: "kujo@example.com" }],
        },
        {
            name: "verifyUserEmail posts with no body",
            call: () => api.verifyUserEmail(id),
            transport: postMock,
            request: [`/admin/users/${id}/verify-email`, undefined],
        },
        {
            name: "unverifyUserEmail posts with no body",
            call: () => api.unverifyUserEmail(id),
            transport: postMock,
            request: [`/admin/users/${id}/unverify-email`, undefined],
        },
        {
            name: "setUserDisplayName puts the snake cased field",
            call: () => api.setUserDisplayName(id, "Victorique"),
            transport: putMock,
            request: [`/admin/users/${id}/display-name`, { display_name: "Victorique" }],
        },
        {
            name: "setDisplayNameLock puts the lock flag",
            call: () => api.setDisplayNameLock(id, true),
            transport: putMock,
            request: [`/admin/users/${id}/display-name-lock`, { locked: true }],
        },
        {
            name: "setDisplayNameLock can unlock again",
            call: () => api.setDisplayNameLock(id, false),
            transport: putMock,
            request: [`/admin/users/${id}/display-name-lock`, { locked: false }],
        },
        {
            name: "forceLogoutUser posts with no body",
            call: () => api.forceLogoutUser(id),
            transport: postMock,
            request: [`/admin/users/${id}/force-logout`, undefined],
        },
        {
            name: "getUserIPMatches reads the shared address list",
            call: () => api.getUserIPMatches(id),
            transport: fetchMock,
            request: [`/admin/users/${id}/ip-matches`],
        },
        {
            name: "getUserAuditLog defaults to the first page of twenty",
            call: () => api.getUserAuditLog(id),
            transport: fetchMock,
            request: [`/admin/users/${id}/audit-log?limit=20`],
        },
        {
            name: "getUserAuditLog pages through the log",
            call: () => api.getUserAuditLog(id, 5, 10),
            transport: fetchMock,
            request: [`/admin/users/${id}/audit-log?limit=5&offset=10`],
        },
    ];

    runRequestCases(cases);
});

describe("admin statistics, settings and invites", () => {
    const cases: RequestCase[] = [
        {
            name: "getAdminStats reads the dashboard statistics",
            call: () => api.getAdminStats(),
            transport: fetchMock,
            request: ["/admin/stats"],
        },
        {
            name: "updateAdminSettings wraps the settings in an envelope",
            call: () => api.updateAdminSettings({ site_name: "When They Cry", default_theme: "gold" }),
            transport: putMock,
            request: ["/admin/settings", { settings: { site_name: "When They Cry", default_theme: "gold" } }],
        },
        {
            name: "sendTestEmail posts with no body",
            call: () => api.sendTestEmail(),
            transport: postMock,
            request: ["/admin/settings/test-email", undefined],
        },
        {
            name: "createInvite posts with no body",
            call: () => api.createInvite(),
            transport: postMock,
            request: ["/admin/invites", undefined],
        },
        {
            name: "getInvites defaults to the first page of fifty",
            call: () => api.getInvites({}),
            transport: fetchMock,
            request: ["/admin/invites?limit=50"],
        },
        {
            name: "getInvites pages through the invite list",
            call: () => api.getInvites({ limit: 10, offset: 20 }),
            transport: fetchMock,
            request: ["/admin/invites?limit=10&offset=20"],
        },
        {
            name: "deleteInvite deletes the invite by its code",
            call: () => api.deleteInvite("beato-2026"),
            transport: deleteMock,
            request: ["/admin/invites/beato-2026"],
        },
    ];

    runRequestCases(cases);
});

describe("site wide banned word rules", () => {
    const substringRule = {
        pattern: "goat",
        match_mode: "substring" as const,
        case_sensitive: false,
        action: "delete" as const,
    };
    const regexRule = {
        pattern: "^witch$",
        match_mode: "regex" as const,
        case_sensitive: true,
        action: "kick" as const,
    };

    const cases: RequestCase[] = [
        {
            name: "listGlobalBannedWords reads the site wide rules",
            call: () => api.listGlobalBannedWords(),
            transport: fetchMock,
            request: ["/admin/banned-words"],
        },
        {
            name: "createGlobalBannedWord posts the whole rule to the site wide collection",
            call: () => api.createGlobalBannedWord(substringRule),
            transport: postMock,
            request: ["/admin/banned-words", substringRule],
        },
        {
            name: "updateGlobalBannedWord puts the rule back under its own id",
            call: () => api.updateGlobalBannedWord("rule-1", regexRule),
            transport: putMock,
            request: ["/admin/banned-words/rule-1", regexRule],
        },
        {
            name: "deleteGlobalBannedWord deletes the site wide rule",
            call: () => api.deleteGlobalBannedWord("rule-1"),
            transport: deleteMock,
            request: ["/admin/banned-words/rule-1"],
        },
    ];

    runRequestCases(cases);
});

describe("admin announcement management", () => {
    const cases: RequestCase[] = [
        {
            name: "createAnnouncement posts the title and body to the admin collection",
            call: () => api.createAnnouncement("The golden feast", "Everyone is invited to the parlour"),
            transport: postMock,
            request: [
                "/admin/announcements",
                { title: "The golden feast", body: "Everyone is invited to the parlour" },
            ],
        },
        {
            name: "updateAnnouncement puts the title and body back under the admin path",
            call: () => api.updateAnnouncement("an-1", "The golden feast", "Postponed until the seventh twilight"),
            transport: putMock,
            request: [
                "/admin/announcements/an-1",
                { title: "The golden feast", body: "Postponed until the seventh twilight" },
            ],
        },
        {
            name: "deleteAnnouncement deletes the announcement under the admin path",
            call: () => api.deleteAnnouncement("an-1"),
            transport: deleteMock,
            request: ["/admin/announcements/an-1"],
        },
        {
            name: "pinAnnouncement posts the pinned flag",
            call: () => api.pinAnnouncement("an-1", true),
            transport: postMock,
            request: ["/admin/announcements/an-1/pin", { pinned: true }],
        },
        {
            name: "pinAnnouncement can unpin the announcement again",
            call: () => api.pinAnnouncement("an-1", false),
            transport: postMock,
            request: ["/admin/announcements/an-1/pin", { pinned: false }],
        },
    ];

    runRequestCases(cases);
});

describe("vanity roles and banned gifs", () => {
    const rolePayload = { label: "Golden Witch", color: "#d4af37", sort_order: 1 };

    const cases: RequestCase[] = [
        {
            name: "getVanityRoles reads the role definitions",
            call: () => api.getVanityRoles(),
            transport: fetchMock,
            request: ["/admin/vanity-roles"],
        },
        {
            name: "createVanityRole posts the whole payload to the role collection",
            call: () => api.createVanityRole(rolePayload),
            transport: postMock,
            request: ["/admin/vanity-roles", rolePayload],
        },
        {
            name: "updateVanityRole puts the whole payload back under its own id",
            call: () => api.updateVanityRole("role-1", rolePayload),
            transport: putMock,
            request: ["/admin/vanity-roles/role-1", rolePayload],
        },
        {
            name: "deleteVanityRole deletes the role",
            call: () => api.deleteVanityRole("role-1"),
            transport: deleteMock,
            request: ["/admin/vanity-roles/role-1"],
        },
        {
            name: "assignVanityRole posts the user id to the role members",
            call: () => api.assignVanityRole("role-1", "u-1"),
            transport: postMock,
            request: ["/admin/vanity-roles/role-1/users", { user_id: "u-1" }],
        },
        {
            name: "unassignVanityRole deletes the named member of the role",
            call: () => api.unassignVanityRole("role-1", "u-1"),
            transport: deleteMock,
            request: ["/admin/vanity-roles/role-1/users/u-1"],
        },
        {
            name: "getBannedGifs reads the banned gif entries",
            call: () => api.getBannedGifs(),
            transport: fetchMock,
            request: ["/admin/banned-gifs"],
        },
        {
            name: "addBannedGif sends only the input when no reason was given",
            call: () => api.addBannedGif({ input: "https://giphy.com/gifs/abc" }),
            transport: postMock,
            request: ["/admin/banned-gifs", { input: "https://giphy.com/gifs/abc" }],
        },
        {
            name: "addBannedGif forwards the reason alongside the input",
            call: () => api.addBannedGif({ input: "https://giphy.com/gifs/abc", reason: "spoilers" }),
            transport: postMock,
            request: ["/admin/banned-gifs", { input: "https://giphy.com/gifs/abc", reason: "spoilers" }],
        },
    ];

    runRequestCases(cases);
});

describe("admin uploads", () => {
    it("uploadOGDefaultImage posts the file under the image field", async () => {
        // given
        const file = new File(["x"], "og.png", { type: "image/png" });

        // when
        await api.uploadOGDefaultImage(file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/admin/settings/og-image");
        expect(postFormDataMock.mock.calls[0][1].get("image")).toBe(file);
    });
});
