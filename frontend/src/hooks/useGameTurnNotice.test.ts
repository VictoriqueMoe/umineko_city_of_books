import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { dispatch } from "../api/realtime/bus";
import type { RealtimeEvent } from "../api/realtime/events";
import { useGameTurnNotice } from "./useGameTurnNotice";

class StubNotification {
    static permission = "granted";
    static instances: StubNotification[] = [];

    readonly close = vi.fn();
    onclick: (() => void) | null = null;

    constructor(
        readonly title: string,
        readonly options: NotificationOptions = {},
    ) {
        StubNotification.instances.push(this);
    }
}

function yourTurn(roomId: string, gameType = "chess"): RealtimeEvent {
    return { type: "game_your_turn", data: { room_id: roomId, game_type: gameType } } as unknown as RealtimeEvent;
}

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

beforeEach(() => {
    StubNotification.instances = [];
    StubNotification.permission = "granted";
    vi.stubGlobal("Notification", StubNotification);
});

afterEach(() => {
    vi.restoreAllMocks();
    window.history.replaceState({}, "", "/");
});

describe("useGameTurnNotice", () => {
    it("raises a desktop notification for your turn when the tab is not focused", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        renderHook(() => useGameTurnNotice("r1"));

        // when
        emit(yourTurn("r1"));

        // then
        expect(StubNotification.instances).toHaveLength(1);
        expect(StubNotification.instances[0].title).toBe("Your move in chess");
        expect(StubNotification.instances[0].options.tag).toBe("game-turn-r1");
        expect(StubNotification.instances[0].options.icon).toBe("/favicon/android-chrome-192x192.png");
    });

    it("stays quiet about your turn while the tab is focused", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(true);
        renderHook(() => useGameTurnNotice("r1"));

        // when
        emit(yourTurn("r1"));

        // then
        expect(StubNotification.instances).toEqual([]);
    });

    it("stays quiet about your turn when notification permission was never granted", () => {
        // given
        StubNotification.permission = "default";
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        renderHook(() => useGameTurnNotice("r1"));

        // when
        emit(yourTurn("r1"));

        // then
        expect(StubNotification.instances).toEqual([]);
    });

    it("stays quiet about a turn taken in another room", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        renderHook(() => useGameTurnNotice("r1"));

        // when
        emit(yourTurn("r2"));

        // then
        expect(StubNotification.instances).toEqual([]);
    });

    it("stays quiet when no room is being watched", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        renderHook(() => useGameTurnNotice(undefined));

        // when
        emit(yourTurn("r1"));

        // then
        expect(StubNotification.instances).toEqual([]);
    });

    it("names the game the event carried in the title", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        renderHook(() => useGameTurnNotice("r1"));

        // when
        emit(yourTurn("r1", "othello"));

        // then
        expect(StubNotification.instances[0].title).toBe("Your move in othello");
    });

    it("stops listening once it is unmounted", () => {
        // given
        vi.spyOn(document, "hasFocus").mockReturnValue(false);
        const view = renderHook(() => useGameTurnNotice("r1"));

        // when
        view.unmount();
        emit(yourTurn("r1"));

        // then
        expect(StubNotification.instances).toEqual([]);
    });
});
