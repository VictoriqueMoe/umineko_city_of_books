import { act, renderHook, waitFor } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { makeWatchPartySession } from "../../test-utils/fixtures";
import { providerWrapper } from "../../test-utils/render";
import type { WatchPartySession } from "../../types/api";
import {
    useEndWatchParty,
    useForceMuteWatchPartyVoiceParticipant,
    useIdentifyWatchPartyParticipant,
    useJoinWatchParty,
    useKickWatchPartyParticipant,
    useLeaveWatchParty,
    useStartWatchParty,
    useTransferWatchPartyControl,
    useWatchPartyVoiceToken,
} from "./watchParty";

const mocks = vi.hoisted(() => ({
    endWatchParty: vi.fn(),
    forceMuteWatchPartyVoiceParticipant: vi.fn(),
    getWatchPartyVoiceToken: vi.fn(),
    identifyWatchPartyParticipant: vi.fn(),
    joinWatchParty: vi.fn(),
    kickWatchPartyParticipant: vi.fn(),
    leaveWatchParty: vi.fn(),
    startWatchParty: vi.fn(),
    transferWatchPartyControl: vi.fn(),
}));

vi.mock("../../api/endpoints/watchParty", () => mocks);

const roomId = "11111111-1111-1111-1111-111111111111";
const sessionId = "22222222-2222-2222-2222-222222222222";
const userId = "33333333-3333-3333-3333-333333333333";

function makeSession(): WatchPartySession {
    return makeWatchPartySession({
        id: sessionId,
        room_id: roomId,
        started_by: userId,
        controller_id: userId,
        title: "Rokkenjima night",
        started_at: "2026-08-28T12:00:00Z",
    });
}

function createRetainingQueryClient(): QueryClient {
    return new QueryClient({
        defaultOptions: {
            queries: { retry: false, gcTime: 5 * 60_000, staleTime: 30_000, refetchOnWindowFocus: false },
            mutations: { retry: false },
        },
    });
}

function setup<T>(hook: () => T) {
    const queryClient = createRetainingQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(hook, { wrapper: providerWrapper({ queryClient }) });

    return {
        result,
        queryClient,
        invalidated: () => invalidate.mock.calls.map(call => JSON.stringify(call[0]?.queryKey)),
    };
}

const listKey = JSON.stringify(queryKeys.watchParty.list(roomId));

beforeEach(() => {
    mocks.endWatchParty.mockResolvedValue(undefined);
    mocks.forceMuteWatchPartyVoiceParticipant.mockResolvedValue(undefined);
    mocks.getWatchPartyVoiceToken.mockResolvedValue({ token: "lk-token", url: "wss://livekit.test" });
    mocks.identifyWatchPartyParticipant.mockResolvedValue(undefined);
    mocks.joinWatchParty.mockResolvedValue({ session: makeSession(), embed_url: "https://hb.test/join" });
    mocks.kickWatchPartyParticipant.mockResolvedValue(undefined);
    mocks.leaveWatchParty.mockResolvedValue(undefined);
    mocks.startWatchParty.mockResolvedValue({ session: makeSession(), embed_url: "https://hb.test/start" });
    mocks.transferWatchPartyControl.mockResolvedValue(undefined);
});

