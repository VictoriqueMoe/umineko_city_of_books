import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import {
    useAdminAnnouncements,
    useAdminSettings,
    useAdminStats,
    useAdminUser,
    useAdminUsers,
    useAuditLog,
    useBannedGifs,
    useGlobalBannedWords,
    useInvites,
    useReports,
    useUserAuditLog,
    useUserIPMatches,
    useVanityRoleUsers,
    useVanityRoles,
} from "./admin";

const endpoints = vi.hoisted(() => ({
    getAdminSettings: vi.fn(),
    getAdminStats: vi.fn(),
    getAdminUser: vi.fn(),
    getAdminUsers: vi.fn(),
    getAuditLog: vi.fn(),
    getBannedGifs: vi.fn(),
    getInvites: vi.fn(),
    getReports: vi.fn(),
    getUserAuditLog: vi.fn(),
    getUserIPMatches: vi.fn(),
    getVanityRoleUsers: vi.fn(),
    getVanityRoles: vi.fn(),
    listAnnouncements: vi.fn(),
    listGlobalBannedWords: vi.fn(),
}));

vi.mock("../endpoints", () => endpoints);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { ...rendered, queryClient };
}

function firstKey(queryClient: QueryClient): readonly unknown[] {
    return queryClient.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    endpoints.getAdminSettings.mockResolvedValue({ site_name: "When They Cry" });
    endpoints.getAdminStats.mockResolvedValue({ user_count: 3 });
    endpoints.getAdminUser.mockResolvedValue({ id: "u-1", username: "beatrice" });
    endpoints.getAdminUsers.mockResolvedValue({ users: [], total: 0 });
    endpoints.getAuditLog.mockResolvedValue({ entries: [], total: 0 });
    endpoints.getBannedGifs.mockResolvedValue({ entries: [] });
    endpoints.getInvites.mockResolvedValue({ invites: [] });
    endpoints.getReports.mockResolvedValue({ reports: [] });
    endpoints.getUserAuditLog.mockResolvedValue({ entries: [], total: 0 });
    endpoints.getUserIPMatches.mockResolvedValue({ ip: "", users: [] });
    endpoints.getVanityRoleUsers.mockResolvedValue({ users: [], total: 0 });
    endpoints.getVanityRoles.mockResolvedValue([]);
    endpoints.listAnnouncements.mockResolvedValue({ announcements: [], total: 0 });
    endpoints.listGlobalBannedWords.mockResolvedValue({ rules: [] });
});

describe("useAdminAnnouncements", () => {
    it("asks for the first hundred announcements under the admin announcements key", async () => {
        // given
        endpoints.listAnnouncements.mockResolvedValue({ announcements: [{ id: "a-1" }], total: 1 });

        // when
        const { result, queryClient } = setup(() => useAdminAnnouncements());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listAnnouncements).toHaveBeenCalledWith(100, 0);
        expect(firstKey(queryClient)).toEqual(["admin", "announcements"]);
        expect(result.current.announcements).toEqual([{ id: "a-1" }]);
    });

    it("reports an empty list while the request is still in flight", async () => {
        // given
        const { result } = setup(() => useAdminAnnouncements());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.announcements).toEqual([]);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("falls back to an empty list when the response carries no announcements", async () => {
        // given
        endpoints.listAnnouncements.mockResolvedValue({});

        // when
        const { result } = setup(() => useAdminAnnouncements());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.announcements).toEqual([]);
    });
});

