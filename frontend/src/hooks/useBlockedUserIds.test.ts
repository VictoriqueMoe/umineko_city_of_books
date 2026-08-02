import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { BlockedUserItem } from "../api/endpoints";
import { useBlockedUsers } from "../api/queries/user";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import { useBlockedUserIds } from "./useBlockedUserIds";

vi.mock("../api/queries/user", () => ({
    useBlockedUsers: vi.fn(),
}));

const mockedUseBlockedUsers = vi.mocked(useBlockedUsers);

function makeBlocked(id: string, username: string): BlockedUserItem {
    return {
        id,
        username,
        display_name: username,
        avatar_url: "",
        blocked_at: "2026-01-01T00:00:00Z",
    };
}

function stubBlocked(blocked: BlockedUserItem[]): void {
    mockedUseBlockedUsers.mockReturnValue({
        blocked,
        loading: false,
        refresh: vi.fn(),
    } as unknown as ReturnType<typeof useBlockedUsers>);
}

describe("useBlockedUserIds", () => {
    it("collects the id of every blocked user into a set", () => {
        // given
        stubBlocked([makeBlocked("u-1", "erika"), makeBlocked("u-2", "kanon")]);
        const wrapper = providerWrapper({ user: makeUser() });

        // when
        const { result } = renderHook(() => useBlockedUserIds(), { wrapper });

        // then
        expect(result.current.size).toBe(2);
        expect(result.current.has("u-1")).toBe(true);
        expect(result.current.has("u-2")).toBe(true);
        expect(result.current.has("u-3")).toBe(false);
    });

    it("returns an empty set when nobody is blocked", () => {
        // given
        stubBlocked([]);
        const wrapper = providerWrapper({ user: makeUser() });

        // when
        const { result } = renderHook(() => useBlockedUserIds(), { wrapper });

        // then
        expect(result.current.size).toBe(0);
    });

    it("collapses a repeated id into a single entry", () => {
        // given
        stubBlocked([makeBlocked("u-1", "erika"), makeBlocked("u-1", "erika")]);
        const wrapper = providerWrapper({ user: makeUser() });

        // when
        const { result } = renderHook(() => useBlockedUserIds(), { wrapper });

        // then
        expect(result.current.size).toBe(1);
    });

    it("looks the block list up against the signed in user id", () => {
        // given
        stubBlocked([]);
        const wrapper = providerWrapper({ user: makeUser({ id: "me-42" }) });

        // when
        renderHook(() => useBlockedUserIds(), { wrapper });

        // then
        expect(mockedUseBlockedUsers).toHaveBeenCalledWith("me-42");
    });

    it("asks with an empty id when nobody is signed in so the query stays disabled", () => {
        // given
        stubBlocked([]);
        const wrapper = providerWrapper({ user: null });

        // when
        const { result } = renderHook(() => useBlockedUserIds(), { wrapper });

        // then
        expect(mockedUseBlockedUsers).toHaveBeenCalledWith("");
        expect(result.current.size).toBe(0);
    });

    it("keeps the same set instance while the block list is unchanged", () => {
        // given
        stubBlocked([makeBlocked("u-1", "erika")]);
        const wrapper = providerWrapper({ user: makeUser() });

        // when
        const { result, rerender } = renderHook(() => useBlockedUserIds(), { wrapper });
        const first = result.current;
        rerender();

        // then
        expect(result.current).toBe(first);
    });

    it("rebuilds the set when the block list changes", () => {
        // given
        stubBlocked([makeBlocked("u-1", "erika")]);
        const wrapper = providerWrapper({ user: makeUser() });
        const { result, rerender } = renderHook(() => useBlockedUserIds(), { wrapper });
        const first = result.current;

        // when
        stubBlocked([makeBlocked("u-1", "erika"), makeBlocked("u-2", "kanon")]);
        rerender();

        // then
        expect(result.current).not.toBe(first);
        expect(result.current.has("u-2")).toBe(true);
    });

    it("throws when it is used outside an AuthProvider", () => {
        // given
        stubBlocked([]);
        const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});

        // when
        const attempt = () => renderHook(() => useBlockedUserIds());

        // then
        expect(attempt).toThrow("useAuth must be used within an AuthProvider");
        consoleError.mockRestore();
    });
});
