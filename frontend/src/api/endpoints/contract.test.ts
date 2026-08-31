import { beforeEach, describe, expect, it, vi } from "vitest";
import * as adminEndpoints from "./admin";
import * as artEndpoints from "./art";
import * as authEndpoints from "./auth";
import * as chatEndpoints from "./chat";
import * as fanficEndpoints from "./fanfic";
import * as gameRoomEndpoints from "./gameRoom";
import * as giphyEndpoints from "./giphy";
import * as journalEndpoints from "./journal";
import * as mysteryEndpoints from "./mystery";
import * as ocEndpoints from "./oc";
import * as postEndpoints from "./post";
import * as reportEndpoints from "./report";
import * as searchEndpoints from "./search";
import * as shipEndpoints from "./ship";
import * as siteEndpoints from "./site";
import * as streamEndpoints from "./stream";
import * as theoryEndpoints from "./theory";
import * as userEndpoints from "./user";
import * as watchPartyEndpoints from "./watchParty";
import {
    deleteMock,
    fetchMock,
    lastFormData,
    patchMock,
    postFormDataMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

const api = {
    ...adminEndpoints,
    ...artEndpoints,
    ...authEndpoints,
    ...chatEndpoints,
    ...fanficEndpoints,
    ...gameRoomEndpoints,
    ...giphyEndpoints,
    ...journalEndpoints,
    ...mysteryEndpoints,
    ...ocEndpoints,
    ...postEndpoints,
    ...reportEndpoints,
    ...searchEndpoints,
    ...shipEndpoints,
    ...siteEndpoints,
    ...streamEndpoints,
    ...theoryEndpoints,
    ...userEndpoints,
    ...watchPartyEndpoints,
};

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
            name: "getVanityRoleUsers builds its own query string rather than using buildQueryString, and always sends a limit",
            call: () => api.getVanityRoleUsers("role-1", {}),
            transport: fetchMock,
            request: ["/admin/vanity-roles/role-1/users?limit=20"],
        },
        {
            name: "getVanityRoleUsers builds its own query string rather than using buildQueryString, so a space in the search term is percent encoded as %20 and not as +",
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

    runRequestCases(cases);
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

    runRequestCases(cases);
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

    runRequestCases(cases);
});

describe("optional request body fields", () => {
    const cases: RequestCase[] = [
        {
            name: "register leaves the invite code and turnstile token unset when they are not supplied",
            call: () => api.register("kujo", "kujo@example.com", "pw", "Kujo"),
            transport: postMock,
            strict: true,
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
            strict: true,
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
            strict: true,
            request: ["/auth/login", { username: "kujo", password: "pw", turnstile_token: undefined }],
        },
        {
            name: "createPost defaults to the general corner with no poll or share",
            call: () => api.createPost("hello"),
            transport: postMock,
            strict: true,
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
            strict: true,
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
            strict: true,
            request: ["/chat/rooms/r-1/join", { ghost: undefined }],
        },
        {
            name: "joinChatRoom forwards the ghost flag",
            call: () => api.joinChatRoom("r-1", { ghost: true }),
            transport: postMock,
            strict: true,
            request: ["/chat/rooms/r-1/join", { ghost: true }],
        },
        {
            name: "createJournalComment can target an entry without a parent comment",
            call: () => api.createJournalComment("j-1", "lovely", undefined, "e-1"),
            transport: postMock,
            strict: true,
            request: ["/journals/j-1/comments", { body: "lovely", parent_id: undefined, entry_id: "e-1" }],
        },
        {
            name: "addMysteryClue leaves the player unset for a public clue",
            call: () => api.addMysteryClue("m-1", "the gold is real", "red"),
            transport: postMock,
            strict: true,
            request: ["/mysteries/m-1/clues", { body: "the gold is real", truth_type: "red", player_id: undefined }],
        },
        {
            name: "createReport leaves the context id unset when there is no context",
            call: () => api.createReport("post", "p-1", "spam"),
            transport: postMock,
            strict: true,
            request: ["/report", { target_type: "post", target_id: "p-1", context_id: undefined, reason: "spam" }],
        },
        {
            name: "resolveSuggestion defaults the status to done",
            call: () => api.resolveSuggestion("p-1"),
            transport: postMock,
            strict: true,
            request: ["/posts/p-1/resolve", { status: "done" }],
        },
        {
            name: "createGallery defaults the description to an empty string",
            call: () => api.createGallery("Witches"),
            transport: postMock,
            strict: true,
            request: ["/galleries", { name: "Witches", description: "" }],
        },
    ];

    runRequestCases(cases);
});

describe("multipart uploads", () => {
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

    it("getSession asks the session endpoint and asks for nothing else", async () => {
        // given
        fetchMock.mockResolvedValue({ authenticated: false });

        // when
        const result = await api.getSession();

        // then
        expect(fetchMock).toHaveBeenCalledTimes(1);
        expect(fetchMock).toHaveBeenCalledWith("/auth/session");
        expect(result).toEqual({ authenticated: false });
    });
});