describe("useAdminUsers", () => {
    it("forwards the search, limit and offset it was given", async () => {
        // given
        endpoints.getAdminUsers.mockResolvedValue({ users: [{ id: "u-1" }], total: 42 });

        // when
        const { result } = setup(() => useAdminUsers("beato", 20, 40));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getAdminUsers).toHaveBeenCalledWith({ search: "beato", limit: 20, offset: 40 });
        expect(result.current.users).toEqual([{ id: "u-1" }]);
        expect(result.current.total).toBe(42);
    });

    it("reports a failed request so the page can tell it apart from an empty roster", async () => {
        // given
        endpoints.getAdminUsers.mockRejectedValue(new Error("boom"));

        // when
        const { result } = setup(() => useAdminUsers("beato", 20, 40));

        // then
        await waitFor(() => expect(result.current.error).toBe(true));
        expect(result.current.users).toEqual([]);
        expect(result.current.loading).toBe(false);
    });

    it("keys the cache entry by the search, limit and offset", () => {
        // given
        const { queryClient } = setup(() => useAdminUsers("beato", 20, 40));

        // when
        const key = firstKey(queryClient);

        // then
        expect(key).toEqual(["admin", "users", { search: "beato", limit: 20, offset: 40 }]);
    });

    it("defaults the user list and the total when the response is empty", async () => {
        // given
        endpoints.getAdminUsers.mockResolvedValue({});

        // when
        const { result } = setup(() => useAdminUsers("", 20, 0));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.users).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useAdminUser", () => {
    it("loads the user behind the id it was given", async () => {
        // given
        const { result, queryClient } = setup(() => useAdminUser("u-7"));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.getAdminUser).toHaveBeenCalledWith("u-7");
        expect(firstKey(queryClient)).toEqual(["admin", "user", "u-7"]);
        expect(result.current.user).toEqual({ id: "u-1", username: "beatrice" });
    });

    it("stays idle and reports no user when the id is empty", () => {
        // given
        const { result } = setup(() => useAdminUser(""));

        // when
        const current = result.current;

        // then
        expect(endpoints.getAdminUser).not.toHaveBeenCalled();
        expect(current.user).toBeNull();
        expect(current.loading).toBe(false);
    });
});

describe("useUserIPMatches", () => {
    it("returns the shared address and the accounts behind it", async () => {
        // given
        endpoints.getUserIPMatches.mockResolvedValue({ ip: "10.0.0.1", users: [{ id: "u-2" }] });

        // when
        const { result, queryClient } = setup(() => useUserIPMatches("u-1", true));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getUserIPMatches).toHaveBeenCalledWith("u-1");
        expect(firstKey(queryClient)).toEqual(["admin", "user", "u-1", "ip-matches"]);
        expect(result.current.ip).toBe("10.0.0.1");
        expect(result.current.users).toEqual([{ id: "u-2" }]);
    });

    it("does not look anything up while it is switched off", () => {
        // given
        const { result } = setup(() => useUserIPMatches("u-1", false));

        // when
        const current = result.current;

        // then
        expect(endpoints.getUserIPMatches).not.toHaveBeenCalled();
        expect(current.ip).toBe("");
        expect(current.users).toEqual([]);
    });

    it("does not look anything up when the id is empty", () => {
        // given
        setup(() => useUserIPMatches("", true));

        // when
        const calls = endpoints.getUserIPMatches.mock.calls;

        // then
        expect(calls).toHaveLength(0);
    });

    it("flags a failure when the lookup rejects", async () => {
        // given
        endpoints.getUserIPMatches.mockRejectedValue(new Error("forbidden"));

        // when
        const { result } = setup(() => useUserIPMatches("u-1", true));

        // then
        await waitFor(() => expect(result.current.failed).toBe(true));
        expect(result.current.users).toEqual([]);
    });
});

describe("useUserAuditLog", () => {
    it("forwards the id together with the requested page window", async () => {
        // given
        endpoints.getUserAuditLog.mockResolvedValue({ entries: [{ id: 1 }], total: 9 });

        // when
        const { result, queryClient } = setup(() => useUserAuditLog("u-1", true, 25, 50));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getUserAuditLog).toHaveBeenCalledWith("u-1", 25, 50);
        expect(firstKey(queryClient)).toEqual(["admin", "user", "u-1", "audit-log", 25, 50]);
        expect(result.current.total).toBe(9);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useUserAuditLog("u-1", false, 25, 0));

        // when
        const current = result.current;

        // then
        expect(endpoints.getUserAuditLog).not.toHaveBeenCalled();
        expect(current.entries).toEqual([]);
        expect(current.total).toBe(0);
    });

    it("flags a failure when the audit log rejects", async () => {
        // given
        endpoints.getUserAuditLog.mockRejectedValue(new Error("nope"));

        // when
        const { result } = setup(() => useUserAuditLog("u-1", true, 25, 0));

        // then
        await waitFor(() => expect(result.current.failed).toBe(true));
    });
});

