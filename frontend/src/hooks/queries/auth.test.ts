import type { QueryClient } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useMe } from "./auth";

const authEndpoints = vi.hoisted(() => ({
    getSession: vi.fn(),
}));

const userEndpoints = vi.hoisted(() => ({
    getUserProfile: vi.fn(),
}));

vi.mock("../../api/endpoints/auth", () => authEndpoints);
vi.mock("../../api/endpoints/user", () => userEndpoints);

function setup<T>(hook: () => T) {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return { ...rendered, queryClient };
}

function firstKey(queryClient: QueryClient): readonly unknown[] {
    return queryClient.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    authEndpoints.getSession.mockResolvedValue({ authenticated: true, username: "beatrice", permissions: [] });
    userEndpoints.getUserProfile.mockResolvedValue(makeUser());
});

describe("useMe", () => {
    it("exposes the signed in profile under the auth me key", async () => {
        // given
        const me = makeUser({ username: "battler" });
        authEndpoints.getSession.mockResolvedValue({ authenticated: true, username: "battler", permissions: [] });
        userEndpoints.getUserProfile.mockResolvedValue(me);

        // when
        const { result, queryClient } = setup(() => useMe());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["auth", "me"]);
        expect(result.current.me).toEqual({ ...me, permissions: [] });
    });

    it("reports no profile while the session is still being checked", async () => {
        // given
        const { result } = setup(() => useMe());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.me).toBeNull();
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("reports no profile for a signed out visitor without asking for one", async () => {
        // given
        authEndpoints.getSession.mockResolvedValue({ authenticated: false });

        // when
        const { result } = setup(() => useMe());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.me).toBeNull();
        expect(userEndpoints.getUserProfile).not.toHaveBeenCalled();
    });

    it("fetches the profile of the signed in username and merges the session permissions", async () => {
        // given
        const profile = makeUser({ username: "kujo" });
        authEndpoints.getSession.mockResolvedValue({
            authenticated: true,
            username: "kujo",
            permissions: ["view_admin_panel"],
        });
        userEndpoints.getUserProfile.mockResolvedValue(profile);

        // when
        const { result } = setup(() => useMe());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(userEndpoints.getUserProfile).toHaveBeenCalledWith("kujo");
        expect(result.current.me).toEqual({ ...profile, permissions: ["view_admin_panel"] });
    });

    it("falls back to no permissions when the session omits them", async () => {
        // given
        const profile = makeUser({ username: "kujo" });
        authEndpoints.getSession.mockResolvedValue({ authenticated: true, username: "kujo" });
        userEndpoints.getUserProfile.mockResolvedValue(profile);

        // when
        const { result } = setup(() => useMe());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.me).toEqual({ ...profile, permissions: [] });
    });

    it("refetches the session when refresh is called", async () => {
        // given
        const { result } = setup(() => useMe());
        await waitFor(() => expect(result.current.loading).toBe(false));

        // when
        await result.current.refresh();

        // then
        expect(authEndpoints.getSession).toHaveBeenCalledTimes(2);
    });
});
