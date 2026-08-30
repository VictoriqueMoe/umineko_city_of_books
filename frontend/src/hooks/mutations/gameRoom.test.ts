import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import {
    useAcceptDraw,
    useAcceptGameInvite,
    useCancelGameInvite,
    useDeclineDraw,
    useDeclineGameInvite,
    useInviteToGame,
    useOfferDraw,
    usePostPlayerChat,
    usePostSpectatorChat,
    useResignGame,
    useSubmitGameAction,
} from "./gameRoom";

const mocks = vi.hoisted(() => ({
    acceptDraw: vi.fn(),
    acceptGameInvite: vi.fn(),
    cancelGameInvite: vi.fn(),
    declineDraw: vi.fn(),
    declineGameInvite: vi.fn(),
    inviteToGame: vi.fn(),
    offerDraw: vi.fn(),
    postPlayerChat: vi.fn(),
    postSpectatorChat: vi.fn(),
    resignGame: vi.fn(),
    submitGameAction: vi.fn(),
}));

vi.mock("../../api/endpoints/gameRoom", () => mocks);

const roomId = "11111111-1111-1111-1111-111111111111";
const allKey = ["gameRoom"];
const listKey = ["gameRoom", "list"];
const detailKey = ["gameRoom", "detail", roomId];
const spectatorChatKey = ["gameRoom", "spectator-chat", roomId];
const playerChatKey = ["gameRoom", "player-chat", roomId];

function room(status: string) {
    return { id: roomId, game_type: "chess", status };
}

function message(id: string, body: string) {
    return {
        id,
        user_id: "u-two",
        user: { id: "u-two", username: "beatrice", display_name: "Beatrice" },
        body,
        created_at: "2026-08-02T10:00:00.000Z",
    };
}

function chatHarness() {
    const queryClient = createTestQueryClient();
    queryClient.setDefaultOptions({
        queries: { retry: false, gcTime: Infinity, staleTime: 0, refetchOnWindowFocus: false },
        mutations: { retry: false },
    });

    return { queryClient, wrapper: providerWrapper({ queryClient }) };
}

function harness() {
    const queryClient = createTestQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");
    const setQueryData = vi.spyOn(queryClient, "setQueryData");

    return { invalidateQueries, queryClient, setQueryData, wrapper: providerWrapper({ queryClient }) };
}

beforeEach(() => {
    for (const fn of Object.values(mocks)) {
        fn.mockResolvedValue(room("active"));
    }
});

describe("useInviteToGame", () => {
    it("sends the opponent and the chosen game type as separate arguments", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useInviteToGame(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ opponentId: "battler", gameType: "othello" });
        });

        // then
        expect(mocks.inviteToGame).toHaveBeenCalledWith("battler", "othello");
    });

    it("refreshes every game room once the invite is sent", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useInviteToGame(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ opponentId: "battler", gameType: "chess" });
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: allKey });
    });

    it("leaves the cached game rooms alone when the invite is rejected", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.inviteToGame.mockRejectedValue(new Error("already playing"));
        const { result } = renderHook(() => useInviteToGame(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync({ opponentId: "battler", gameType: "chess" })).rejects.toThrow(
                "already playing",
            );
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useAcceptGameInvite", () => {
    it("accepts the invite it was handed", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useAcceptGameInvite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.acceptGameInvite).toHaveBeenCalledWith(roomId);
    });

    it("writes the accepted room straight into the detail cache", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        mocks.acceptGameInvite.mockResolvedValue(room("active"));
        const { result } = renderHook(() => useAcceptGameInvite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(setQueryData).toHaveBeenCalledExactlyOnceWith(detailKey, room("active"));
    });

    it("refreshes the game room lists so the accepted invite stops showing as pending", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useAcceptGameInvite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledExactlyOnceWith({ queryKey: listKey });
    });

    it("marks the cached game room list stale rather than the room it just wrote", async () => {
        // given
        const { wrapper, queryClient } = harness();
        queryClient.setQueryDefaults(allKey, { gcTime: Infinity });
        queryClient.setQueryData(["gameRoom", "list", {}], { rooms: [room("pending")], total: 1 });
        const { result } = renderHook(() => useAcceptGameInvite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(queryClient.getQueryState(["gameRoom", "list", {}])?.isInvalidated).toBe(true);
        expect(queryClient.getQueryState(detailKey)?.isInvalidated).toBe(false);
    });
});

