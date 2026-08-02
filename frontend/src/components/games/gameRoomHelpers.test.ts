import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { GameRoom, GameRoomPlayer } from "../../types/api";
import {
    formatDuration,
    gameResultLabel,
    getMySlot,
    performResignWithConfirm,
    useDisconnectForfeit,
} from "./gameRoomHelpers";

const NOW = new Date("2026-08-02T12:00:00.000Z");

function makePlayer(overrides: Partial<GameRoomPlayer> = {}): GameRoomPlayer {
    const id = overrides.user_id ?? "player-0";
    return {
        user_id: id,
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        role: "player",
        slot: 0,
        joined: true,
        connected: true,
        user: { id, username: "battler", display_name: "Battler" },
        ...overrides,
    };
}

function makeRoom(overrides: Partial<GameRoom> = {}): GameRoom {
    return {
        id: "room-1",
        game_type: "chess",
        status: "active",
        state: {},
        created_by: "player-0",
        created_at: "2026-08-02T11:58:00.000Z",
        updated_at: "2026-08-02T11:58:00.000Z",
        players: [],
        watcher_count: 0,
        ...overrides,
    };
}

describe("getMySlot", () => {
    it("has no slot for a logged out viewer", () => {
        // given
        const room = makeRoom({ players: [makePlayer({ user_id: "a", slot: 0 })] });

        // when
        const slot = getMySlot(room, null);

        // then
        expect(slot).toBeNull();
    });

    it("has no slot for someone who is only watching", () => {
        // given
        const room = makeRoom({ players: [makePlayer({ user_id: "a", slot: 0 })] });

        // when
        const slot = getMySlot(room, "spectator");

        // then
        expect(slot).toBeNull();
    });

    it("finds the zero slot rather than treating it as missing", () => {
        // given
        const room = makeRoom({
            players: [makePlayer({ user_id: "a", slot: 0 }), makePlayer({ user_id: "b", slot: 1 })],
        });

        // when
        const slot = getMySlot(room, "a");

        // then
        expect(slot).toBe(0);
    });

    it("finds the second slot", () => {
        // given
        const room = makeRoom({
            players: [makePlayer({ user_id: "a", slot: 0 }), makePlayer({ user_id: "b", slot: 1 })],
        });

        // when
        const slot = getMySlot(room, "b");

        // then
        expect(slot).toBe(1);
    });
});

describe("formatDuration", () => {
    it("shows a dash when no time has passed", () => {
        // given
        const seconds = 0;

        // when
        const text = formatDuration(seconds);

        // then
        expect(text).toBe("-");
    });

    it("shows a dash for a negative duration", () => {
        // given
        const seconds = -30;

        // when
        const text = formatDuration(seconds);

        // then
        expect(text).toBe("-");
    });

    it("shows only seconds below a minute", () => {
        // given
        const seconds = 45;

        // when
        const text = formatDuration(seconds);

        // then
        expect(text).toBe("45s");
    });

    it("shows minutes and seconds below an hour", () => {
        // given
        const seconds = 330;

        // when
        const text = formatDuration(seconds);

        // then
        expect(text).toBe("5m 30s");
    });

    it("keeps a zero seconds part on a whole number of minutes", () => {
        // given
        const seconds = 60;

        // when
        const text = formatDuration(seconds);

        // then
        expect(text).toBe("1m 0s");
    });

    it("drops the seconds once there is an hour to show", () => {
        // given
        const seconds = 3600 * 2 + 60 * 5 + 9;

        // when
        const text = formatDuration(seconds);

        // then
        expect(text).toBe("2h 5m");
    });

    it("keeps a zero minutes part on a whole number of hours", () => {
        // given
        const seconds = 3600;

        // when
        const text = formatDuration(seconds);

        // then
        expect(text).toBe("1h 0m");
    });
});

