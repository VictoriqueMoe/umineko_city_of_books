import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeWatchPartySession } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { WatchPartyListResponse, WatchPartySession } from "../../types/api";
import { useWatchParties } from "./watchParty";

const mocks = vi.hoisted(() => ({
    listWatchParties: vi.fn(),
}));

vi.mock("../../api/endpoints/watchParty", () => mocks);

const roomId = "11111111-1111-1111-1111-111111111111";

function makeSession(id: string): WatchPartySession {
    return makeWatchPartySession({
        id,
        room_id: roomId,
        started_by: "user-1",
        controller_id: "user-1",
        title: "Rokkenjima night",
        started_at: "2026-08-28T12:00:00Z",
    });
}

function makeList(sessions: WatchPartySession[]): WatchPartyListResponse {
    return { sessions, enabled: true, screen_share_enabled: true };
}

let queryClient: QueryClient;

beforeEach(() => {
    queryClient = createTestQueryClient();
    mocks.listWatchParties.mockResolvedValue(makeList([makeSession("session-1")]));
});

describe("useWatchParties", () => {
    it("reports the sessions and both feature flags once the list arrives", async () => {
        // given
        const { result } = renderHook(() => useWatchParties(roomId), { wrapper: providerWrapper({ queryClient }) });

        // then
        expect(result.current.loaded).toBe(false);

        await waitFor(() => {
            expect(result.current.sessions).toHaveLength(1);
        });
        expect(result.current.enabled).toBe(true);
        expect(result.current.screenShareEnabled).toBe(true);
        expect(result.current.loaded).toBe(true);
    });

    it("keys the list by the room so switching rooms never shows the previous room's sessions", async () => {
        // given
        mocks.listWatchParties.mockImplementation((id: string) =>
            Promise.resolve(makeList([makeSession(`session-of-${id}`)])),
        );
        const { result, rerender } = renderHook(({ id }: { id: string }) => useWatchParties(id), {
            wrapper: providerWrapper({ queryClient }),
            initialProps: { id: "room-a" },
        });
        await waitFor(() => {
            expect(result.current.sessions[0]?.id).toBe("session-of-room-a");
        });

        // when
        rerender({ id: "room-b" });

        // then
        expect(result.current.sessions).toEqual([]);
        await waitFor(() => {
            expect(result.current.sessions[0]?.id).toBe("session-of-room-b");
        });
        expect(mocks.listWatchParties).toHaveBeenLastCalledWith("room-b");
    });

    it("asks for nothing while no room is open", () => {
        // given
        const { result } = renderHook(() => useWatchParties(null), { wrapper: providerWrapper({ queryClient }) });

        // then
        expect(mocks.listWatchParties).not.toHaveBeenCalled();
        expect(result.current.loaded).toBe(false);
        expect(result.current.enabled).toBe(false);
        expect(result.current.sessions).toEqual([]);
    });

    it("surfaces the load failure", async () => {
        // given
        mocks.listWatchParties.mockRejectedValue(new Error("no watch parties for you"));

        // when
        const { result } = renderHook(() => useWatchParties(roomId), { wrapper: providerWrapper({ queryClient }) });

        // then
        await waitFor(() => {
            expect(result.current.error).toBe("no watch parties for you");
        });
        expect(result.current.loaded).toBe(false);
    });

    it("refetches on demand", async () => {
        // given
        const { result } = renderHook(() => useWatchParties(roomId), { wrapper: providerWrapper({ queryClient }) });
        await waitFor(() => {
            expect(result.current.sessions).toHaveLength(1);
        });

        // when
        mocks.listWatchParties.mockResolvedValue(makeList([makeSession("session-1"), makeSession("session-2")]));
        await act(async () => {
            await result.current.refresh();
        });

        // then
        await waitFor(() => {
            expect(result.current.sessions).toHaveLength(2);
        });
    });
});