describe("useDeclineGameInvite", () => {
    it("declines the invite it was handed and refreshes every game room", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeclineGameInvite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.declineGameInvite).toHaveBeenCalledWith(roomId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: allKey });
    });

    it("leaves the detail cache untouched", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        const { result } = renderHook(() => useDeclineGameInvite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(setQueryData).not.toHaveBeenCalled();
    });
});

describe("useCancelGameInvite", () => {
    it("cancels the invite it was handed and refreshes every game room", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useCancelGameInvite(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.cancelGameInvite).toHaveBeenCalledWith(roomId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: allKey });
    });
});

describe("useSubmitGameAction", () => {
    it("submits the action against the room the hook was built for", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useSubmitGameAction(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ type: "move", from: "e2", to: "e4" });
        });

        // then
        expect(mocks.submitGameAction).toHaveBeenCalledWith(roomId, { type: "move", from: "e2", to: "e4" });
    });

    it("replaces the cached room with the state the server returned", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        mocks.submitGameAction.mockResolvedValue(room("finished"));
        const { result } = renderHook(() => useSubmitGameAction(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ type: "move" });
        });

        // then
        expect(setQueryData).toHaveBeenCalledExactlyOnceWith(detailKey, room("finished"));
    });

    it("refreshes the game room lists so a finished game stops showing as active", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.submitGameAction.mockResolvedValue(room("finished"));
        const { result } = renderHook(() => useSubmitGameAction(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ type: "move" });
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledExactlyOnceWith({ queryKey: listKey });
    });

    it("keeps the cached room as it was when the action is refused", async () => {
        // given
        const { wrapper, setQueryData, invalidateQueries } = harness();
        mocks.submitGameAction.mockRejectedValue(new Error("not your turn"));
        const { result } = renderHook(() => useSubmitGameAction(roomId), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync({ type: "move" })).rejects.toThrow("not your turn");
        });

        // then
        expect(setQueryData).not.toHaveBeenCalled();
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useResignGame", () => {
    it("resigns the room it was handed and caches the finished room", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        mocks.resignGame.mockResolvedValue(room("finished"));
        const { result } = renderHook(() => useResignGame(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.resignGame).toHaveBeenCalledWith(roomId);
        expect(setQueryData).toHaveBeenCalledExactlyOnceWith(detailKey, room("finished"));
    });

    it("refreshes the game room lists so the resigned game stops showing as active", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.resignGame.mockResolvedValue(room("finished"));
        const { result } = renderHook(() => useResignGame(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledExactlyOnceWith({ queryKey: listKey });
    });
});

describe("useOfferDraw", () => {
    it("offers a draw in the room it was handed and caches the updated room", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        mocks.offerDraw.mockResolvedValue(room("active"));
        const { result } = renderHook(() => useOfferDraw(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.offerDraw).toHaveBeenCalledWith(roomId);
        expect(setQueryData).toHaveBeenCalledExactlyOnceWith(detailKey, room("active"));
    });
});