describe("useAdminStats", () => {
    it("exposes the stats payload under the admin stats key", async () => {
        // given
        const { result, queryClient } = setup(() => useAdminStats());

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(queryClient)).toEqual(["admin", "stats"]);
        expect(result.current.stats).toEqual({ user_count: 3 });
    });

    it("reports no stats before the request settles", async () => {
        // given
        const { result } = setup(() => useAdminStats());

        // when
        const initial = result.current;

        // then
        expect(initial.stats).toBeNull();
        await waitFor(() => expect(result.current.loading).toBe(false));
    });
});

describe("useAdminSettings", () => {
    it("exposes the settings payload under the admin settings key", async () => {
        // given
        const { result, queryClient } = setup(() => useAdminSettings());

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(queryClient)).toEqual(["admin", "settings"]);
        expect(result.current.settings).toEqual({ site_name: "When They Cry" });
    });

    it("reports no settings before the request settles", async () => {
        // given
        const { result } = setup(() => useAdminSettings());

        // when
        const initial = result.current;

        // then
        expect(initial.settings).toBeNull();
        await waitFor(() => expect(result.current.loading).toBe(false));
    });
});

describe("useAuditLog", () => {
    it("drops an empty action filter so the endpoint sees no filter at all", async () => {
        // given
        const { result } = setup(() => useAuditLog("", 50, 0));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.getAuditLog).toHaveBeenCalledWith({ action: undefined, limit: 50, offset: 0 });
    });

    it("forwards a chosen action and keys the cache entry by it", async () => {
        // given
        endpoints.getAuditLog.mockResolvedValue({ entries: [{ id: 3 }], total: 1 });

        // when
        const { result, queryClient } = setup(() => useAuditLog("ban_user", 50, 100));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getAuditLog).toHaveBeenCalledWith({ action: "ban_user", limit: 50, offset: 100 });
        expect(firstKey(queryClient)).toEqual(["admin", "audit-log", { action: "ban_user", limit: 50, offset: 100 }]);
        expect(result.current.entries).toEqual([{ id: 3 }]);
    });

    it("defaults the entries and the total when the response is empty", async () => {
        // given
        endpoints.getAuditLog.mockResolvedValue({});

        // when
        const { result } = setup(() => useAuditLog("", 50, 0));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.entries).toEqual([]);
        expect(result.current.total).toBe(0);
    });
});

describe("useInvites", () => {
    it("forwards the requested page window to the endpoint", async () => {
        // given
        endpoints.getInvites.mockResolvedValue({ invites: [{ code: "abc" }] });

        // when
        const { result } = setup(() => useInvites(50, 100));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getInvites).toHaveBeenCalledWith({ limit: 50, offset: 100 });
        expect(result.current.invites).toEqual([{ code: "abc" }]);
    });

    it("folds the page window into the cache key", () => {
        // given
        const { queryClient } = setup(() => useInvites(50, 100));

        // when
        const key = firstKey(queryClient);

        // then
        expect(key).toEqual(["admin", "invites", 50, 100]);
    });

    it("refetches when the page window moves rather than serving the cached first page", async () => {
        // given
        endpoints.getInvites.mockResolvedValue({ invites: [{ code: "first-page" }] });
        const queryClient = createTestQueryClient();
        const { result, rerender } = renderHook(({ offset }: { offset: number }) => useInvites(50, offset), {
            initialProps: { offset: 0 },
            wrapper: providerWrapper({ queryClient }),
        });
        await waitFor(() => expect(result.current.loading).toBe(false));
        endpoints.getInvites.mockResolvedValue({ invites: [{ code: "second-page" }] });

        // when
        rerender({ offset: 50 });

        // then
        await waitFor(() => expect(result.current.invites).toEqual([{ code: "second-page" }]));
        expect(endpoints.getInvites).toHaveBeenLastCalledWith({ limit: 50, offset: 50 });
    });

    it("keeps the admin invites prefix so an invite mutation still invalidates the page", () => {
        // given
        const { queryClient } = setup(() => useInvites(50, 100));

        // when
        const key = firstKey(queryClient);

        // then
        expect(key.slice(0, 2)).toEqual(["admin", "invites"]);
    });

    it("defaults to an empty invite list", async () => {
        // given
        endpoints.getInvites.mockResolvedValue({});

        // when
        const { result } = setup(() => useInvites(50, 0));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.invites).toEqual([]);
    });
});