describe("useStartWatchParty", () => {
    it("starts the party in the room the hook was built for and returns the embed url", async () => {
        // given
        const { result } = setup(() => useStartWatchParty(roomId));

        // when
        act(() => {
            result.current.mutate({ title: "Rokkenjima night", region: "eu-west", type: "hyperbeam", light: true });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.startWatchParty).toHaveBeenCalledWith(roomId, {
            title: "Rokkenjima night",
            region: "eu-west",
            type: "hyperbeam",
            light: true,
        });
        expect(result.current.data?.embed_url).toBe("https://hb.test/start");
    });

    it("refreshes the session list of its own room", async () => {
        // given
        const { result, invalidated } = setup(() => useStartWatchParty(roomId));

        // when
        act(() => {
            result.current.mutate({ type: "screenshare" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(invalidated()).toContain(listKey);
    });

    it("fails loudly rather than posting to a room that is not open", async () => {
        // given
        const { result } = setup(() => useStartWatchParty(null));

        // when
        act(() => {
            result.current.mutate({ type: "hyperbeam" });
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(mocks.startWatchParty).not.toHaveBeenCalled();
        expect(result.current.error?.message).toBe("No chat room is open.");
    });
});

describe("useJoinWatchParty", () => {
    it("joins the session and refreshes the list", async () => {
        // given
        const { result, invalidated } = setup(() => useJoinWatchParty(roomId));

        // when
        act(() => {
            result.current.mutate(sessionId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.joinWatchParty).toHaveBeenCalledWith(roomId, sessionId);
        expect(result.current.data?.embed_url).toBe("https://hb.test/join");
        expect(invalidated()).toContain(listKey);
    });
});

describe("useLeaveWatchParty", () => {
    it("leaves the session and refreshes the list", async () => {
        // given
        const { result, invalidated } = setup(() => useLeaveWatchParty(roomId));

        // when
        act(() => {
            result.current.mutate(sessionId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.leaveWatchParty).toHaveBeenCalledWith(roomId, sessionId);
        expect(invalidated()).toContain(listKey);
    });
});

describe("useEndWatchParty", () => {
    it("ends the session and refreshes the list", async () => {
        // given
        const { result, invalidated } = setup(() => useEndWatchParty(roomId));

        // when
        act(() => {
            result.current.mutate(sessionId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.endWatchParty).toHaveBeenCalledWith(roomId, sessionId);
        expect(invalidated()).toContain(listKey);
    });

    it("surfaces a refused end rather than leaving the party looking closed", async () => {
        // given
        mocks.endWatchParty.mockRejectedValue(new Error("only the host may end it"));
        const { result } = setup(() => useEndWatchParty(roomId));

        // when
        act(() => {
            result.current.mutate(sessionId);
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error?.message).toBe("only the host may end it");
    });
});

describe("useTransferWatchPartyControl", () => {
    it("hands control to the named participant and refreshes the list", async () => {
        // given
        const { result, invalidated } = setup(() => useTransferWatchPartyControl(roomId));

        // when
        act(() => {
            result.current.mutate({ sessionId, userId });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.transferWatchPartyControl).toHaveBeenCalledWith(roomId, sessionId, userId);
        expect(invalidated()).toContain(listKey);
    });
});

describe("useKickWatchPartyParticipant", () => {
    it("kicks the named participant and refreshes the list", async () => {
        // given
        const { result, invalidated } = setup(() => useKickWatchPartyParticipant(roomId));

        // when
        act(() => {
            result.current.mutate({ sessionId, userId });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.kickWatchPartyParticipant).toHaveBeenCalledWith(roomId, sessionId, userId);
        expect(invalidated()).toContain(listKey);
    });
});

describe("useIdentifyWatchPartyParticipant", () => {
    it("registers the embed identity without refetching the list", async () => {
        // given
        const { result, invalidated } = setup(() => useIdentifyWatchPartyParticipant(roomId));

        // when
        act(() => {
            result.current.mutate({ sessionId, identifier: "hb-identity" });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.identifyWatchPartyParticipant).toHaveBeenCalledWith(roomId, sessionId, "hb-identity");
        expect(invalidated()).toEqual([]);
    });
});

describe("useWatchPartyVoiceToken", () => {
    it("returns the voice token and never caches it under a query key", async () => {
        // given
        const { result, queryClient } = setup(() => useWatchPartyVoiceToken(roomId));

        // when
        act(() => {
            result.current.mutate(sessionId);
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.getWatchPartyVoiceToken).toHaveBeenCalledWith(roomId, sessionId);
        expect(result.current.data).toEqual({ token: "lk-token", url: "wss://livekit.test" });
        expect(queryClient.getQueryCache().getAll()).toEqual([]);
    });
});

describe("useForceMuteWatchPartyVoiceParticipant", () => {
    it("mutes the named participant", async () => {
        // given
        const { result } = setup(() => useForceMuteWatchPartyVoiceParticipant(roomId));

        // when
        act(() => {
            result.current.mutate({ sessionId, userId, muted: true });
        });

        // then
        await waitFor(() => expect(result.current.isSuccess).toBe(true));
        expect(mocks.forceMuteWatchPartyVoiceParticipant).toHaveBeenCalledWith(roomId, sessionId, userId, true);
    });

    it("surfaces the failure the modal used to swallow", async () => {
        // given
        mocks.forceMuteWatchPartyVoiceParticipant.mockRejectedValue(new Error("not a moderator"));
        const { result } = setup(() => useForceMuteWatchPartyVoiceParticipant(roomId));

        // when
        act(() => {
            result.current.mutate({ sessionId, userId, muted: true });
        });

        // then
        await waitFor(() => expect(result.current.isError).toBe(true));
        expect(result.current.error?.message).toBe("not a moderator");
    });
});
