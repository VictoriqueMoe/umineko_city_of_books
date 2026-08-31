import { beforeEach, describe, vi } from "vitest";
import * as api from "./user";
import { deleteMock, fetchMock, postMock, resetTransports, runRequestCases, type RequestCase } from "./testHarness";

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

describe("profile reads and activity", () => {
    const cases: RequestCase[] = [
        {
            name: "getUserProfile reads a profile by username",
            call: () => api.getUserProfile("kujo"),
            transport: fetchMock,
            request: ["/users/kujo"],
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
    ];

    runRequestCases(cases);
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

    runRequestCases(cases);
});

describe("a user's own galleries and art", () => {
    const cases: RequestCase[] = [
        {
            name: "getUserGalleries reads the galleries of a user",
            call: () => api.getUserGalleries("u-1"),
            transport: fetchMock,
            request: ["/users/u-1/galleries"],
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
    ];

    runRequestCases(cases);
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

    runRequestCases(cases);
});

describe("a user's own ships and mysteries", () => {
    const cases: RequestCase[] = [
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
    ];

    runRequestCases(cases);
});

describe("a user's own fanfics and favourites", () => {
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
    ];

    runRequestCases(cases);
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
    ];

    runRequestCases(cases);
});

describe("a user's own original characters", () => {
    const cases: RequestCase[] = [
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

    runRequestCases(cases);
});
