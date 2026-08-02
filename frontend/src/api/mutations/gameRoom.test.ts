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
    resignGame: vi.fn(),
    submitGameAction: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const roomId = "11111111-1111-1111-1111-111111111111";
const allKey = ["gameRoom"];
const listKey = ["gameRoom", "list"];
const detailKey = ["gameRoom", "detail", roomId];

function room(status: string) {
    return { id: roomId, game_type: "chess", status };
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