describe("useReports", () => {
    it("asks for the reports of a given status and keys the cache entry by it", async () => {
        // given
        endpoints.getReports.mockResolvedValue({ reports: [{ id: "r-1" }] });

        // when
        const { result, queryClient } = setup(() => useReports("resolved"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getReports).toHaveBeenCalledWith("resolved");
        expect(firstKey(queryClient)).toEqual(["admin", "reports", { status: "resolved" }]);
        expect(result.current.reports).toEqual([{ id: "r-1" }]);
    });

    it("defaults to an empty report list", async () => {
        // given
        endpoints.getReports.mockResolvedValue({});

        // when
        const { result } = setup(() => useReports("open"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.reports).toEqual([]);
    });
});

describe("useBannedGifs", () => {
    it("exposes the banned entries under the banned gifs key", async () => {
        // given
        endpoints.getBannedGifs.mockResolvedValue({ entries: [{ kind: "gif", value: "g-1" }] });

        // when
        const { result, queryClient } = setup(() => useBannedGifs());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["admin", "banned-gifs"]);
        expect(result.current.entries).toEqual([{ kind: "gif", value: "g-1" }]);
    });

    it("defaults to an empty entry list", async () => {
        // given
        endpoints.getBannedGifs.mockResolvedValue({});

        // when
        const { result } = setup(() => useBannedGifs());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.entries).toEqual([]);
    });
});

describe("useGlobalBannedWords", () => {
    it("exposes the global rules under the global banned words scope", async () => {
        // given
        endpoints.listGlobalBannedWords.mockResolvedValue({ rules: [{ id: "w-1" }] });

        // when
        const { result, queryClient } = setup(() => useGlobalBannedWords());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["admin", "banned-words", "global"]);
        expect(result.current.rules).toEqual([{ id: "w-1" }]);
    });

    it("defaults to an empty rule list", async () => {
        // given
        endpoints.listGlobalBannedWords.mockResolvedValue({});

        // when
        const { result } = setup(() => useGlobalBannedWords());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.rules).toEqual([]);
    });
});

describe("useVanityRoles", () => {
    it("returns the role list the endpoint hands back", async () => {
        // given
        endpoints.getVanityRoles.mockResolvedValue([{ id: "vr-1" }]);

        // when
        const { result, queryClient } = setup(() => useVanityRoles());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["admin", "vanity-roles"]);
        expect(result.current.roles).toEqual([{ id: "vr-1" }]);
    });

    it("reports no roles before the request settles", async () => {
        // given
        const { result } = setup(() => useVanityRoles());

        // when
        const initial = result.current;

        // then
        expect(initial.roles).toEqual([]);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });
});

describe("useVanityRoleUsers", () => {
    it("drops an empty search term before calling the endpoint", async () => {
        // given
        const { result } = setup(() => useVanityRoleUsers("vr-1", "", 20, 0));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.getVanityRoleUsers).toHaveBeenCalledWith("vr-1", {
            search: undefined,
            limit: 20,
            offset: 0,
        });
    });

    it("forwards a search term and keys the cache entry by every argument", async () => {
        // given
        endpoints.getVanityRoleUsers.mockResolvedValue({ users: [{ id: "u-1" }], total: 5 });

        // when
        const { result, queryClient } = setup(() => useVanityRoleUsers("vr-1", "beato", 20, 40));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getVanityRoleUsers).toHaveBeenCalledWith("vr-1", {
            search: "beato",
            limit: 20,
            offset: 40,
        });
        expect(firstKey(queryClient)).toEqual(["admin", "vanity-role-users", "vr-1", "beato", 20, 40]);
        expect(result.current.total).toBe(5);
    });

    it("stays idle when no role id has been chosen", () => {
        // given
        const { result } = setup(() => useVanityRoleUsers("", "", 20, 0));

        // when
        const current = result.current;

        // then
        expect(endpoints.getVanityRoleUsers).not.toHaveBeenCalled();
        expect(current.users).toEqual([]);
        expect(current.total).toBe(0);
    });
});