describe("useAcceptDraw", () => {
    it("accepts the draw in the room it was handed and caches the finished room", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        mocks.acceptDraw.mockResolvedValue(room("finished"));
        const { result } = renderHook(() => useAcceptDraw(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.acceptDraw).toHaveBeenCalledWith(roomId);
        expect(setQueryData).toHaveBeenCalledExactlyOnceWith(detailKey, room("finished"));
    });

    it("refreshes the game room lists so the drawn game stops showing as active", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.acceptDraw.mockResolvedValue(room("finished"));
        const { result } = renderHook(() => useAcceptDraw(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledExactlyOnceWith({ queryKey: listKey });
    });
});

describe("useDeclineDraw", () => {
    it("declines the draw in the room it was handed and caches the still active room", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        mocks.declineDraw.mockResolvedValue(room("active"));
        const { result } = renderHook(() => useDeclineDraw(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.declineDraw).toHaveBeenCalledWith(roomId);
        expect(setQueryData).toHaveBeenCalledExactlyOnceWith(detailKey, room("active"));
    });

    it("caches the room under the id that was passed to the mutation, not the one it was built with", async () => {
        // given
        const { wrapper, setQueryData } = harness();
        const otherId = "22222222-2222-2222-2222-222222222222";
        const otherRoom = { id: otherId, game_type: "chess", status: "active" };
        mocks.declineDraw.mockResolvedValue(otherRoom);
        const { result } = renderHook(() => useDeclineDraw(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(otherId);
        });

        // then
        expect(setQueryData).toHaveBeenCalledExactlyOnceWith(["gameRoom", "detail", otherId], otherRoom);
    });
});

describe("usePostSpectatorChat", () => {
    it("sends the body to the spectator chat of the room it was built with", async () => {
        // given
        const { wrapper } = chatHarness();
        mocks.postSpectatorChat.mockResolvedValue(message("m-sent", "hello"));
        const { result } = renderHook(() => usePostSpectatorChat(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("hello");
        });

        // then
        expect(mocks.postSpectatorChat).toHaveBeenCalledWith(roomId, "hello");
    });

    it("adds the sent message to the end of the cached spectator chat", async () => {
        // given
        const { wrapper, queryClient } = chatHarness();
        queryClient.setQueryData(spectatorChatKey, { messages: [message("m1", "the golden truth")] });
        mocks.postSpectatorChat.mockResolvedValue(message("m-sent", "hello"));
        const { result } = renderHook(() => usePostSpectatorChat(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("hello");
        });

        // then
        expect(queryClient.getQueryData(spectatorChatKey)).toEqual({
            messages: [message("m1", "the golden truth"), message("m-sent", "hello")],
        });
    });

    it("opens the cached spectator chat with the sent message when nothing was cached yet", async () => {
        // given
        const { wrapper, queryClient } = chatHarness();
        mocks.postSpectatorChat.mockResolvedValue(message("m-sent", "hello"));
        const { result } = renderHook(() => usePostSpectatorChat(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("hello");
        });

        // then
        expect(queryClient.getQueryData(spectatorChatKey)).toEqual({ messages: [message("m-sent", "hello")] });
    });

    it("never lets the same message land in the cached chat twice", async () => {
        // given
        const { wrapper, queryClient } = chatHarness();
        queryClient.setQueryData(spectatorChatKey, { messages: [message("m-sent", "hello")] });
        mocks.postSpectatorChat.mockResolvedValue(message("m-sent", "hello"));
        const { result } = renderHook(() => usePostSpectatorChat(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("hello");
        });

        // then
        expect(queryClient.getQueryData(spectatorChatKey)).toEqual({ messages: [message("m-sent", "hello")] });
    });

    it("leaves the cached spectator chat alone when the message is refused", async () => {
        // given
        const { wrapper, queryClient } = chatHarness();
        queryClient.setQueryData(spectatorChatKey, { messages: [message("m1", "the golden truth")] });
        mocks.postSpectatorChat.mockRejectedValue(new Error("you are timed out"));
        const { result } = renderHook(() => usePostSpectatorChat(roomId), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync("hello")).rejects.toThrow("you are timed out");
        });

        // then
        expect(queryClient.getQueryData(spectatorChatKey)).toEqual({
            messages: [message("m1", "the golden truth")],
        });
    });
});

describe("usePostPlayerChat", () => {
    it("sends the body to the player chat of the room it was built with", async () => {
        // given
        const { wrapper } = chatHarness();
        mocks.postPlayerChat.mockResolvedValue(message("m-sent", "your move"));
        const { result } = renderHook(() => usePostPlayerChat(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("your move");
        });

        // then
        expect(mocks.postPlayerChat).toHaveBeenCalledWith(roomId, "your move");
        expect(mocks.postSpectatorChat).not.toHaveBeenCalled();
    });

    it("adds the sent message to the cached player chat and never to the spectator one", async () => {
        // given
        const { wrapper, queryClient } = chatHarness();
        mocks.postPlayerChat.mockResolvedValue(message("m-sent", "your move"));
        const { result } = renderHook(() => usePostPlayerChat(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("your move");
        });

        // then
        expect(queryClient.getQueryData(playerChatKey)).toEqual({ messages: [message("m-sent", "your move")] });
        expect(queryClient.getQueryData(spectatorChatKey)).toBeUndefined();
    });
});
