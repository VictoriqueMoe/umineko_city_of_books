import { renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserProfile } from "../../types/api";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { getUserProfile } from "../endpoints";
import { useProfile } from "./profile";

vi.mock("../endpoints", () => ({
    getUserProfile: vi.fn(),
}));

const mockedGetUserProfile = vi.mocked(getUserProfile);

function firstKey(qc: QueryClient): readonly unknown[] {
    return qc.getQueryCache().getAll()[0].queryKey;
}

beforeEach(() => {
    mockedGetUserProfile.mockResolvedValue(makeUser());
});

describe("useProfile", () => {
    it("keys the profile query by the username it was handed", async () => {
        // given
        const qc = createTestQueryClient();

        // when
        const { result } = renderHook(() => useProfile("beatrice"), { wrapper: providerWrapper({ queryClient: qc }) });
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(firstKey(qc)).toEqual(["profile", "username", "beatrice"]);
        expect(mockedGetUserProfile).toHaveBeenCalledWith("beatrice");
    });

    it("does not ask the server for a profile without a username", () => {
        // given
        const wrapper = providerWrapper();

        // when
        const { result } = renderHook(() => useProfile(""), { wrapper });

        // then
        expect(mockedGetUserProfile).not.toHaveBeenCalled();
        expect(result.current.profile).toBeNull();
        expect(result.current.loading).toBe(false);
    });

    it("reports no profile while the request is still in flight", () => {
        // given
        mockedGetUserProfile.mockReturnValue(new Promise<UserProfile>(() => {}));

        // when
        const { result } = renderHook(() => useProfile("beatrice"), { wrapper: providerWrapper() });

        // then
        expect(result.current.profile).toBeNull();
        expect(result.current.loading).toBe(true);
    });

    it("returns the profile once it has loaded", async () => {
        // given
        const profile = makeUser({ username: "erika", display_name: "Erika Furudo" });
        mockedGetUserProfile.mockResolvedValue(profile);

        // when
        const { result } = renderHook(() => useProfile("erika"), { wrapper: providerWrapper() });

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(result.current.profile).toEqual(profile);
    });

    it("refetches under a new key when the username changes", async () => {
        // given
        mockedGetUserProfile.mockImplementation(username => Promise.resolve(makeUser({ username })));
        const wrapper = providerWrapper();
        const { result, rerender } = renderHook(props => useProfile(props.username), {
            wrapper,
            initialProps: { username: "beatrice" },
        });
        await waitFor(() => expect(result.current.profile?.username).toBe("beatrice"));

        // when
        rerender({ username: "erika" });

        // then
        await waitFor(() => expect(result.current.profile?.username).toBe("erika"));
        expect(mockedGetUserProfile).toHaveBeenCalledTimes(2);
    });
});
