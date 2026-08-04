import { beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { apiDelete, apiDeleteWithBody, apiFetch, apiPatch, apiPost, apiPostFormData, apiPut } from "./client";
import { clearAuthToken } from "../utils/authToken";
import * as api from "./endpoints";

vi.mock("../utils/authToken", () => ({
    clearAuthToken: vi.fn(),
    isNativeApp: () => false,
    clientPlatform: () => "web",
    getAuthToken: () => null,
    setAuthToken: vi.fn(),
    loadAuthToken: vi.fn(),
}));

vi.mock("./client", async importOriginal => {
    const actual = await importOriginal<typeof import("./client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

interface RequestCase {
    name: string;
    call: () => Promise<unknown>;
    transport: Mock;
    request: unknown[];
}

const fetchMock = vi.mocked(apiFetch);
const postMock = vi.mocked(apiPost);
const putMock = vi.mocked(apiPut);
const patchMock = vi.mocked(apiPatch);
const deleteMock = vi.mocked(apiDelete);
const deleteWithBodyMock = vi.mocked(apiDeleteWithBody);
const postFormDataMock = vi.mocked(apiPostFormData);
const globalFetch = vi.fn();

beforeEach(() => {
    fetchMock.mockResolvedValue({});
    postMock.mockResolvedValue({});
    putMock.mockResolvedValue({});
    patchMock.mockResolvedValue({});
    deleteMock.mockResolvedValue({});
    deleteWithBodyMock.mockResolvedValue({});
    postFormDataMock.mockResolvedValue({});
    globalFetch.mockResolvedValue({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
        text: () => Promise.resolve(""),
    });
    vi.stubGlobal("fetch", globalFetch);
});

describe("query string building", () => {
    const cases: RequestCase[] = [
        {
            name: "listTheories defaults to the umineko series and a page of twenty",
            call: () => api.listTheories({}),
            transport: fetchMock,
            request: ["/theories?series=umineko&limit=20"],
        },
        {
            name: "listTheories passes every filter through",
            call: () =>
                api.listTheories({
                    sort: "top",
                    episode: 3,
                    author: "battler",
                    search: "gold",
                    series: "higurashi",
                    limit: 5,
                    offset: 10,
                }),
            transport: fetchMock,
            request: ["/theories?sort=top&episode=3&author=battler&search=gold&series=higurashi&limit=5&offset=10"],
        },
        {
            name: "listTheories keeps an episode of zero",
            call: () => api.listTheories({ episode: 0 }),
            transport: fetchMock,
            request: ["/theories?episode=0&series=umineko&limit=20"],
        },
        {
            name: "getAdminUsers form encodes the search term",
            call: () => api.getAdminUsers({ search: "beato & co" }),
            transport: fetchMock,
            request: ["/admin/users?search=beato+%26+co&limit=20"],
        },
        {
            name: "getReports defaults to open reports and drops an offset of zero",
            call: () => api.getReports(),
            transport: fetchMock,
            request: ["/admin/reports?status=open&limit=50"],
        },
        {
            name: "getAuditLog filters by action",
            call: () => api.getAuditLog({ action: "user.ban", limit: 10, offset: 20 }),
            transport: fetchMock,
            request: ["/admin/audit-log?action=user.ban&limit=10&offset=20"],
        },
        {
            name: "listFanfics only sends the boolean filters that are switched on",
            call: () => api.listFanfics({ pairing: true, lemons: false, search: "beato" }),
            transport: fetchMock,
            request: ["/fanfics?pairing=true&search=beato&limit=25"],
        },
        {
            name: "listJournals drops an empty work filter",
            call: () => api.listJournals({ work: "", includeArchived: true }),
            transport: fetchMock,
            request: ["/journals?include_archived=true&limit=20"],
        },
        {
            name: "listPublicChatRooms only flags rp and archived rooms when they are wanted",
            call: () => api.listPublicChatRooms({ rp: true, tag: "rp", includeArchived: false }),
            transport: fetchMock,
            request: ["/chat/rooms/public?rp=true&tag=rp&limit=20"],
        },
        {
            name: "listMyChatRooms filters by role",
            call: () => api.listMyChatRooms({ role: "host", includeArchived: true }),
            transport: fetchMock,
            request: ["/chat/rooms/mine?role=host&include_archived=true&limit=20"],
        },
        {
            name: "listShips sends no query string when nothing is filtered",
            call: () => api.listShips({}),
            transport: fetchMock,
            request: ["/ships"],
        },
        {
            name: "listShips flags crackships only when asked",
            call: () => api.listShips({ crackships: true, limit: 10, offset: 20 }),
            transport: fetchMock,
            request: ["/ships?crackships=true&limit=10&offset=20"],
        },
        {
            name: "listOCs flags crack characters",
            call: () => api.listOCs({ user_id: "u-1", crack: true }),
            transport: fetchMock,
            request: ["/ocs?user_id=u-1&crack=true"],
        },
        {
            name: "searchGiphy sends only the query when no paging is asked for",
            call: () => api.searchGiphy("cat"),
            transport: fetchMock,
            request: ["/giphy/search?q=cat"],
        },
        {
            name: "searchGiphy drops a zero offset but keeps an explicit zero limit",
            call: () => api.searchGiphy("cat", 0, 0),
            transport: fetchMock,
            request: ["/giphy/search?q=cat&limit=0"],
        },
        {
            name: "listMyGameRooms sends no query string when no filters are given",
            call: () => api.listMyGameRooms(),
            transport: fetchMock,
            request: ["/game-rooms"],
        },
        {
            name: "listFinishedGameRooms defaults to the first page of twenty",
            call: () => api.listFinishedGameRooms(),
            transport: fetchMock,
            request: ["/game-rooms/finished?limit=20"],
        },
        {
            name: "searchSite drops the empty types filter",
            call: () => api.searchSite("beato"),
            transport: fetchMock,
            request: ["/search?q=beato&limit=20"],
        },
        {
            name: "searchSite passes types, paging and the room scope",
            call: () => api.searchSite("beato", "posts", 10, 5, "room-1"),
            transport: fetchMock,
            request: ["/search?q=beato&types=posts&limit=10&offset=5&room=room-1"],
        },
        {
            name: "quickSearch defaults to three results per type",
            call: () => api.quickSearch("bea"),
            transport: fetchMock,
            request: ["/search/quick?q=bea&perType=3"],
        },
        {
            name: "getRoomMessagesBefore encodes the cursor and defaults to fifty messages",
            call: () => api.getRoomMessagesBefore("r-1", "2026-01-01T00:00:00Z"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/messages?before=2026-01-01T00%3A00%3A00Z&limit=50"],
        },
        {
            name: "getVanityRoleUsers always sends a limit",
            call: () => api.getVanityRoleUsers("role-1", {}),
            transport: fetchMock,
            request: ["/admin/vanity-roles/role-1/users?limit=20"],
        },
        {
            name: "getVanityRoleUsers percent encodes the search term",
            call: () => api.getVanityRoleUsers("role-1", { search: "a b", limit: 5, offset: 10 }),
            transport: fetchMock,
            request: ["/admin/vanity-roles/role-1/users?search=a%20b&limit=5&offset=10"],
        },
        {
            name: "getPopularTags sends no query string without a corner",
            call: () => api.getPopularTags(),
            transport: fetchMock,
            request: ["/art/tags"],
        },
        {
            name: "getPopularTags encodes the corner",
            call: () => api.getPopularTags("umineko & co"),
            transport: fetchMock,
            request: ["/art/tags?corner=umineko%20%26%20co"],
        },
        {
            name: "listPosts passes its parameters straight through",
            call: () => api.listPosts({ tab: "all", corner: "general", seed: 5 }),
            transport: fetchMock,
            request: ["/posts?tab=all&corner=general&seed=5"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("path segment encoding", () => {
    const cases: RequestCase[] = [
        {
            name: "searchUsers encodes the query",
            call: () => api.searchUsers("beato & co"),
            transport: fetchMock,
            request: ["/users/search?q=beato%20%26%20co"],
        },
        {
            name: "resolveUsernames joins the names with commas and encodes them",
            call: () => api.resolveUsernames(["kujo", "victorique gosick"]),
            transport: fetchMock,
            request: ["/users/resolve?usernames=kujo%2Cvictorique%20gosick"],
        },
        {
            name: "removeChatMessageReaction encodes the emoji",
            call: () => api.removeChatMessageReaction("m-1", "🍬"),
            transport: deleteMock,
            request: ["/chat/messages/m-1/reactions/%F0%9F%8D%AC"],
        },
        {
            name: "removeBannedGif encodes both the kind and the value",
            call: () => api.removeBannedGif("user", "some user"),
            transport: deleteMock,
            request: ["/admin/banned-gifs/user/some%20user"],
        },
        {
            name: "removeGiphyFavourite encodes a slash in the id",
            call: () => api.removeGiphyFavourite("gif/1"),
            transport: deleteMock,
            request: ["/giphy/favourites/gif%2F1"],
        },
        {
            name: "deleteInvite encodes a slash in the invite code",
            call: () => api.deleteInvite("beato/2026"),
            transport: deleteMock,
            request: ["/admin/invites/beato%2F2026"],
        },
        {
            name: "getRules encodes the page name",
            call: () => api.getRules("chat rules#top"),
            transport: fetchMock,
            request: ["/rules/chat%20rules%23top"],
        },
        {
            name: "getUserProfile encodes the username",
            call: () => api.getUserProfile("victorique gosick"),
            transport: fetchMock,
            request: ["/users/victorique%20gosick"],
        },
        {
            name: "getTheory encodes the theory id",
            call: () => api.getTheory("t/1"),
            transport: fetchMock,
            request: ["/theories/t%2F1"],
        },
        {
            name: "resolveDMRoom encodes the recipient id",
            call: () => api.resolveDMRoom("u 1#x"),
            transport: fetchMock,
            request: ["/chat/dm/u%201%23x/resolve"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

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

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("nested resource paths", () => {
    const cases: RequestCase[] = [
        {
            name: "getWatchPartyVoiceToken nests the session under the room",
            call: () => api.getWatchPartyVoiceToken("r-1", "s-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/voice/token", {}],
        },
        {
            name: "forceMuteWatchPartyVoiceParticipant nests the room, session and user",
            call: () => api.forceMuteWatchPartyVoiceParticipant("r-1", "s-1", "u-1", true),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/voice/participants/u-1/mute", { muted: true }],
        },
        {
            name: "kickWatchPartyParticipant deletes the named participant",
            call: () => api.kickWatchPartyParticipant("r-1", "s-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/participants/u-1"],
        },
        {
            name: "leaveWatchParty deletes the caller's own participant",
            call: () => api.leaveWatchParty("r-1", "s-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/participants/me"],
        },
        {
            name: "transferWatchPartyControl patches the target participant",
            call: () => api.transferWatchPartyControl("r-1", "s-1", "u-1"),
            transport: patchMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/participants/u-1", {}],
        },
        {
            name: "setChatRoomMemberTimeout sends the amount and unit",
            call: () => api.setChatRoomMemberTimeout("r-1", "u-1", 5, "minutes"),
            transport: putMock,
            request: ["/chat/rooms/r-1/members/u-1/timeout", { amount: 5, unit: "minutes" }],
        },
        {
            name: "getFanficChapter interpolates a numeric chapter",
            call: () => api.getFanficChapter("f-1", 3),
            transport: fetchMock,
            request: ["/fanfics/f-1/chapters/3"],
        },
        {
            name: "deleteJournalEntryMedia interpolates a numeric media id",
            call: () => api.deleteJournalEntryMedia("e-1", 4),
            transport: deleteMock,
            request: ["/journal-entries/e-1/media/4"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("optional request body fields", () => {
    const cases: RequestCase[] = [
        {
            name: "register leaves the invite code and turnstile token unset when they are not supplied",
            call: () => api.register("kujo", "kujo@example.com", "pw", "Kujo"),
            transport: postMock,
            request: [
                "/auth/register",
                {
                    username: "kujo",
                    email: "kujo@example.com",
                    password: "pw",
                    display_name: "Kujo",
                    invite_code: undefined,
                    turnstile_token: undefined,
                },
            ],
        },
        {
            name: "register forwards the invite code and turnstile token",
            call: () => api.register("kujo", "kujo@example.com", "pw", "Kujo", "invite-1", "token-1"),
            transport: postMock,
            request: [
                "/auth/register",
                {
                    username: "kujo",
                    email: "kujo@example.com",
                    password: "pw",
                    display_name: "Kujo",
                    invite_code: "invite-1",
                    turnstile_token: "token-1",
                },
            ],
        },
        {
            name: "login sends an undefined turnstile token when there is none",
            call: () => api.login("kujo", "pw"),
            transport: postMock,
            request: ["/auth/login", { username: "kujo", password: "pw", turnstile_token: undefined }],
        },
        {
            name: "createPost defaults to the general corner with no poll or share",
            call: () => api.createPost("hello"),
            transport: postMock,
            request: [
                "/posts",
                {
                    body: "hello",
                    corner: "general",
                    poll: undefined,
                    shared_content_id: undefined,
                    shared_content_type: undefined,
                },
            ],
        },
        {
            name: "createPost forwards a poll and the shared content",
            call: () =>
                api.createPost(
                    "hello",
                    "suggestions",
                    { options: [{ label: "yes" }], duration_seconds: 3600 },
                    "art-1",
                    "art",
                ),
            transport: postMock,
            request: [
                "/posts",
                {
                    body: "hello",
                    corner: "suggestions",
                    poll: { options: [{ label: "yes" }], duration_seconds: 3600 },
                    shared_content_id: "art-1",
                    shared_content_type: "art",
                },
            ],
        },
        {
            name: "joinChatRoom omits the ghost flag when no options are given",
            call: () => api.joinChatRoom("r-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/join", { ghost: undefined }],
        },
        {
            name: "joinChatRoom forwards the ghost flag",
            call: () => api.joinChatRoom("r-1", { ghost: true }),
            transport: postMock,
            request: ["/chat/rooms/r-1/join", { ghost: true }],
        },
        {
            name: "createJournalComment can target an entry without a parent comment",
            call: () => api.createJournalComment("j-1", "lovely", undefined, "e-1"),
            transport: postMock,
            request: ["/journals/j-1/comments", { body: "lovely", parent_id: undefined, entry_id: "e-1" }],
        },
        {
            name: "addMysteryClue leaves the player unset for a public clue",
            call: () => api.addMysteryClue("m-1", "the gold is real", "red"),
            transport: postMock,
            request: ["/mysteries/m-1/clues", { body: "the gold is real", truth_type: "red", player_id: undefined }],
        },
        {
            name: "createReport leaves the context id unset when there is no context",
            call: () => api.createReport("post", "p-1", "spam"),
            transport: postMock,
            request: ["/report", { target_type: "post", target_id: "p-1", context_id: undefined, reason: "spam" }],
        },
        {
            name: "resolveSuggestion defaults the status to done",
            call: () => api.resolveSuggestion("p-1"),
            transport: postMock,
            request: ["/posts/p-1/resolve", { status: "done" }],
        },
        {
            name: "createGallery defaults the description to an empty string",
            call: () => api.createGallery("Witches"),
            transport: postMock,
            request: ["/galleries", { name: "Witches", description: "" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("multipart uploads", () => {
    function lastFormData(): FormData {
        const calls = postFormDataMock.mock.calls;
        return calls[calls.length - 1][1];
    }

    it("uploadAvatar posts the file under the avatar field", async () => {
        // given
        const file = new File(["x"], "avatar.png", { type: "image/png" });

        // when
        await api.uploadAvatar(file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/auth/avatar");
        expect(lastFormData().get("avatar")).toBe(file);
    });

    it("sendChatMessage attaches the reply target and every media file", async () => {
        // given
        const first = new File(["a"], "a.png", { type: "image/png" });
        const second = new File(["b"], "b.png", { type: "image/png" });

        // when
        await api.sendChatMessage("r-1", { body: "hello", reply_to_id: "m-0", files: [first, second] });

        // then
        const formData = lastFormData();
        expect(postFormDataMock.mock.calls[0][0]).toBe("/chat/rooms/r-1/messages");
        expect(formData.get("body")).toBe("hello");
        expect(formData.get("reply_to_id")).toBe("m-0");
        expect(formData.getAll("media")).toEqual([first, second]);
    });

    it("sendChatMessage omits the reply target and media when there are none", async () => {
        // given
        const payload = { body: "hello" };

        // when
        await api.sendChatMessage("r-1", payload);

        // then
        const formData = lastFormData();
        expect(formData.has("reply_to_id")).toBe(false);
        expect(formData.getAll("media")).toEqual([]);
    });

    it("createArt serialises the metadata alongside the image", async () => {
        // given
        const metadata = {
            title: "Golden Witch",
            description: "",
            corner: "umineko",
            art_type: "digital",
            tags: ["beatrice"],
            is_spoiler: false,
        };
        const image = new File(["x"], "art.png", { type: "image/png" });

        // when
        await api.createArt(metadata, image);

        // then
        const formData = lastFormData();
        expect(postFormDataMock.mock.calls[0][0]).toBe("/art");
        expect(formData.get("metadata")).toBe(JSON.stringify(metadata));
        expect(formData.get("image")).toBe(image);
    });

    it("addOCGalleryImage only sends a caption when one was written", async () => {
        // given
        const image = new File(["x"], "oc.png", { type: "image/png" });

        // when
        await api.addOCGalleryImage("oc-1", image, "");

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/ocs/oc-1/gallery");
        expect(lastFormData().has("caption")).toBe(false);
    });

    it("uploadStreamThumbnail names the blob so the server sees a webp", async () => {
        // given
        const blob = new Blob(["x"], { type: "image/webp" });

        // when
        await api.uploadStreamThumbnail("s-1", blob);

        // then
        const thumbnail = lastFormData().get("thumbnail") as File;
        expect(postFormDataMock.mock.calls[0][0]).toBe("/streams/s-1/thumbnail");
        expect(thumbnail.name).toBe("thumb.webp");
    });
});

describe("response unwrapping", () => {
    it("getAdminSettings unwraps the settings envelope", async () => {
        // given
        const settings = { site_name: "When They Cry" };
        fetchMock.mockResolvedValue({ settings });

        // when
        const result = await api.getAdminSettings();

        // then
        expect(fetchMock).toHaveBeenCalledWith("/admin/settings");
        expect(result).toEqual(settings);
    });

    it("getFanficLanguages returns just the language list", async () => {
        // given
        fetchMock.mockResolvedValue({ languages: ["English", "Japanese"] });

        // when
        const result = await api.getFanficLanguages();

        // then
        expect(result).toEqual(["English", "Japanese"]);
    });

    it("searchOCCharacters queries by name and returns just the characters", async () => {
        // given
        fetchMock.mockResolvedValue({ characters: ["Beatrice"] });

        // when
        const result = await api.searchOCCharacters("bea");

        // then
        expect(fetchMock).toHaveBeenCalledWith("/fanfic-oc-characters?q=bea");
        expect(result).toEqual(["Beatrice"]);
    });

    it("getMe returns null without asking for a profile when nobody is signed in", async () => {
        // given
        fetchMock.mockResolvedValue({ authenticated: false });

        // when
        const result = await api.getMe();

        // then
        expect(result).toBeNull();
        expect(fetchMock).toHaveBeenCalledTimes(1);
        expect(fetchMock).toHaveBeenCalledWith("/auth/session");
    });

    it("getMe fetches the profile of the signed in username", async () => {
        // given
        const profile = { username: "kujo" };
        fetchMock.mockResolvedValueOnce({ authenticated: true, username: "kujo" }).mockResolvedValueOnce(profile);

        // when
        const result = await api.getMe();

        // then
        expect(fetchMock).toHaveBeenNthCalledWith(2, "/users/kujo");
        expect(result).toEqual(profile);
    });
});

describe("the quote API", () => {
    it("searchQuotes defaults to umineko and a page of thirty", async () => {
        // given
        const params = { query: "gold" };

        // when
        await api.searchQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/umineko/search?q=gold&limit=30");
    });

    it("searchQuotes keeps the language filter for umineko", async () => {
        // given
        const params = { query: "gold", lang: "jp" };

        // when
        await api.searchQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith(
            "https://quotes.auaurora.moe/api/v1/umineko/search?q=gold&lang=jp&limit=30",
        );
    });

    it("searchQuotes drops the language filter for the other series", async () => {
        // given
        const params = { query: "gold", lang: "jp", series: "higurashi" as const };

        // when
        await api.searchQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/higurashi/search?q=gold&limit=30");
    });

    it("browseQuotes pages through a character's lines", async () => {
        // given
        const params = { character: "Beatrice", episode: 4, limit: 10, offset: 20 };

        // when
        await api.browseQuotes(params);

        // then
        expect(globalFetch).toHaveBeenCalledWith(
            "https://quotes.auaurora.moe/api/v1/umineko/browse?character=Beatrice&episode=4&limit=10&offset=20",
        );
    });

    it("searchQuotes reports the status when the quote API fails", async () => {
        // given
        globalFetch.mockResolvedValue({ ok: false, status: 503 });

        // when
        const attempt = api.searchQuotes({ query: "gold" });

        // then
        await expect(attempt).rejects.toThrow("Quote API error: 503");
    });

    it("getCharacters merges the main and additional casts", async () => {
        // given
        globalFetch.mockResolvedValue({
            ok: true,
            json: () => Promise.resolve({ characters: { bea: "Beatrice" }, additional: { kanon: "Kanon" } }),
        });

        // when
        const result = await api.getCharacters("higurashi");

        // then
        expect(globalFetch).toHaveBeenCalledWith("https://quotes.auaurora.moe/api/v1/higurashi/characters");
        expect(result).toEqual({ bea: "Beatrice", kanon: "Kanon" });
    });

    it("getCharacterGroups falls back to empty groups when the payload is bare", async () => {
        // given
        globalFetch.mockResolvedValue({ ok: true, json: () => Promise.resolve({}) });

        // when
        const result = await api.getCharacterGroups();

        // then
        expect(result).toEqual({ main: {}, additional: {} });
    });
});

describe("session and overlay helpers", () => {
    it("logout clears the stored auth token after the server call", async () => {
        // given
        const clearToken = vi.mocked(clearAuthToken);

        // when
        await api.logout();

        // then
        expect(postMock).toHaveBeenCalledWith("/auth/logout", undefined);
        expect(clearToken).toHaveBeenCalledOnce();
    });

    it("fetchOverlayConnectorSEF returns the connector file as text", async () => {
        // given
        globalFetch.mockResolvedValue({ ok: true, text: () => Promise.resolve("<sef/>") });

        // when
        const result = await api.fetchOverlayConnectorSEF();

        // then
        expect(globalFetch).toHaveBeenCalledWith("/api/v1/overlay/connector.sef", {
            credentials: "include",
            headers: {},
        });
        expect(result).toBe("<sef/>");
    });

    it("fetchOverlayConnectorSEF explains that the download failed", async () => {
        // given
        globalFetch.mockResolvedValue({ ok: false, status: 500 });

        // when
        const attempt = api.fetchOverlayConnectorSEF();

        // then
        await expect(attempt).rejects.toThrow("Could not download the connector file.");
    });
});

describe("site metadata", () => {
    const cases: RequestCase[] = [
        {
            name: "getSiteInfo reads the site info document",
            call: () => api.getSiteInfo(),
            transport: fetchMock,
            request: ["/site-info"],
        },
        {
            name: "getStaff reads the staff list",
            call: () => api.getStaff(),
            transport: fetchMock,
            request: ["/staff"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("email verification and password recovery", () => {
    const cases: RequestCase[] = [
        {
            name: "setEmail posts the new address alongside the current password",
            call: () => api.setEmail("kujo@example.com", "pw"),
            transport: postMock,
            request: ["/auth/set-email", { email: "kujo@example.com", password: "pw" }],
        },
        {
            name: "verifyEmail posts the token from the email",
            call: () => api.verifyEmail("verify-1"),
            transport: postMock,
            request: ["/auth/verify-email", { token: "verify-1" }],
        },
        {
            name: "resendVerification posts with no body",
            call: () => api.resendVerification(),
            transport: postMock,
            request: ["/auth/resend-verification", undefined],
        },
        {
            name: "forgotPassword leaves the turnstile token unset when it is not supplied",
            call: () => api.forgotPassword("kujo"),
            transport: postMock,
            request: ["/auth/forgot-password", { username: "kujo", turnstile_token: undefined }],
        },
        {
            name: "forgotPassword forwards the turnstile token",
            call: () => api.forgotPassword("kujo", "token-1"),
            transport: postMock,
            request: ["/auth/forgot-password", { username: "kujo", turnstile_token: "token-1" }],
        },
        {
            name: "resetPassword posts the token with the snake cased new password",
            call: () => api.resetPassword("reset-1", "new-pw"),
            transport: postMock,
            request: ["/auth/reset-password", { token: "reset-1", new_password: "new-pw" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("push notification devices", () => {
    const cases: RequestCase[] = [
        {
            name: "registerDeviceToken posts the token and its platform",
            call: () => api.registerDeviceToken("device-1", "android"),
            transport: postMock,
            request: ["/push/device", { token: "device-1", platform: "android" }],
        },
        {
            name: "unregisterDeviceToken deletes with the token in the body",
            call: () => api.unregisterDeviceToken("device-1"),
            transport: deleteWithBodyMock,
            request: ["/push/device", { token: "device-1" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the theory API", () => {
    const theoryPayload = {
        title: "The gold is real",
        body: "Beatrice hid it beneath the rose garden",
        episode: 4,
        series: "umineko",
        evidence: [{ note: "the letter", quote_index: 2 }],
    };
    const responsePayload = {
        side: "with_love" as const,
        body: "Then the seventh twilight is a trap",
        evidence: [],
    };
    const replyPayload = { ...responsePayload, parent_id: "resp-0" };

    const cases: RequestCase[] = [
        {
            name: "createTheory posts the whole payload to the theory collection",
            call: () => api.createTheory(theoryPayload),
            transport: postMock,
            request: ["/theories", theoryPayload],
        },
        {
            name: "getTheory reads a single theory",
            call: () => api.getTheory("t-1"),
            transport: fetchMock,
            request: ["/theories/t-1"],
        },
        {
            name: "updateTheory puts the whole payload back",
            call: () => api.updateTheory("t-1", theoryPayload),
            transport: putMock,
            request: ["/theories/t-1", theoryPayload],
        },
        {
            name: "deleteTheory deletes the theory",
            call: () => api.deleteTheory("t-1"),
            transport: deleteMock,
            request: ["/theories/t-1"],
        },
        {
            name: "createResponse nests a top level response under its theory",
            call: () => api.createResponse("t-1", responsePayload),
            transport: postMock,
            request: ["/theories/t-1/responses", responsePayload],
        },
        {
            name: "createResponse carries the parent id of a reply",
            call: () => api.createResponse("t-1", replyPayload),
            transport: postMock,
            request: ["/theories/t-1/responses", replyPayload],
        },
        {
            name: "deleteResponse deletes from the response collection",
            call: () => api.deleteResponse("resp-1"),
            transport: deleteMock,
            request: ["/responses/resp-1"],
        },
        {
            name: "voteTheory posts an upvote",
            call: () => api.voteTheory("t-1", 1),
            transport: postMock,
            request: ["/theories/t-1/vote", { value: 1 }],
        },
        {
            name: "voteResponse posts a downvote",
            call: () => api.voteResponse("resp-1", -1),
            transport: postMock,
            request: ["/responses/resp-1/vote", { value: -1 }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("profile and preference updates", () => {
    const profilePayload = {
        display_name: "Victorique",
        bio: "the golden witch of the parlour",
        avatar_url: "/media/avatar.png",
        banner_url: "/media/banner.png",
        banner_position: 50,
        favourite_character: "Beatrice",
        gender: "female",
        pronoun_subject: "she",
        pronoun_possessive: "her",
        social_twitter: "",
        social_discord: "",
        social_waifulist: "",
        social_tumblr: "",
        social_github: "",
        social_bluesky: "",
        website: "",
        dms_enabled: true,
        episode_progress: 8,
        higurashi_arc_progress: 0,
        ciconia_chapter_progress: 0,
        dob: "1986-06-06",
        dob_public: false,
        email: "victorique@example.com",
        email_public: false,
        email_notifications: true,
        play_message_sound: true,
        play_notification_sound: false,
        home_page: "home",
        game_board_sort: "recent",
        default_profile_tab: "posts",
    };

    const cases: RequestCase[] = [
        {
            name: "getUserProfile reads a profile by username",
            call: () => api.getUserProfile("kujo"),
            transport: fetchMock,
            request: ["/users/kujo"],
        },
        {
            name: "updateProfile puts the whole payload to the auth profile",
            call: () => api.updateProfile(profilePayload),
            transport: putMock,
            request: ["/auth/profile", profilePayload],
        },
        {
            name: "updateGameBoardSort puts the chosen sort",
            call: () => api.updateGameBoardSort("recent"),
            transport: putMock,
            request: ["/preferences/game-board-sort", { sort: "recent" }],
        },
        {
            name: "updateAppearance puts the theme, font and the snake cased wide layout flag",
            call: () => api.updateAppearance("gold", "serif", true),
            transport: putMock,
            request: ["/preferences/appearance", { theme: "gold", font: "serif", wide_layout: true }],
        },
        {
            name: "updateAppearance can switch the wide layout back off",
            call: () => api.updateAppearance("gold", "serif", false),
            transport: putMock,
            request: ["/preferences/appearance", { theme: "gold", font: "serif", wide_layout: false }],
        },
        {
            name: "unlockSecret puts the secret and the guessed phrase",
            call: () => api.unlockSecret("golden-land", "without love it cannot be seen"),
            transport: putMock,
            request: [
                "/preferences/secret-unlock",
                { secret: "golden-land", phrase: "without love it cannot be seen" },
            ],
        },
        {
            name: "changePassword puts the old and new passwords",
            call: () => api.changePassword({ old_password: "old-pw", new_password: "new-pw" }),
            transport: putMock,
            request: ["/auth/password", { old_password: "old-pw", new_password: "new-pw" }],
        },
        {
            name: "deleteAccount deletes with the password in the body",
            call: () => api.deleteAccount({ password: "pw" }),
            transport: deleteWithBodyMock,
            request: ["/auth/account", { password: "pw" }],
        },
        {
            name: "getUserActivity defaults to the first page of twenty",
            call: () => api.getUserActivity("kujo"),
            transport: fetchMock,
            request: ["/users/kujo/activity?limit=20"],
        },
        {
            name: "getUserActivity pages through the activity feed",
            call: () => api.getUserActivity("kujo", 5, 10),
            transport: fetchMock,
            request: ["/users/kujo/activity?limit=5&offset=10"],
        },
        {
            name: "getUserActivity drops an offset of zero",
            call: () => api.getUserActivity("kujo", 5, 0),
            transport: fetchMock,
            request: ["/users/kujo/activity?limit=5"],
        },
        {
            name: "getUserActivity sends an explicit limit of zero",
            call: () => api.getUserActivity("kujo", 0),
            transport: fetchMock,
            request: ["/users/kujo/activity?limit=0"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the notification API", () => {
    const cases: RequestCase[] = [
        {
            name: "getNotifications defaults to the first page of twenty",
            call: () => api.getNotifications({}),
            transport: fetchMock,
            request: ["/notifications?limit=20"],
        },
        {
            name: "getNotifications pages through the list",
            call: () => api.getNotifications({ limit: 5, offset: 10 }),
            transport: fetchMock,
            request: ["/notifications?limit=5&offset=10"],
        },
        {
            name: "getNotifications drops an offset of zero",
            call: () => api.getNotifications({ limit: 5, offset: 0 }),
            transport: fetchMock,
            request: ["/notifications?limit=5"],
        },
        {
            name: "getNotifications sends an explicit limit of zero",
            call: () => api.getNotifications({ limit: 0 }),
            transport: fetchMock,
            request: ["/notifications?limit=0"],
        },
        {
            name: "markNotificationRead posts to the numbered notification with no body",
            call: () => api.markNotificationRead(3),
            transport: postMock,
            request: ["/notifications/3/read", undefined],
        },
        {
            name: "markAllNotificationsRead posts to the collection with no body",
            call: () => api.markAllNotificationsRead(),
            transport: postMock,
            request: ["/notifications/read", undefined],
        },
        {
            name: "getUnreadCount reads the unread counter",
            call: () => api.getUnreadCount(),
            transport: fetchMock,
            request: ["/notifications/unread-count"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
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

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("direct messages and group chat rooms", () => {
    const groupPayload = {
        name: "Rokkenjima",
        description: "the family conference",
        is_public: true,
        is_rp: false,
        tags: ["umineko"],
        member_ids: ["u-1", "u-2"],
    };

    const cases: RequestCase[] = [
        {
            name: "resolveDMRoom reads the conversation with a recipient",
            call: () => api.resolveDMRoom("u-1"),
            transport: fetchMock,
            request: ["/chat/dm/u-1/resolve"],
        },
        {
            name: "createGroupRoom posts the whole payload to the room collection",
            call: () => api.createGroupRoom(groupPayload),
            transport: postMock,
            request: ["/chat/rooms", groupPayload],
        },
        {
            name: "leaveChatRoom posts an empty body",
            call: () => api.leaveChatRoom("r-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/leave", {}],
        },
        {
            name: "setChatRoomMuted puts the muted flag",
            call: () => api.setChatRoomMuted("r-1", true),
            transport: putMock,
            request: ["/chat/rooms/r-1/mute", { muted: true }],
        },
        {
            name: "setChatRoomMuted can unmute the room again",
            call: () => api.setChatRoomMuted("r-1", false),
            transport: putMock,
            request: ["/chat/rooms/r-1/mute", { muted: false }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("profile and admin uploads", () => {
    it("uploadBanner posts the file under the banner field", async () => {
        // given
        const file = new File(["x"], "banner.png", { type: "image/png" });

        // when
        await api.uploadBanner(file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/auth/banner");
        expect(postFormDataMock.mock.calls[0][1].get("banner")).toBe(file);
    });

    it("uploadOGDefaultImage posts the file under the image field", async () => {
        // given
        const file = new File(["x"], "og.png", { type: "image/png" });

        // when
        await api.uploadOGDefaultImage(file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/admin/settings/og-image");
        expect(postFormDataMock.mock.calls[0][1].get("image")).toBe(file);
    });

    it("sendFirstDMMessage attaches the body and every media file", async () => {
        // given
        const first = new File(["a"], "a.png", { type: "image/png" });
        const second = new File(["b"], "b.png", { type: "image/png" });

        // when
        await api.sendFirstDMMessage("u-1", "good evening", [first, second]);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/chat/dm/u-1/messages");
        expect(formData.get("body")).toBe("good evening");
        expect(formData.getAll("media")).toEqual([first, second]);
    });

    it("sendFirstDMMessage omits the media when no files are given", async () => {
        // given
        const recipientId = "u-1";

        // when
        await api.sendFirstDMMessage(recipientId, "good evening");

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(formData.get("body")).toBe("good evening");
        expect(formData.has("media")).toBe(false);
    });
});

describe("chat room membership and bans", () => {
    const cases: RequestCase[] = [
        {
            name: "getChatRoomMembers reads the member list of a room",
            call: () => api.getChatRoomMembers("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/members"],
        },
        {
            name: "kickChatRoomMember deletes the named member",
            call: () => api.kickChatRoomMember("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/members/u-1"],
        },
        {
            name: "banChatRoomMember posts the reason to the ban collection",
            call: () => api.banChatRoomMember("r-1", "u-1", "spoiling the seventh twilight"),
            transport: postMock,
            request: ["/chat/rooms/r-1/bans/u-1", { reason: "spoiling the seventh twilight" }],
        },
        {
            name: "unbanChatRoomMember deletes the ban",
            call: () => api.unbanChatRoomMember("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/bans/u-1"],
        },
        {
            name: "listChatRoomBans reads the ban list of a room",
            call: () => api.listChatRoomBans("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/bans"],
        },
        {
            name: "inviteChatRoomMembers posts the ids under the snake cased field",
            call: () => api.inviteChatRoomMembers("r-1", ["u-1", "u-2"]),
            transport: postMock,
            request: ["/chat/rooms/r-1/members", { user_ids: ["u-1", "u-2"] }],
        },
        {
            name: "inviteChatRoomMembers still sends an empty list when nobody was chosen",
            call: () => api.inviteChatRoomMembers("r-1", []),
            transport: postMock,
            request: ["/chat/rooms/r-1/members", { user_ids: [] }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("banned word rules", () => {
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
            name: "listChatRoomBannedWords reads the rules of a room",
            call: () => api.listChatRoomBannedWords("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/banned-words"],
        },
        {
            name: "createChatRoomBannedWord posts the whole rule to the room",
            call: () => api.createChatRoomBannedWord("r-1", substringRule),
            transport: postMock,
            request: ["/chat/rooms/r-1/banned-words", substringRule],
        },
        {
            name: "updateChatRoomBannedWord puts the rule back under its own id",
            call: () => api.updateChatRoomBannedWord("r-1", "rule-1", regexRule),
            transport: putMock,
            request: ["/chat/rooms/r-1/banned-words/rule-1", regexRule],
        },
        {
            name: "deleteChatRoomBannedWord deletes the rule from the room",
            call: () => api.deleteChatRoomBannedWord("r-1", "rule-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/banned-words/rule-1"],
        },
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

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("room voice channels", () => {
    const cases: RequestCase[] = [
        {
            name: "getVoiceToken posts an empty body to the room voice token",
            call: () => api.getVoiceToken("r-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/voice/token", {}],
        },
        {
            name: "forceMuteVoiceParticipant mutes the named participant",
            call: () => api.forceMuteVoiceParticipant("r-1", "u-1", true),
            transport: postMock,
            request: ["/chat/rooms/r-1/voice/participants/u-1/mute", { muted: true }],
        },
        {
            name: "forceMuteVoiceParticipant can unmute the participant again",
            call: () => api.forceMuteVoiceParticipant("r-1", "u-1", false),
            transport: postMock,
            request: ["/chat/rooms/r-1/voice/participants/u-1/mute", { muted: false }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the live stream API", () => {
    const cases: RequestCase[] = [
        {
            name: "listLiveStreams reads the live stream list",
            call: () => api.listLiveStreams(),
            transport: fetchMock,
            request: ["/streams/live"],
        },
        {
            name: "getStream reads a single stream",
            call: () => api.getStream("s-1"),
            transport: fetchMock,
            request: ["/streams/s-1"],
        },
        {
            name: "getMyStream reads the caller's own stream",
            call: () => api.getMyStream(),
            transport: fetchMock,
            request: ["/streams/mine"],
        },
        {
            name: "getStreamCredentials reads the ingest credentials",
            call: () => api.getStreamCredentials(),
            transport: fetchMock,
            request: ["/streams/credentials"],
        },
        {
            name: "resetStreamCredentials posts an empty body to the credential reset",
            call: () => api.resetStreamCredentials(),
            transport: postMock,
            request: ["/streams/credentials/reset", {}],
        },
        {
            name: "getOverlayConnection reads the overlay token",
            call: () => api.getOverlayConnection(),
            transport: fetchMock,
            request: ["/overlay/token"],
        },
        {
            name: "resetOverlayToken posts an empty body to the token reset",
            call: () => api.resetOverlayToken(),
            transport: postMock,
            request: ["/overlay/token/reset", {}],
        },
        {
            name: "testOverlay posts an empty body to the overlay test",
            call: () => api.testOverlay(),
            transport: postMock,
            request: ["/overlay/test", {}],
        },
        {
            name: "startStream posts the title, the default mode and the bitrate",
            call: () => api.startStream("the golden feast", "webrtc", 4500),
            transport: postMock,
            request: ["/streams", { title: "the golden feast", defaultMode: "webrtc", bitrate: 4500 }],
        },
        {
            name: "startStream can favour hls as the default mode",
            call: () => api.startStream("the golden feast", "hls", 6000),
            transport: postMock,
            request: ["/streams", { title: "the golden feast", defaultMode: "hls", bitrate: 6000 }],
        },
        {
            name: "stopStream deletes the stream",
            call: () => api.stopStream("s-1"),
            transport: deleteMock,
            request: ["/streams/s-1"],
        },
        {
            name: "updateStreamTitle patches only the title",
            call: () => api.updateStreamTitle("s-1", "the seventh twilight"),
            transport: patchMock,
            request: ["/streams/s-1", { title: "the seventh twilight" }],
        },
        {
            name: "getStreamViewerToken posts an empty body for a viewer token",
            call: () => api.getStreamViewerToken("s-1"),
            transport: postMock,
            request: ["/streams/s-1/token", {}],
        },
        {
            name: "joinStreamChat posts an empty body to the join chat endpoint",
            call: () => api.joinStreamChat("s-1"),
            transport: postMock,
            request: ["/streams/s-1/join-chat", {}],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("watch party sessions", () => {
    const fullOptions = {
        start_url: "https://example.com/video",
        region: "EU",
        title: "the golden feast",
        type: "hyperbeam" as const,
    };

    const cases: RequestCase[] = [
        {
            name: "listWatchParties reads the sessions of a room",
            call: () => api.listWatchParties("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/watch-parties"],
        },
        {
            name: "startWatchParty posts an empty options object when nothing was chosen",
            call: () => api.startWatchParty("r-1", {}),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties", {}],
        },
        {
            name: "startWatchParty forwards the start url, region, title and type",
            call: () => api.startWatchParty("r-1", fullOptions),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties", fullOptions],
        },
        {
            name: "startWatchParty can start a screenshare with only a type",
            call: () => api.startWatchParty("r-1", { type: "screenshare" }),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties", { type: "screenshare" }],
        },
        {
            name: "joinWatchParty posts an empty body to the session join",
            call: () => api.joinWatchParty("r-1", "s-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/join", {}],
        },
        {
            name: "endWatchParty deletes the whole session",
            call: () => api.endWatchParty("r-1", "s-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1"],
        },
        {
            name: "identifyWatchPartyParticipant posts the identifier",
            call: () => api.identifyWatchPartyParticipant("r-1", "s-1", "peer-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/watch-parties/s-1/identify", { identifier: "peer-1" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("chat room reading and housekeeping", () => {
    const cases: RequestCase[] = [
        {
            name: "getUserRooms reads the caller's rooms",
            call: () => api.getUserRooms(),
            transport: fetchMock,
            request: ["/chat/rooms"],
        },
        {
            name: "getRoomMessages defaults to fifty messages and no offset",
            call: () => api.getRoomMessages("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/messages?limit=50"],
        },
        {
            name: "getRoomMessages pages through the history",
            call: () => api.getRoomMessages("r-1", 25, 50),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/messages?limit=25&offset=50"],
        },
        {
            name: "getRoomMessages drops an offset of zero",
            call: () => api.getRoomMessages("r-1", 25, 0),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/messages?limit=25"],
        },
        {
            name: "deleteChatRoom deletes the room",
            call: () => api.deleteChatRoom("r-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1"],
        },
        {
            name: "getChatUnreadCount reads the unread counter",
            call: () => api.getChatUnreadCount(),
            transport: fetchMock,
            request: ["/chat/unread-count"],
        },
        {
            name: "markChatRoomRead posts an empty body to the read marker",
            call: () => api.markChatRoomRead("r-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/read", {}],
        },
        {
            name: "updateChatRoomNickname puts the caller's own nickname",
            call: () => api.updateChatRoomNickname("r-1", "Victorique"),
            transport: putMock,
            request: ["/chat/rooms/r-1/me", { nickname: "Victorique" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("chat room member nicknames, timeouts and avatars", () => {
    const cases: RequestCase[] = [
        {
            name: "setChatRoomMemberNickname puts the nickname on the member",
            call: () => api.setChatRoomMemberNickname("r-1", "u-1", "Beatrice"),
            transport: putMock,
            request: ["/chat/rooms/r-1/members/u-1/nickname", { nickname: "Beatrice" }],
        },
        {
            name: "unlockChatRoomMemberNickname deletes the member's nickname lock",
            call: () => api.unlockChatRoomMemberNickname("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/members/u-1/nickname"],
        },
        {
            name: "clearChatRoomMemberTimeout deletes the member's timeout",
            call: () => api.clearChatRoomMemberTimeout("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/members/u-1/timeout"],
        },
        {
            name: "clearChatRoomAvatar deletes the caller's own room avatar",
            call: () => api.clearChatRoomAvatar("r-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/me/avatar"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadChatRoomAvatar posts the file under the avatar field of the caller's membership", async () => {
        // given
        const file = new File(["x"], "room-avatar.png", { type: "image/png" });

        // when
        await api.uploadChatRoomAvatar("r-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/chat/rooms/r-1/me/avatar");
        expect(postFormDataMock.mock.calls[0][1].get("avatar")).toBe(file);
    });
});

describe("chat message editing, pinning and reactions", () => {
    const cases: RequestCase[] = [
        {
            name: "deleteChatMessage deletes the message",
            call: () => api.deleteChatMessage("m-1"),
            transport: deleteMock,
            request: ["/chat/messages/m-1"],
        },
        {
            name: "editChatMessage patches only the body",
            call: () => api.editChatMessage("m-1", "without love it cannot be seen"),
            transport: patchMock,
            request: ["/chat/messages/m-1", { body: "without love it cannot be seen" }],
        },
        {
            name: "pinChatMessage posts an empty body to the pin",
            call: () => api.pinChatMessage("m-1"),
            transport: postMock,
            request: ["/chat/messages/m-1/pin", {}],
        },
        {
            name: "unpinChatMessage deletes the pin",
            call: () => api.unpinChatMessage("m-1"),
            transport: deleteMock,
            request: ["/chat/messages/m-1/pin"],
        },
        {
            name: "getChatRoomPinnedMessages reads the pins of a room",
            call: () => api.getChatRoomPinnedMessages("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/pins"],
        },
        {
            name: "addChatMessageReaction posts the emoji to the reaction collection",
            call: () => api.addChatMessageReaction("m-1", "🍬"),
            transport: postMock,
            request: ["/chat/messages/m-1/reactions", { emoji: "🍬" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("report resolution and rule pages", () => {
    const cases: RequestCase[] = [
        {
            name: "resolveReport posts the moderator's comment to the numbered report",
            call: () => api.resolveReport(3, "handled in the parlour"),
            transport: postMock,
            request: ["/admin/reports/3/resolve", { comment: "handled in the parlour" }],
        },
        {
            name: "resolveReport still sends an empty comment",
            call: () => api.resolveReport(3, ""),
            transport: postMock,
            request: ["/admin/reports/3/resolve", { comment: "" }],
        },
        {
            name: "getRules reads the named rules page",
            call: () => api.getRules("chat"),
            transport: fetchMock,
            request: ["/rules/chat"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the post API", () => {
    const cases: RequestCase[] = [
        {
            name: "getCornerCounts reads the corner counters",
            call: () => api.getCornerCounts(),
            transport: fetchMock,
            request: ["/posts/corner-counts"],
        },
        {
            name: "getPost reads a single post",
            call: () => api.getPost("p-1"),
            transport: fetchMock,
            request: ["/posts/p-1"],
        },
        {
            name: "updatePost puts only the body",
            call: () => api.updatePost("p-1", "the gold is real"),
            transport: putMock,
            request: ["/posts/p-1", { body: "the gold is real" }],
        },
        {
            name: "votePoll posts the snake cased option id",
            call: () => api.votePoll("p-1", 2),
            transport: postMock,
            request: ["/posts/p-1/poll/vote", { option_id: 2 }],
        },
        {
            name: "unresolveSuggestion deletes the resolution",
            call: () => api.unresolveSuggestion("p-1"),
            transport: deleteMock,
            request: ["/posts/p-1/resolve"],
        },
        {
            name: "deletePost deletes the post",
            call: () => api.deletePost("p-1"),
            transport: deleteMock,
            request: ["/posts/p-1"],
        },
        {
            name: "deletePostMedia interpolates a numeric media id",
            call: () => api.deletePostMedia("p-1", 4),
            transport: deleteMock,
            request: ["/posts/p-1/media/4"],
        },
        {
            name: "likePost posts with no body",
            call: () => api.likePost("p-1"),
            transport: postMock,
            request: ["/posts/p-1/like", undefined],
        },
        {
            name: "unlikePost deletes the like",
            call: () => api.unlikePost("p-1"),
            transport: deleteMock,
            request: ["/posts/p-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the comment API", () => {
    const cases: RequestCase[] = [
        {
            name: "createComment leaves the parent unset for a top level comment",
            call: () => api.createComment("p-1", "lovely"),
            transport: postMock,
            request: ["/posts/p-1/comments", { body: "lovely", parent_id: undefined }],
        },
        {
            name: "createComment carries the parent id of a reply",
            call: () => api.createComment("p-1", "lovely", "c-0"),
            transport: postMock,
            request: ["/posts/p-1/comments", { body: "lovely", parent_id: "c-0" }],
        },
        {
            name: "updateComment puts only the body",
            call: () => api.updateComment("c-1", "on second thoughts"),
            transport: putMock,
            request: ["/comments/c-1", { body: "on second thoughts" }],
        },
        {
            name: "deleteComment deletes the comment",
            call: () => api.deleteComment("c-1"),
            transport: deleteMock,
            request: ["/comments/c-1"],
        },
        {
            name: "likeComment posts with no body",
            call: () => api.likeComment("c-1"),
            transport: postMock,
            request: ["/comments/c-1/like", undefined],
        },
        {
            name: "unlikeComment deletes the like",
            call: () => api.unlikeComment("c-1"),
            transport: deleteMock,
            request: ["/comments/c-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("post and comment media uploads", () => {
    it("uploadPostMedia posts the file under the media field of the post", async () => {
        // given
        const file = new File(["x"], "post.png", { type: "image/png" });

        // when
        await api.uploadPostMedia("p-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/posts/p-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
    });

    it("uploadCommentMedia posts the file under the media field of the comment", async () => {
        // given
        const file = new File(["x"], "comment.png", { type: "image/png" });

        // when
        await api.uploadCommentMedia("c-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/comments/c-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
    });
});

describe("user posts, follows and the public directory", () => {
    const cases: RequestCase[] = [
        {
            name: "getUserPosts defaults to the first page of twenty",
            call: () => api.getUserPosts("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/posts?limit=20"],
        },
        {
            name: "getUserPosts pages through the posts",
            call: () => api.getUserPosts("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/posts?limit=5&offset=10"],
        },
        {
            name: "getUserPosts drops an offset of zero",
            call: () => api.getUserPosts("u-1", 5, 0),
            transport: fetchMock,
            request: ["/users/u-1/posts?limit=5"],
        },
        {
            name: "followUser posts with no body",
            call: () => api.followUser("u-1"),
            transport: postMock,
            request: ["/users/u-1/follow", undefined],
        },
        {
            name: "unfollowUser deletes the follow",
            call: () => api.unfollowUser("u-1"),
            transport: deleteMock,
            request: ["/users/u-1/follow"],
        },
        {
            name: "getFollowStats reads the follower counters",
            call: () => api.getFollowStats("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/follow-stats"],
        },
        {
            name: "getFollowers defaults to the first page of fifty",
            call: () => api.getFollowers("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/followers?limit=50"],
        },
        {
            name: "getFollowers pages through the followers",
            call: () => api.getFollowers("u-1", 10, 20),
            transport: fetchMock,
            request: ["/users/u-1/followers?limit=10&offset=20"],
        },
        {
            name: "getFollowing defaults to the first page of fifty",
            call: () => api.getFollowing("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/following?limit=50"],
        },
        {
            name: "getFollowing pages through the followed users",
            call: () => api.getFollowing("u-1", 10, 20),
            transport: fetchMock,
            request: ["/users/u-1/following?limit=10&offset=20"],
        },
        {
            name: "getMutualFollowers reads the mutuals list",
            call: () => api.getMutualFollowers(),
            transport: fetchMock,
            request: ["/users/mutuals"],
        },
        {
            name: "listUsersPublic reads the public user directory",
            call: () => api.listUsersPublic(),
            transport: fetchMock,
            request: ["/users"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the art API", () => {
    const artPayload = {
        title: "Golden Witch",
        description: "the endless witch in her rose garden",
        tags: ["beatrice", "umineko"],
        is_spoiler: true,
    };

    const cases: RequestCase[] = [
        {
            name: "listArt sends no query string when nothing is filtered",
            call: () => api.listArt({}),
            transport: fetchMock,
            request: ["/art"],
        },
        {
            name: "listArt passes every filter through",
            call: () =>
                api.listArt({
                    corner: "umineko",
                    type: "digital",
                    search: "golden witch",
                    tag: "beatrice",
                    sort: "top",
                    limit: 10,
                    offset: 20,
                }),
            transport: fetchMock,
            request: ["/art?corner=umineko&type=digital&search=golden+witch&tag=beatrice&sort=top&limit=10&offset=20"],
        },
        {
            name: "listArt drops an offset of zero",
            call: () => api.listArt({ corner: "umineko", offset: 0 }),
            transport: fetchMock,
            request: ["/art?corner=umineko"],
        },
        {
            name: "getArt reads a single piece",
            call: () => api.getArt("a-1"),
            transport: fetchMock,
            request: ["/art/a-1"],
        },
        {
            name: "updateArt puts the whole payload back",
            call: () => api.updateArt("a-1", artPayload),
            transport: putMock,
            request: ["/art/a-1", artPayload],
        },
        {
            name: "deleteArt deletes the piece",
            call: () => api.deleteArt("a-1"),
            transport: deleteMock,
            request: ["/art/a-1"],
        },
        {
            name: "likeArt posts with no body",
            call: () => api.likeArt("a-1"),
            transport: postMock,
            request: ["/art/a-1/like", undefined],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("art likes, corner counts and art comments", () => {
    const cases: RequestCase[] = [
        {
            name: "unlikeArt deletes the like",
            call: () => api.unlikeArt("a-1"),
            transport: deleteMock,
            request: ["/art/a-1/like"],
        },
        {
            name: "getArtCornerCounts reads the art corner counters",
            call: () => api.getArtCornerCounts(),
            transport: fetchMock,
            request: ["/art/corner-counts"],
        },
        {
            name: "createArtComment leaves the parent unset for a top level comment",
            call: () => api.createArtComment("a-1", "what a lovely rose garden"),
            transport: postMock,
            request: ["/art/a-1/comments", { body: "what a lovely rose garden", parent_id: undefined }],
        },
        {
            name: "createArtComment carries the parent id of a reply",
            call: () => api.createArtComment("a-1", "quite so", "ac-0"),
            transport: postMock,
            request: ["/art/a-1/comments", { body: "quite so", parent_id: "ac-0" }],
        },
        {
            name: "updateArtComment puts only the body to the art comment collection",
            call: () => api.updateArtComment("ac-1", "on second thoughts"),
            transport: putMock,
            request: ["/art-comments/ac-1", { body: "on second thoughts" }],
        },
        {
            name: "deleteArtComment deletes the art comment",
            call: () => api.deleteArtComment("ac-1"),
            transport: deleteMock,
            request: ["/art-comments/ac-1"],
        },
        {
            name: "likeArtComment posts with no body",
            call: () => api.likeArtComment("ac-1"),
            transport: postMock,
            request: ["/art-comments/ac-1/like", undefined],
        },
        {
            name: "unlikeArtComment deletes the like",
            call: () => api.unlikeArtComment("ac-1"),
            transport: deleteMock,
            request: ["/art-comments/ac-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadArtCommentMedia posts the file under the media field of the art comment", async () => {
        // given
        const file = new File(["x"], "art-comment.png", { type: "image/png" });

        // when
        await api.uploadArtCommentMedia("ac-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/art-comments/ac-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
    });
});

describe("the gallery API", () => {
    const cases: RequestCase[] = [
        {
            name: "updateGallery defaults the description to an empty string",
            call: () => api.updateGallery("g-1", "Witches"),
            transport: putMock,
            request: ["/galleries/g-1", { name: "Witches", description: "" }],
        },
        {
            name: "updateGallery forwards the description when one was written",
            call: () => api.updateGallery("g-1", "Witches", "the endless witch and her furniture"),
            transport: putMock,
            request: ["/galleries/g-1", { name: "Witches", description: "the endless witch and her furniture" }],
        },
        {
            name: "setGalleryCover puts the chosen cover under the snake cased field",
            call: () => api.setGalleryCover("g-1", "a-1"),
            transport: putMock,
            request: ["/galleries/g-1/cover", { cover_art_id: "a-1" }],
        },
        {
            name: "setGalleryCover clears the cover with an explicit null",
            call: () => api.setGalleryCover("g-1", null),
            transport: putMock,
            request: ["/galleries/g-1/cover", { cover_art_id: null }],
        },
        {
            name: "deleteGallery deletes the gallery",
            call: () => api.deleteGallery("g-1"),
            transport: deleteMock,
            request: ["/galleries/g-1"],
        },
        {
            name: "getGallery defaults to the first page of twenty four",
            call: () => api.getGallery("g-1"),
            transport: fetchMock,
            request: ["/galleries/g-1?limit=24"],
        },
        {
            name: "getGallery pages through the gallery",
            call: () => api.getGallery("g-1", 10, 20),
            transport: fetchMock,
            request: ["/galleries/g-1?limit=10&offset=20"],
        },
        {
            name: "getGallery drops an offset of zero",
            call: () => api.getGallery("g-1", 10, 0),
            transport: fetchMock,
            request: ["/galleries/g-1?limit=10"],
        },
        {
            name: "listAllGalleries sends no query string without a corner",
            call: () => api.listAllGalleries(),
            transport: fetchMock,
            request: ["/galleries"],
        },
        {
            name: "listAllGalleries encodes the corner",
            call: () => api.listAllGalleries("umineko & co"),
            transport: fetchMock,
            request: ["/galleries?corner=umineko%20%26%20co"],
        },
        {
            name: "getUserGalleries reads the galleries of a user",
            call: () => api.getUserGalleries("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/galleries"],
        },
        {
            name: "setArtGallery puts the chosen gallery under the snake cased field",
            call: () => api.setArtGallery("a-1", "g-1"),
            transport: putMock,
            request: ["/art/a-1/gallery", { gallery_id: "g-1" }],
        },
        {
            name: "setArtGallery removes the piece from its gallery with an explicit null",
            call: () => api.setArtGallery("a-1", null),
            transport: putMock,
            request: ["/art/a-1/gallery", { gallery_id: null }],
        },
        {
            name: "getUserArt defaults to the first page of twenty four",
            call: () => api.getUserArt("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/art?limit=24"],
        },
        {
            name: "getUserArt pages through the art of a user",
            call: () => api.getUserArt("u-1", 10, 20),
            transport: fetchMock,
            request: ["/users/u-1/art?limit=10&offset=20"],
        },
        {
            name: "getUserArt drops an offset of zero",
            call: () => api.getUserArt("u-1", 10, 0),
            transport: fetchMock,
            request: ["/users/u-1/art?limit=10"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("blocking other users", () => {
    const cases: RequestCase[] = [
        {
            name: "blockUser posts with no body",
            call: () => api.blockUser("u-1"),
            transport: postMock,
            request: ["/users/u-1/block", undefined],
        },
        {
            name: "unblockUser deletes the block",
            call: () => api.unblockUser("u-1"),
            transport: deleteMock,
            request: ["/users/u-1/block"],
        },
        {
            name: "getBlockStatus reads the block status of a user",
            call: () => api.getBlockStatus("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/block-status"],
        },
        {
            name: "getBlockedUsers reads the caller's own block list",
            call: () => api.getBlockedUsers(),
            transport: fetchMock,
            request: ["/blocked-users"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the announcement API", () => {
    const cases: RequestCase[] = [
        {
            name: "listAnnouncements defaults to the first page of twenty",
            call: () => api.listAnnouncements(),
            transport: fetchMock,
            request: ["/announcements?limit=20"],
        },
        {
            name: "listAnnouncements pages through the announcements",
            call: () => api.listAnnouncements(5, 10),
            transport: fetchMock,
            request: ["/announcements?limit=5&offset=10"],
        },
        {
            name: "listAnnouncements drops an offset of zero",
            call: () => api.listAnnouncements(5, 0),
            transport: fetchMock,
            request: ["/announcements?limit=5"],
        },
        {
            name: "getAnnouncement reads a single announcement",
            call: () => api.getAnnouncement("an-1"),
            transport: fetchMock,
            request: ["/announcements/an-1"],
        },
        {
            name: "getLatestAnnouncement reads the latest announcement from its own path",
            call: () => api.getLatestAnnouncement(),
            transport: fetchMock,
            request: ["/announcements-latest"],
        },
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

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the mystery board API", () => {
    const mysteryPayload = {
        title: "The locked study",
        body: "Kinzo was found behind a bolted door",
        difficulty: "hard",
        free_for_all: false,
        keep_open_after_solve: true,
        clues: [{ body: "the key was still inside", truth_type: "red" }],
    };

    const cases: RequestCase[] = [
        {
            name: "listMysteries defaults to a page of twenty",
            call: () => api.listMysteries({}),
            transport: fetchMock,
            request: ["/mysteries?limit=20"],
        },
        {
            name: "listMysteries passes the sort, the solved filter and the paging through",
            call: () => api.listMysteries({ sort: "top", solved: "false", limit: 5, offset: 10 }),
            transport: fetchMock,
            request: ["/mysteries?sort=top&solved=false&limit=5&offset=10"],
        },
        {
            name: "listMysteries drops an offset of zero",
            call: () => api.listMysteries({ limit: 5, offset: 0 }),
            transport: fetchMock,
            request: ["/mysteries?limit=5"],
        },
        {
            name: "getMystery reads a single mystery",
            call: () => api.getMystery("m-1"),
            transport: fetchMock,
            request: ["/mysteries/m-1"],
        },
        {
            name: "createMystery posts the whole payload to the mystery collection",
            call: () => api.createMystery(mysteryPayload),
            transport: postMock,
            request: ["/mysteries", mysteryPayload],
        },
        {
            name: "updateMystery puts the whole payload back",
            call: () => api.updateMystery("m-1", mysteryPayload),
            transport: putMock,
            request: ["/mysteries/m-1", mysteryPayload],
        },
        {
            name: "deleteMystery deletes the mystery",
            call: () => api.deleteMystery("m-1"),
            transport: deleteMock,
            request: ["/mysteries/m-1"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("mystery attempts, votes and comments", () => {
    const cases: RequestCase[] = [
        {
            name: "createMysteryAttempt leaves the parent unset for a top level attempt",
            call: () => api.createMysteryAttempt("m-1", "the butler did it"),
            transport: postMock,
            request: ["/mysteries/m-1/attempts", { body: "the butler did it", parent_id: undefined }],
        },
        {
            name: "createMysteryAttempt carries the parent id of a reply",
            call: () => api.createMysteryAttempt("m-1", "no, the sister did", "at-0"),
            transport: postMock,
            request: ["/mysteries/m-1/attempts", { body: "no, the sister did", parent_id: "at-0" }],
        },
        {
            name: "deleteMysteryAttempt deletes from the attempt collection",
            call: () => api.deleteMysteryAttempt("at-1"),
            transport: deleteMock,
            request: ["/mystery-attempts/at-1"],
        },
        {
            name: "voteMysteryAttempt posts an upvote",
            call: () => api.voteMysteryAttempt("at-1", 1),
            transport: postMock,
            request: ["/mystery-attempts/at-1/vote", { value: 1 }],
        },
        {
            name: "voteMysteryAttempt posts a downvote",
            call: () => api.voteMysteryAttempt("at-1", -1),
            transport: postMock,
            request: ["/mystery-attempts/at-1/vote", { value: -1 }],
        },
        {
            name: "markMysterySolved posts the winning attempt under the snake cased field",
            call: () => api.markMysterySolved("m-1", "at-1"),
            transport: postMock,
            request: ["/mysteries/m-1/solve", { attempt_id: "at-1" }],
        },
        {
            name: "createMysteryComment leaves the parent unset for a top level comment",
            call: () => api.createMysteryComment("m-1", "a fine puzzle"),
            transport: postMock,
            request: ["/mysteries/m-1/comments", { body: "a fine puzzle", parent_id: undefined }],
        },
        {
            name: "createMysteryComment carries the parent id of a reply",
            call: () => api.createMysteryComment("m-1", "quite so", "mc-0"),
            transport: postMock,
            request: ["/mysteries/m-1/comments", { body: "quite so", parent_id: "mc-0" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("mystery clues and game master controls", () => {
    const cases: RequestCase[] = [
        {
            name: "closeMystery posts an empty body to the close endpoint",
            call: () => api.closeMystery("m-1"),
            transport: postMock,
            request: ["/mysteries/m-1/close", {}],
        },
        {
            name: "setMysteryPaused posts the paused flag",
            call: () => api.setMysteryPaused("m-1", true),
            transport: postMock,
            request: ["/mysteries/m-1/pause", { paused: true }],
        },
        {
            name: "setMysteryPaused can resume the mystery again",
            call: () => api.setMysteryPaused("m-1", false),
            transport: postMock,
            request: ["/mysteries/m-1/pause", { paused: false }],
        },
        {
            name: "setMysteryGmAway posts the away flag",
            call: () => api.setMysteryGmAway("m-1", true),
            transport: postMock,
            request: ["/mysteries/m-1/away", { away: true }],
        },
        {
            name: "setMysteryGmAway can bring the game master back",
            call: () => api.setMysteryGmAway("m-1", false),
            transport: postMock,
            request: ["/mysteries/m-1/away", { away: false }],
        },
        {
            name: "deleteMysteryClue interpolates a numeric clue id",
            call: () => api.deleteMysteryClue("m-1", 4),
            transport: deleteMock,
            request: ["/mysteries/m-1/clues/4"],
        },
        {
            name: "updateMysteryClue puts only the body to the numbered clue",
            call: () => api.updateMysteryClue("m-1", 4, "the key was never inside"),
            transport: putMock,
            request: ["/mysteries/m-1/clues/4", { body: "the key was never inside" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("mystery comment editing and likes", () => {
    const cases: RequestCase[] = [
        {
            name: "updateMysteryComment puts only the body to the mystery comment collection",
            call: () => api.updateMysteryComment("mc-1", "on second thoughts"),
            transport: putMock,
            request: ["/mystery-comments/mc-1", { body: "on second thoughts" }],
        },
        {
            name: "deleteMysteryComment deletes the mystery comment",
            call: () => api.deleteMysteryComment("mc-1"),
            transport: deleteMock,
            request: ["/mystery-comments/mc-1"],
        },
        {
            name: "likeMysteryComment posts an empty body to the like",
            call: () => api.likeMysteryComment("mc-1"),
            transport: postMock,
            request: ["/mystery-comments/mc-1/like", {}],
        },
        {
            name: "unlikeMysteryComment deletes the like",
            call: () => api.unlikeMysteryComment("mc-1"),
            transport: deleteMock,
            request: ["/mystery-comments/mc-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadMysteryCommentMedia posts the file under the media field of the mystery comment", async () => {
        // given
        const file = new File(["x"], "mystery-comment.png", { type: "image/png" });

        // when
        await api.uploadMysteryCommentMedia("mc-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/mystery-comments/mc-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
    });
});

describe("the secret API", () => {
    const cases: RequestCase[] = [
        {
            name: "listSecrets reads the secret collection",
            call: () => api.listSecrets(),
            transport: fetchMock,
            request: ["/secrets"],
        },
        {
            name: "getSecret reads a single secret",
            call: () => api.getSecret("sec-1"),
            transport: fetchMock,
            request: ["/secrets/sec-1"],
        },
        {
            name: "createSecretComment leaves the parent unset for a top level comment",
            call: () => api.createSecretComment("sec-1", "without love it cannot be seen"),
            transport: postMock,
            request: ["/secrets/sec-1/comments", { body: "without love it cannot be seen", parent_id: undefined }],
        },
        {
            name: "createSecretComment carries the parent id of a reply",
            call: () => api.createSecretComment("sec-1", "quite so", "sc-0"),
            transport: postMock,
            request: ["/secrets/sec-1/comments", { body: "quite so", parent_id: "sc-0" }],
        },
        {
            name: "updateSecretComment puts only the body to the secret comment collection",
            call: () => api.updateSecretComment("sc-1", "on second thoughts"),
            transport: putMock,
            request: ["/secret-comments/sc-1", { body: "on second thoughts" }],
        },
        {
            name: "deleteSecretComment deletes the secret comment",
            call: () => api.deleteSecretComment("sc-1"),
            transport: deleteMock,
            request: ["/secret-comments/sc-1"],
        },
        {
            name: "likeSecretComment posts an empty body to the like",
            call: () => api.likeSecretComment("sc-1"),
            transport: postMock,
            request: ["/secret-comments/sc-1/like", {}],
        },
        {
            name: "unlikeSecretComment deletes the like",
            call: () => api.unlikeSecretComment("sc-1"),
            transport: deleteMock,
            request: ["/secret-comments/sc-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadSecretCommentMedia posts the file under the media field of the secret comment", async () => {
        // given
        const file = new File(["x"], "secret-comment.png", { type: "image/png" });

        // when
        await api.uploadSecretCommentMedia("sc-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/secret-comments/sc-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
    });
});

describe("mystery attachments and media", () => {
    const cases: RequestCase[] = [
        {
            name: "deleteMysteryAttachment interpolates a numeric attachment id",
            call: () => api.deleteMysteryAttachment("m-1", 4),
            transport: deleteMock,
            request: ["/mysteries/m-1/attachments/4"],
        },
        {
            name: "deleteMysteryMedia interpolates a numeric media id",
            call: () => api.deleteMysteryMedia("m-1", 7),
            transport: deleteMock,
            request: ["/mysteries/m-1/media/7"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadMysteryAttachment posts the document under the file field of the mystery", async () => {
        // given
        const file = new File(["x"], "floor-plan.pdf", { type: "application/pdf" });

        // when
        await api.uploadMysteryAttachment("m-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/mysteries/m-1/attachments");
        expect(postFormDataMock.mock.calls[0][1].get("file")).toBe(file);
        expect(postFormDataMock.mock.calls[0][1].has("media")).toBe(false);
    });

    it("uploadMysteryMedia posts the image under the media field of the mystery", async () => {
        // given
        const file = new File(["x"], "mystery.png", { type: "image/png" });

        // when
        await api.uploadMysteryMedia("m-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/mysteries/m-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
        expect(postFormDataMock.mock.calls[0][1].has("file")).toBe(false);
    });
});

describe("leaderboards and a user's own boards", () => {
    const cases: RequestCase[] = [
        {
            name: "getMysteryLeaderboard sends no query string without a limit",
            call: () => api.getMysteryLeaderboard(),
            transport: fetchMock,
            request: ["/mysteries/leaderboard"],
        },
        {
            name: "getMysteryLeaderboard forwards the limit",
            call: () => api.getMysteryLeaderboard(10),
            transport: fetchMock,
            request: ["/mysteries/leaderboard?limit=10"],
        },
        {
            name: "getGMLeaderboard sends no query string without a limit",
            call: () => api.getGMLeaderboard(),
            transport: fetchMock,
            request: ["/mysteries/gm-leaderboard"],
        },
        {
            name: "getGMLeaderboard forwards the limit",
            call: () => api.getGMLeaderboard(5),
            transport: fetchMock,
            request: ["/mysteries/gm-leaderboard?limit=5"],
        },
        {
            name: "getUserShips defaults to the first page of twenty",
            call: () => api.getUserShips("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/ships?limit=20"],
        },
        {
            name: "getUserShips pages through the ships of a user",
            call: () => api.getUserShips("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/ships?limit=5&offset=10"],
        },
        {
            name: "getUserShips drops an offset of zero",
            call: () => api.getUserShips("u-1", 5, 0),
            transport: fetchMock,
            request: ["/users/u-1/ships?limit=5"],
        },
        {
            name: "getUserMysteries defaults to the first page of twenty",
            call: () => api.getUserMysteries("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/mysteries?limit=20"],
        },
        {
            name: "getUserMysteries pages through the mysteries of a user",
            call: () => api.getUserMysteries("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/mysteries?limit=5&offset=10"],
        },
        {
            name: "getUserMysteries drops an offset of zero",
            call: () => api.getUserMysteries("u-1", 5, 0),
            transport: fetchMock,
            request: ["/users/u-1/mysteries?limit=5"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the fanfic API", () => {
    const oneshotPayload = {
        title: "The Golden Land",
        summary: "Beatrice pours the tea one last time",
        series: "umineko",
        rating: "teen",
        language: "English",
        is_oneshot: true,
        contains_lemons: false,
        genres: ["mystery"],
        tags: ["beatrice"],
        characters: [{ series: "umineko", character_name: "Beatrice", sort_order: 0 }],
        is_pairing: false,
    };
    const draftedPayload = {
        ...oneshotPayload,
        status: "in_progress",
        body: "The rose garden was still that evening.",
    };
    const updatePayload = {
        ...oneshotPayload,
        status: "complete",
    };

    const cases: RequestCase[] = [
        {
            name: "getFanfic reads a single fanfic",
            call: () => api.getFanfic("f-1"),
            transport: fetchMock,
            request: ["/fanfics/f-1"],
        },
        {
            name: "createFanfic leaves the status and body out when they are not supplied",
            call: () => api.createFanfic(oneshotPayload),
            transport: postMock,
            request: ["/fanfics", oneshotPayload],
        },
        {
            name: "createFanfic forwards the status and the opening body",
            call: () => api.createFanfic(draftedPayload),
            transport: postMock,
            request: ["/fanfics", draftedPayload],
        },
        {
            name: "updateFanfic puts the whole payload back",
            call: () => api.updateFanfic("f-1", updatePayload),
            transport: putMock,
            request: ["/fanfics/f-1", updatePayload],
        },
        {
            name: "deleteFanfic deletes the fanfic",
            call: () => api.deleteFanfic("f-1"),
            transport: deleteMock,
            request: ["/fanfics/f-1"],
        },
        {
            name: "deleteFanficCover deletes the cover image",
            call: () => api.deleteFanficCover("f-1"),
            transport: deleteMock,
            request: ["/fanfics/f-1/cover"],
        },
        {
            name: "createFanficChapter posts the title and body to the chapter collection",
            call: () => api.createFanficChapter("f-1", "The first twilight", "Six chosen by the key"),
            transport: postMock,
            request: ["/fanfics/f-1/chapters", { title: "The first twilight", body: "Six chosen by the key" }],
        },
        {
            name: "updateFanficChapter puts the title and body under the chapter's own id",
            call: () => api.updateFanficChapter("ch-1", "The second twilight", "The two who are close"),
            transport: putMock,
            request: ["/fanfic-chapters/ch-1", { title: "The second twilight", body: "The two who are close" }],
        },
        {
            name: "deleteFanficChapter deletes from the chapter collection",
            call: () => api.deleteFanficChapter("ch-1"),
            transport: deleteMock,
            request: ["/fanfic-chapters/ch-1"],
        },
        {
            name: "favouriteFanfic posts an empty body to the favourite",
            call: () => api.favouriteFanfic("f-1"),
            transport: postMock,
            request: ["/fanfics/f-1/favourite", {}],
        },
        {
            name: "unfavouriteFanfic deletes the favourite",
            call: () => api.unfavouriteFanfic("f-1"),
            transport: deleteMock,
            request: ["/fanfics/f-1/favourite"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadFanficCover posts the file under the image field of the fanfic", async () => {
        // given
        const file = new File(["x"], "cover.png", { type: "image/png" });

        // when
        await api.uploadFanficCover("f-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/fanfics/f-1/cover");
        expect(postFormDataMock.mock.calls[0][1].get("image")).toBe(file);
    });
});

describe("fanfic comments", () => {
    const cases: RequestCase[] = [
        {
            name: "createFanficComment leaves the parent unset for a top level comment",
            call: () => api.createFanficComment("f-1", "what a lovely tea party"),
            transport: postMock,
            request: ["/fanfics/f-1/comments", { body: "what a lovely tea party", parent_id: undefined }],
        },
        {
            name: "createFanficComment carries the parent id of a reply",
            call: () => api.createFanficComment("f-1", "quite so", "fc-0"),
            transport: postMock,
            request: ["/fanfics/f-1/comments", { body: "quite so", parent_id: "fc-0" }],
        },
        {
            name: "updateFanficComment puts only the body to the fanfic comment collection",
            call: () => api.updateFanficComment("fc-1", "on second thoughts"),
            transport: putMock,
            request: ["/fanfic-comments/fc-1", { body: "on second thoughts" }],
        },
        {
            name: "deleteFanficComment deletes the fanfic comment",
            call: () => api.deleteFanficComment("fc-1"),
            transport: deleteMock,
            request: ["/fanfic-comments/fc-1"],
        },
        {
            name: "likeFanficComment posts an empty body to the like",
            call: () => api.likeFanficComment("fc-1"),
            transport: postMock,
            request: ["/fanfic-comments/fc-1/like", {}],
        },
        {
            name: "unlikeFanficComment deletes the like",
            call: () => api.unlikeFanficComment("fc-1"),
            transport: deleteMock,
            request: ["/fanfic-comments/fc-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadFanficCommentMedia posts the file under the media field of the fanfic comment", async () => {
        // given
        const file = new File(["x"], "fanfic-comment.png", { type: "image/png" });

        // when
        await api.uploadFanficCommentMedia("fc-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/fanfic-comments/fc-1/media");
        expect(postFormDataMock.mock.calls[0][1].get("media")).toBe(file);
    });
});

describe("fanfic series and a user's own fanfics", () => {
    const cases: RequestCase[] = [
        {
            name: "getUserFanfics defaults to the first page of twenty",
            call: () => api.getUserFanfics("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/fanfics?limit=20"],
        },
        {
            name: "getUserFanfics pages through the fanfics of a user",
            call: () => api.getUserFanfics("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/fanfics?limit=5&offset=10"],
        },
        {
            name: "getUserFanfics drops an offset of zero",
            call: () => api.getUserFanfics("u-1", 5, 0),
            transport: fetchMock,
            request: ["/users/u-1/fanfics?limit=5"],
        },
        {
            name: "getUserFanficFavourites defaults to the first page of twenty",
            call: () => api.getUserFanficFavourites("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/fanfic-favourites?limit=20"],
        },
        {
            name: "getUserFanficFavourites pages through the favourites of a user",
            call: () => api.getUserFanficFavourites("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/fanfic-favourites?limit=5&offset=10"],
        },
        {
            name: "getUserFanficFavourites drops an offset of zero",
            call: () => api.getUserFanficFavourites("u-1", 5, 0),
            transport: fetchMock,
            request: ["/users/u-1/fanfic-favourites?limit=5"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("getFanficSeries returns just the series list", async () => {
        // given
        fetchMock.mockResolvedValue({ series: ["Umineko", "Higurashi"] });

        // when
        const result = await api.getFanficSeries();

        // then
        expect(fetchMock).toHaveBeenCalledWith("/fanfic-series");
        expect(result).toEqual(["Umineko", "Higurashi"]);
    });
});

describe("announcement comments", () => {
    const cases: RequestCase[] = [
        {
            name: "createAnnouncementComment leaves the parent unset for a top level comment",
            call: () => api.createAnnouncementComment("an-1", "we shall be there"),
            transport: postMock,
            request: ["/announcements/an-1/comments", { body: "we shall be there", parent_id: undefined }],
        },
        {
            name: "createAnnouncementComment carries the parent id of a reply",
            call: () => api.createAnnouncementComment("an-1", "quite so", "anc-0"),
            transport: postMock,
            request: ["/announcements/an-1/comments", { body: "quite so", parent_id: "anc-0" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("announcement comment editing, likes and media", () => {
    const cases: RequestCase[] = [
        {
            name: "updateAnnouncementComment puts the new body under the comment id",
            call: () => api.updateAnnouncementComment("anc-1", "on reflection, no"),
            transport: putMock,
            request: ["/announcement-comments/anc-1", { body: "on reflection, no" }],
        },
        {
            name: "deleteAnnouncementComment deletes the comment",
            call: () => api.deleteAnnouncementComment("anc-1"),
            transport: deleteMock,
            request: ["/announcement-comments/anc-1"],
        },
        {
            name: "likeAnnouncementComment posts an empty body to the like",
            call: () => api.likeAnnouncementComment("anc-1"),
            transport: postMock,
            request: ["/announcement-comments/anc-1/like", {}],
        },
        {
            name: "unlikeAnnouncementComment deletes the like",
            call: () => api.unlikeAnnouncementComment("anc-1"),
            transport: deleteMock,
            request: ["/announcement-comments/anc-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadAnnouncementCommentMedia posts the file under the media field", async () => {
        // given
        const file = new File(["x"], "reply.png", { type: "image/png" });

        // when
        await api.uploadAnnouncementCommentMedia("anc-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/announcement-comments/anc-1/media");
        expect(formData.get("media")).toBe(file);
    });
});

describe("the journal API", () => {
    const journalPayload = { title: "Notes on the epitaph", work: "umineko" as const };
    const entryPayload = { title: "The first twilight", body: "six chosen by the key", is_draft: false };
    const draftEntryPayload = { title: "The second twilight", body: "the two who are close", is_draft: true };

    const cases: RequestCase[] = [
        {
            name: "getJournal reads a single journal",
            call: () => api.getJournal("j-1"),
            transport: fetchMock,
            request: ["/journals/j-1"],
        },
        {
            name: "createJournal posts the whole payload to the journal collection",
            call: () => api.createJournal(journalPayload),
            transport: postMock,
            request: ["/journals", journalPayload],
        },
        {
            name: "updateJournal puts the whole payload back under its own id",
            call: () => api.updateJournal("j-1", journalPayload),
            transport: putMock,
            request: ["/journals/j-1", journalPayload],
        },
        {
            name: "deleteJournal deletes the journal",
            call: () => api.deleteJournal("j-1"),
            transport: deleteMock,
            request: ["/journals/j-1"],
        },
        {
            name: "followJournal posts an empty body to the follow",
            call: () => api.followJournal("j-1"),
            transport: postMock,
            request: ["/journals/j-1/follow", {}],
        },
        {
            name: "unfollowJournal deletes the follow",
            call: () => api.unfollowJournal("j-1"),
            transport: deleteMock,
            request: ["/journals/j-1/follow"],
        },
        {
            name: "getJournalEntry interpolates the numbered entry under its journal",
            call: () => api.getJournalEntry("j-1", 3),
            transport: fetchMock,
            request: ["/journals/j-1/entries/3"],
        },
        {
            name: "createJournalEntry posts the entry to the journal's entry collection",
            call: () => api.createJournalEntry("j-1", entryPayload),
            transport: postMock,
            request: ["/journals/j-1/entries", entryPayload],
        },
        {
            name: "createJournalEntry carries the draft flag",
            call: () => api.createJournalEntry("j-1", draftEntryPayload),
            transport: postMock,
            request: ["/journals/j-1/entries", draftEntryPayload],
        },
        {
            name: "updateJournalEntry puts the entry back on the flat entry path",
            call: () => api.updateJournalEntry("e-1", entryPayload),
            transport: putMock,
            request: ["/journal-entries/e-1", entryPayload],
        },
        {
            name: "deleteJournalEntry deletes the entry",
            call: () => api.deleteJournalEntry("e-1"),
            transport: deleteMock,
            request: ["/journal-entries/e-1"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("journal comment editing and likes", () => {
    const cases: RequestCase[] = [
        {
            name: "updateJournalComment puts the new body under the comment id",
            call: () => api.updateJournalComment("jc-1", "a kinder reading"),
            transport: putMock,
            request: ["/journal-comments/jc-1", { body: "a kinder reading" }],
        },
        {
            name: "deleteJournalComment deletes the comment",
            call: () => api.deleteJournalComment("jc-1"),
            transport: deleteMock,
            request: ["/journal-comments/jc-1"],
        },
        {
            name: "likeJournalComment posts an empty body to the like",
            call: () => api.likeJournalComment("jc-1"),
            transport: postMock,
            request: ["/journal-comments/jc-1/like", {}],
        },
        {
            name: "unlikeJournalComment deletes the like",
            call: () => api.unlikeJournalComment("jc-1"),
            transport: deleteMock,
            request: ["/journal-comments/jc-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("journal media uploads", () => {
    it("uploadJournalCommentMedia posts the file under the media field", async () => {
        // given
        const file = new File(["x"], "comment.png", { type: "image/png" });

        // when
        await api.uploadJournalCommentMedia("jc-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/journal-comments/jc-1/media");
        expect(formData.get("media")).toBe(file);
    });

    it("uploadJournalEntryMedia posts the file under the media field", async () => {
        // given
        const file = new File(["x"], "entry.png", { type: "image/png" });

        // when
        await api.uploadJournalEntryMedia("e-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/journal-entries/e-1/media");
        expect(formData.get("media")).toBe(file);
    });
});

describe("a user's own journals and journal follows", () => {
    const cases: RequestCase[] = [
        {
            name: "getUserJournals defaults to the first page of twenty",
            call: () => api.getUserJournals("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/journals?limit=20"],
        },
        {
            name: "getUserJournals pages through the journals",
            call: () => api.getUserJournals("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/journals?limit=5&offset=10"],
        },
        {
            name: "getUserJournals drops an offset of zero",
            call: () => api.getUserJournals("u-1", 5, 0),
            transport: fetchMock,
            request: ["/users/u-1/journals?limit=5"],
        },
        {
            name: "getUserFollowedJournals defaults to the first page of twenty",
            call: () => api.getUserFollowedJournals("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/journal-follows?limit=20"],
        },
        {
            name: "getUserFollowedJournals pages through the followed journals",
            call: () => api.getUserFollowedJournals("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/journal-follows?limit=5&offset=10"],
        },
        {
            name: "getUserFollowedJournals drops an offset of zero",
            call: () => api.getUserFollowedJournals("u-1", 5, 0),
            transport: fetchMock,
            request: ["/users/u-1/journal-follows?limit=5"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the ship API", () => {
    const shipPayload = {
        title: "Beato and Battler",
        description: "the golden witch and her opponent",
        characters: [
            { series: "umineko", character_name: "Beatrice", sort_order: 0 },
            { series: "umineko", character_id: "c-2", character_name: "Battler", sort_order: 1 },
        ],
    };

    const cases: RequestCase[] = [
        {
            name: "getShip reads a single ship",
            call: () => api.getShip("s-1"),
            transport: fetchMock,
            request: ["/ships/s-1"],
        },
        {
            name: "createShip posts the whole payload to the ship collection",
            call: () => api.createShip(shipPayload),
            transport: postMock,
            request: ["/ships", shipPayload],
        },
        {
            name: "updateShip puts the whole payload back under its own id",
            call: () => api.updateShip("s-1", shipPayload),
            transport: putMock,
            request: ["/ships/s-1", shipPayload],
        },
        {
            name: "deleteShip deletes the ship",
            call: () => api.deleteShip("s-1"),
            transport: deleteMock,
            request: ["/ships/s-1"],
        },
        {
            name: "voteShip posts an upvote",
            call: () => api.voteShip("s-1", 1),
            transport: postMock,
            request: ["/ships/s-1/vote", { value: 1 }],
        },
        {
            name: "voteShip posts a downvote",
            call: () => api.voteShip("s-1", -1),
            transport: postMock,
            request: ["/ships/s-1/vote", { value: -1 }],
        },
        {
            name: "createShipComment leaves the parent unset for a top level comment",
            call: () => api.createShipComment("s-1", "they are perfect"),
            transport: postMock,
            request: ["/ships/s-1/comments", { body: "they are perfect", parent_id: undefined }],
        },
        {
            name: "createShipComment carries the parent id of a reply",
            call: () => api.createShipComment("s-1", "quite so", "sc-0"),
            transport: postMock,
            request: ["/ships/s-1/comments", { body: "quite so", parent_id: "sc-0" }],
        },
        {
            name: "updateShipComment puts the new body under the comment id",
            call: () => api.updateShipComment("sc-1", "on reflection, no"),
            transport: putMock,
            request: ["/ship-comments/sc-1", { body: "on reflection, no" }],
        },
        {
            name: "deleteShipComment deletes the comment",
            call: () => api.deleteShipComment("sc-1"),
            transport: deleteMock,
            request: ["/ship-comments/sc-1"],
        },
        {
            name: "likeShipComment posts an empty body to the like",
            call: () => api.likeShipComment("sc-1"),
            transport: postMock,
            request: ["/ship-comments/sc-1/like", {}],
        },
        {
            name: "unlikeShipComment deletes the like",
            call: () => api.unlikeShipComment("sc-1"),
            transport: deleteMock,
            request: ["/ship-comments/sc-1/like"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("ship image and comment media uploads", () => {
    it("uploadShipImage posts the file under the image field", async () => {
        // given
        const file = new File(["x"], "ship.png", { type: "image/png" });

        // when
        await api.uploadShipImage("s-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/ships/s-1/image");
        expect(formData.get("image")).toBe(file);
    });

    it("uploadShipCommentMedia posts the file under the media field", async () => {
        // given
        const file = new File(["x"], "comment.png", { type: "image/png" });

        // when
        await api.uploadShipCommentMedia("sc-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/ship-comments/sc-1/media");
        expect(formData.get("media")).toBe(file);
    });
});

describe("the original character API", () => {
    const ocPayload = {
        name: "Victorique",
        description: "the golden fairy of the library",
        series: "custom",
        custom_series_name: "Gosick",
    };

    const cases: RequestCase[] = [
        {
            name: "listCharacters reads the cast of a series",
            call: () => api.listCharacters("umineko"),
            transport: fetchMock,
            request: ["/characters/umineko"],
        },
        {
            name: "getOC reads a single original character",
            call: () => api.getOC("oc-1"),
            transport: fetchMock,
            request: ["/ocs/oc-1"],
        },
        {
            name: "createOC posts the whole payload to the character collection",
            call: () => api.createOC(ocPayload),
            transport: postMock,
            request: ["/ocs", ocPayload],
        },
        {
            name: "updateOC puts the whole payload back under its own id",
            call: () => api.updateOC("oc-1", ocPayload),
            transport: putMock,
            request: ["/ocs/oc-1", ocPayload],
        },
        {
            name: "deleteOC deletes the character",
            call: () => api.deleteOC("oc-1"),
            transport: deleteMock,
            request: ["/ocs/oc-1"],
        },
        {
            name: "deleteOCGalleryImage interpolates a numeric image id",
            call: () => api.deleteOCGalleryImage("oc-1", 4),
            transport: deleteMock,
            request: ["/ocs/oc-1/gallery/4"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadOCImage posts the file under the image field", async () => {
        // given
        const file = new File(["x"], "oc.png", { type: "image/png" });

        // when
        await api.uploadOCImage("oc-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/ocs/oc-1/image");
        expect(formData.get("image")).toBe(file);
    });
});

describe("original character votes, favourites and comments", () => {
    const cases: RequestCase[] = [
        {
            name: "voteOC posts an upvote",
            call: () => api.voteOC("oc-1", 1),
            transport: postMock,
            request: ["/ocs/oc-1/vote", { value: 1 }],
        },
        {
            name: "voteOC posts a downvote",
            call: () => api.voteOC("oc-1", -1),
            transport: postMock,
            request: ["/ocs/oc-1/vote", { value: -1 }],
        },
        {
            name: "favouriteOC posts an empty body to the favourite",
            call: () => api.favouriteOC("oc-1"),
            transport: postMock,
            request: ["/ocs/oc-1/favourite", {}],
        },
        {
            name: "createOCComment leaves the parent unset for a top level comment",
            call: () => api.createOCComment("oc-1", "what a lovely witch"),
            transport: postMock,
            request: ["/ocs/oc-1/comments", { body: "what a lovely witch", parent_id: undefined }],
        },
        {
            name: "createOCComment carries the parent id of a reply",
            call: () => api.createOCComment("oc-1", "quite so", "occ-0"),
            transport: postMock,
            request: ["/ocs/oc-1/comments", { body: "quite so", parent_id: "occ-0" }],
        },
        {
            name: "updateOCComment puts the new body under the comment collection",
            call: () => api.updateOCComment("occ-1", "on reflection, no"),
            transport: putMock,
            request: ["/oc-comments/occ-1", { body: "on reflection, no" }],
        },
        {
            name: "deleteOCComment deletes the comment",
            call: () => api.deleteOCComment("occ-1"),
            transport: deleteMock,
            request: ["/oc-comments/occ-1"],
        },
        {
            name: "likeOCComment posts an empty body to the like",
            call: () => api.likeOCComment("occ-1"),
            transport: postMock,
            request: ["/oc-comments/occ-1/like", {}],
        },
        {
            name: "unlikeOCComment deletes the like",
            call: () => api.unlikeOCComment("occ-1"),
            transport: deleteMock,
            request: ["/oc-comments/occ-1/like"],
        },
        {
            name: "listUserOCs defaults to the first page of twenty and drops the zero offset",
            call: () => api.listUserOCs("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/ocs?limit=20"],
        },
        {
            name: "listUserOCs pages through a user's characters",
            call: () => api.listUserOCs("u-1", 5, 10),
            transport: fetchMock,
            request: ["/users/u-1/ocs?limit=5&offset=10"],
        },
        {
            name: "listUserOCSummaries reads the summary list without a query string",
            call: () => api.listUserOCSummaries("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/oc-summaries"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });

    it("uploadOCCommentMedia posts the file under the media field", async () => {
        // given
        const file = new File(["x"], "comment.png", { type: "image/png" });

        // when
        await api.uploadOCCommentMedia("occ-1", file);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/oc-comments/occ-1/media");
        expect(formData.get("media")).toBe(file);
    });
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

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the giphy API", () => {
    const favourite = {
        giphy_id: "gif-1",
        url: "https://giphy.com/gifs/gif-1",
        title: "a golden butterfly",
        preview_url: "https://giphy.com/preview/gif-1",
        width: 320,
        height: 240,
    };

    const cases: RequestCase[] = [
        {
            name: "trendingGiphy sends no query string when no paging is asked for",
            call: () => api.trendingGiphy(),
            transport: fetchMock,
            request: ["/giphy/trending"],
        },
        {
            name: "trendingGiphy pages through the trending gifs",
            call: () => api.trendingGiphy(10, 25),
            transport: fetchMock,
            request: ["/giphy/trending?offset=10&limit=25"],
        },
        {
            name: "trendingGiphy drops an offset of zero",
            call: () => api.trendingGiphy(0, 25),
            transport: fetchMock,
            request: ["/giphy/trending?limit=25"],
        },
        {
            name: "listGiphyFavourites sends no query string when no paging is asked for",
            call: () => api.listGiphyFavourites(),
            transport: fetchMock,
            request: ["/giphy/favourites"],
        },
        {
            name: "listGiphyFavourites pages through the saved gifs",
            call: () => api.listGiphyFavourites(10, 25),
            transport: fetchMock,
            request: ["/giphy/favourites?offset=10&limit=25"],
        },
        {
            name: "addGiphyFavourite posts the whole favourite to the collection",
            call: () => api.addGiphyFavourite(favourite),
            transport: postMock,
            request: ["/giphy/favourites", favourite],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("the game room API", () => {
    const cases: RequestCase[] = [
        {
            name: "inviteToGame posts the opponent and the snake cased game type",
            call: () => api.inviteToGame("u-1", "chess"),
            transport: postMock,
            request: ["/game-rooms", { opponent_id: "u-1", game_type: "chess" }],
        },
        {
            name: "getGameRoom reads a single room",
            call: () => api.getGameRoom("g-1"),
            transport: fetchMock,
            request: ["/game-rooms/g-1"],
        },
        {
            name: "acceptGameInvite posts an empty body to the accept",
            call: () => api.acceptGameInvite("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/accept", {}],
        },
        {
            name: "declineGameInvite posts an empty body to the decline",
            call: () => api.declineGameInvite("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/decline", {}],
        },
        {
            name: "cancelGameInvite posts an empty body to the cancel",
            call: () => api.cancelGameInvite("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/cancel", {}],
        },
        {
            name: "submitGameAction wraps the move in an action envelope",
            call: () => api.submitGameAction("g-1", { type: "move", from: "e2", to: "e4" }),
            transport: postMock,
            request: ["/game-rooms/g-1/action", { action: { type: "move", from: "e2", to: "e4" } }],
        },
        {
            name: "resignGame posts an empty body to the resignation",
            call: () => api.resignGame("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/resign", {}],
        },
        {
            name: "offerDraw posts an empty body to the draw offer",
            call: () => api.offerDraw("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/offer-draw", {}],
        },
        {
            name: "acceptDraw posts an empty body to the draw acceptance",
            call: () => api.acceptDraw("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/accept-draw", {}],
        },
        {
            name: "declineDraw posts an empty body to the draw refusal",
            call: () => api.declineDraw("g-1"),
            transport: postMock,
            request: ["/game-rooms/g-1/decline-draw", {}],
        },
        {
            name: "getGameScoreboard nests the scoreboard under the game type",
            call: () => api.getGameScoreboard("chess"),
            transport: fetchMock,
            request: ["/games/chess/scoreboard"],
        },
        {
            name: "getGameScoreboard leaves an underscored game type alone",
            call: () => api.getGameScoreboard("snakes_and_ladders"),
            transport: fetchMock,
            request: ["/games/snakes_and_ladders/scoreboard"],
        },
        {
            name: "listLiveGameRooms sends no query string when no game type is given",
            call: () => api.listLiveGameRooms(),
            transport: fetchMock,
            request: ["/game-rooms/live"],
        },
        {
            name: "listLiveGameRooms filters the live rooms by game type",
            call: () => api.listLiveGameRooms("othello"),
            transport: fetchMock,
            request: ["/game-rooms/live?game_type=othello"],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("spectator and player chat", () => {
    const cases: RequestCase[] = [
        {
            name: "getSpectatorChat reads the spectator messages of a room",
            call: () => api.getSpectatorChat("g-1"),
            transport: fetchMock,
            request: ["/game-rooms/g-1/chat"],
        },
        {
            name: "postSpectatorChat posts the body to the spectator chat",
            call: () => api.postSpectatorChat("g-1", "what a move"),
            transport: postMock,
            request: ["/game-rooms/g-1/chat", { body: "what a move" }],
        },
        {
            name: "getPlayerChat reads the private player messages of a room",
            call: () => api.getPlayerChat("g-1"),
            transport: fetchMock,
            request: ["/game-rooms/g-1/player-chat"],
        },
        {
            name: "postPlayerChat posts the body to the player chat",
            call: () => api.postPlayerChat("g-1", "good game"),
            transport: postMock,
            request: ["/game-rooms/g-1/player-chat", { body: "good game" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});

describe("home and sidebar activity", () => {
    const cases: RequestCase[] = [
        {
            name: "getHomeActivity reads the home activity feed",
            call: () => api.getHomeActivity(),
            transport: fetchMock,
            request: ["/home/activity"],
        },
        {
            name: "getSidebarActivity reads the sidebar activity counts",
            call: () => api.getSidebarActivity(),
            transport: fetchMock,
            request: ["/sidebar/activity"],
        },
        {
            name: "getSidebarLastVisited reads the last visited markers",
            call: () => api.getSidebarLastVisited(),
            transport: fetchMock,
            request: ["/sidebar/last-visited"],
        },
        {
            name: "markSidebarVisited posts the key of the visited section",
            call: () => api.markSidebarVisited("theories"),
            transport: postMock,
            request: ["/sidebar/last-visited", { key: "theories" }],
        },
    ];

    it.each(cases)("$name", async ({ call, transport, request }) => {
        // given the call and the request it should produce, from the table row

        // when
        await call();

        // then
        expect(transport).toHaveBeenCalledWith(...request);
    });
});