describe("gameResultLabel", () => {
    const players = [
        makePlayer({ user_id: "a", slot: 0, display_name: "Battler" }),
        makePlayer({ user_id: "b", slot: 1, display_name: "Beatrice" }),
    ];

    it("says nothing while the game has not started", () => {
        // given
        const room = makeRoom({ status: "pending", players });

        // when
        const result = gameResultLabel(room, "a", false);

        // then
        expect(result).toEqual({ text: "", tone: "neutral" });
    });

    it("says nothing while the game is being played", () => {
        // given
        const room = makeRoom({ status: "active", players, winner_user_id: "a" });

        // when
        const result = gameResultLabel(room, "a", false);

        // then
        expect(result).toEqual({ text: "", tone: "neutral" });
    });

    it("reports a cancelled game when the room timed out", () => {
        // given
        const room = makeRoom({ status: "abandoned", players, result: "timeout", winner_user_id: "a" });

        // when
        const result = gameResultLabel(room, "a", false);

        // then
        expect(result).toEqual({ text: "Game cancelled", tone: "draw" });
    });

    it("reports a draw when the finished game has no winner", () => {
        // given
        const room = makeRoom({ status: "finished", players, result: "draw" });

        // when
        const result = gameResultLabel(room, "a", false);

        // then
        expect(result).toEqual({ text: "Draw", tone: "draw" });
    });

    it("names the winner for a spectator", () => {
        // given
        const room = makeRoom({ status: "finished", players, winner_user_id: "b" });

        // when
        const result = gameResultLabel(room, "a", true);

        // then
        expect(result).toEqual({ text: "Beatrice won", tone: "neutral" });
    });

    it("names the winner for a logged out viewer", () => {
        // given
        const room = makeRoom({ status: "finished", players, winner_user_id: "a" });

        // when
        const result = gameResultLabel(room, null, false);

        // then
        expect(result).toEqual({ text: "Battler won", tone: "neutral" });
    });

    it("falls back to a question mark when the winner is not in the player list", () => {
        // given
        const room = makeRoom({ status: "finished", players, winner_user_id: "ghost" });

        // when
        const result = gameResultLabel(room, null, false);

        // then
        expect(result).toEqual({ text: "? won", tone: "neutral" });
    });

    it("congratulates the player who won", () => {
        // given
        const room = makeRoom({ status: "finished", players, winner_user_id: "a" });

        // when
        const result = gameResultLabel(room, "a", false);

        // then
        expect(result).toEqual({ text: "You won", tone: "win" });
    });

    it("tells the other player they lost", () => {
        // given
        const room = makeRoom({ status: "finished", players, winner_user_id: "a" });

        // when
        const result = gameResultLabel(room, "b", false);

        // then
        expect(result).toEqual({ text: "You lost", tone: "loss" });
    });

    it("reads the result of an abandoned game the same way", () => {
        // given
        const room = makeRoom({ status: "abandoned", players, winner_user_id: "b", result: "forfeit" });

        // when
        const result = gameResultLabel(room, "b", false);

        // then
        expect(result).toEqual({ text: "You won", tone: "win" });
    });
});

describe("performResignWithConfirm", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("does nothing when the confirmation is dismissed", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const onResign = vi.fn(() => Promise.resolve());
        const setSubmitting = vi.fn();
        const setError = vi.fn();

        // when
        await performResignWithConfirm(onResign, setSubmitting, setError);

        // then
        expect(onResign).not.toHaveBeenCalled();
        expect(setSubmitting).not.toHaveBeenCalled();
        expect(setError).not.toHaveBeenCalled();
    });

    it("clears the error and lowers the submitting flag after a successful resign", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const onResign = vi.fn(() => Promise.resolve());
        const setSubmitting = vi.fn();
        const setError = vi.fn();

        // when
        await performResignWithConfirm(onResign, setSubmitting, setError);

        // then
        expect(onResign).toHaveBeenCalledOnce();
        expect(setSubmitting.mock.calls).toEqual([[true], [false]]);
        expect(setError).toHaveBeenCalledExactlyOnceWith("");
    });

    it("surfaces the message of a failed resign", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const onResign = vi.fn(() => Promise.reject(new Error("not your turn")));
        const setSubmitting = vi.fn();
        const setError = vi.fn();

        // when
        await performResignWithConfirm(onResign, setSubmitting, setError);

        // then
        expect(setError).toHaveBeenNthCalledWith(2, "not your turn");
        expect(setSubmitting.mock.calls).toEqual([[true], [false]]);
    });

    it("falls back to a generic message when the failure is not an error", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const onResign = vi.fn(() => Promise.reject("catbox"));
        const setSubmitting = vi.fn();
        const setError = vi.fn();

        // when
        await performResignWithConfirm(onResign, setSubmitting, setError);

        // then
        expect(setError).toHaveBeenNthCalledWith(2, "Resign failed");
        expect(setSubmitting).toHaveBeenLastCalledWith(false);
    });
});

describe("useDisconnectForfeit", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(NOW);
    });

    it("reports nobody offline while every player is connected", () => {
        // given
        const room = makeRoom({
            players: [makePlayer({ user_id: "a", slot: 0 }), makePlayer({ user_id: "b", slot: 1 })],
        });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.offlinePlayer).toBeUndefined();
        expect(result.current.forfeitRemaining).toBeNull();
        expect(result.current.now).toBe(NOW.getTime());
    });

    it("counts down the grace period for a player who dropped out", () => {
        // given
        const room = makeRoom({
            players: [
                makePlayer({ user_id: "a", slot: 0 }),
                makePlayer({
                    user_id: "b",
                    slot: 1,
                    connected: false,
                    disconnected_at: "2026-08-02T11:59:50.000Z",
                }),
            ],
        });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.offlinePlayer?.user_id).toBe("b");
        expect(result.current.forfeitRemaining).toBe(50);
    });

    it("clamps the countdown at zero once the grace period has run out", () => {
        // given
        const room = makeRoom({
            players: [
                makePlayer({
                    user_id: "b",
                    slot: 1,
                    connected: false,
                    disconnected_at: "2026-08-02T11:58:00.000Z",
                }),
            ],
        });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.forfeitRemaining).toBe(0);
    });

    it("ticks the countdown down as the seconds pass", () => {
        // given
        const room = makeRoom({
            players: [
                makePlayer({
                    user_id: "b",
                    slot: 1,
                    connected: false,
                    disconnected_at: "2026-08-02T11:59:50.000Z",
                }),
            ],
        });
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // when
        act(() => {
            vi.advanceTimersByTime(4000);
        });

        // then
        expect(result.current.forfeitRemaining).toBe(46);
    });

    it("ignores a disconnection once the game is no longer active", () => {
        // given
        const room = makeRoom({
            status: "finished",
            finished_at: "2026-08-02T11:59:00.000Z",
            players: [
                makePlayer({
                    user_id: "b",
                    slot: 1,
                    connected: false,
                    disconnected_at: "2026-08-02T11:59:50.000Z",
                }),
            ],
        });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.offlinePlayer).toBeUndefined();
        expect(result.current.forfeitRemaining).toBeNull();
    });

    it("ignores a player who is offline without a disconnection time", () => {
        // given
        const room = makeRoom({ players: [makePlayer({ user_id: "b", slot: 1, connected: false })] });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.offlinePlayer).toBeUndefined();
        expect(result.current.forfeitRemaining).toBeNull();
    });

    it("gives up on a disconnection time it cannot parse", () => {
        // given
        const room = makeRoom({
            players: [makePlayer({ user_id: "b", slot: 1, connected: false, disconnected_at: "not a date" })],
        });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.offlinePlayer?.user_id).toBe("b");
        expect(result.current.forfeitRemaining).toBeNull();
    });

    it("measures the live duration from the moment the room was created", () => {
        // given
        const room = makeRoom({ created_at: "2026-08-02T11:58:00.000Z" });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.liveDurationSeconds).toBe(120);
    });

    it("grows the live duration while the game is still active", () => {
        // given
        const room = makeRoom({ created_at: "2026-08-02T11:58:00.000Z" });
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // when
        act(() => {
            vi.advanceTimersByTime(3000);
        });

        // then
        expect(result.current.liveDurationSeconds).toBe(123);
        expect(result.current.now).toBe(NOW.getTime() + 3000);
    });

    it("freezes the duration at the finishing time once the game is over", () => {
        // given
        const room = makeRoom({
            status: "finished",
            created_at: "2026-08-02T11:58:00.000Z",
            finished_at: "2026-08-02T11:59:00.000Z",
        });
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // when
        act(() => {
            vi.advanceTimersByTime(10000);
        });

        // then
        expect(result.current.liveDurationSeconds).toBe(60);
        expect(vi.getTimerCount()).toBe(0);
    });

    it("reports a zero duration when the creation time cannot be parsed", () => {
        // given
        const room = makeRoom({ status: "finished", created_at: "who knows" });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.liveDurationSeconds).toBe(0);
    });

    it("reports a zero duration when the finishing time cannot be parsed", () => {
        // given
        const room = makeRoom({ status: "finished", finished_at: "sometime" });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.liveDurationSeconds).toBe(0);
    });

    it("never reports a negative duration for a room created in the future", () => {
        // given
        const room = makeRoom({ status: "finished", created_at: "2026-08-02T12:05:00.000Z" });

        // when
        const { result } = renderHook(() => useDisconnectForfeit(room));

        // then
        expect(result.current.liveDurationSeconds).toBe(0);
    });
});
