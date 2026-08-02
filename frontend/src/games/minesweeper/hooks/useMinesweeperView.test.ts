import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { MinesweeperState } from "../../../types/api";
import { useMinesweeperView } from "./useMinesweeperView";

function makeState(overrides: Partial<MinesweeperState> = {}): MinesweeperState {
    return {
        phase: "playing",
        width: 10,
        height: 10,
        mine_count: 10,
        characters: ["bernkastel", "erika"],
        revealed: [[], []],
        flagged: [[], []],
        revealed_count: [0, 0],
        values: [[], []],
        mines_placed: true,
        pending_clicks: [null, null],
        ...overrides,
    };
}

interface HookProps {
    state: MinesweeperState | null;
    roomFinished: boolean;
}

function setup(overrides: Partial<HookProps> = {}) {
    const initialProps: HookProps = { state: makeState(), roomFinished: false, ...overrides };

    return renderHook(props => useMinesweeperView(props.state, props.roomFinished), { initialProps });
}

describe("useMinesweeperView", () => {
    it("waits on the character select screen while there is no game state yet", () => {
        // given
        const props = { state: null, roomFinished: false };

        // when
        const { result } = setup(props);

        // then
        expect(result.current.clientPhase).toBe("char_select");
        expect(result.current.introPlayed).toBe(false);
    });

    it("shows the character select screen while the players are still choosing", () => {
        // given
        const state = makeState({ phase: "char_select", mines_placed: false });

        // when
        const { result } = setup({ state });

        // then
        expect(result.current.clientPhase).toBe("char_select");
    });

    it("plays the versus intro as soon as the game starts", () => {
        // given
        const state = makeState({ phase: "playing" });

        // when
        const { result } = setup({ state });

        // then
        expect(result.current.clientPhase).toBe("vs_intro");
        expect(result.current.introPlayed).toBe(false);
    });

    it("moves to the board once the intro has been marked as played", () => {
        // given
        const { result } = setup();
        expect(result.current.clientPhase).toBe("vs_intro");

        // when
        act(() => {
            result.current.markIntroPlayed();
        });

        // then
        expect(result.current.clientPhase).toBe("playing");
        expect(result.current.introPlayed).toBe(true);
    });

    it("shows the finished screen when the game state reports it is over", () => {
        // given
        const state = makeState({ phase: "finished", winner_slot: 0 });

        // when
        const { result } = setup({ state });

        // then
        expect(result.current.clientPhase).toBe("finished");
    });

    it("shows the finished screen when the room itself is already over", () => {
        // given
        const state = makeState({ phase: "playing" });

        // when
        const { result } = setup({ state, roomFinished: true });

        // then
        expect(result.current.clientPhase).toBe("finished");
    });

    it("treats the intro as already seen when the room was finished before it opened", () => {
        // given
        const state = makeState({ phase: "playing" });

        // when
        const { result } = setup({ state, roomFinished: true });

        // then
        expect(result.current.introPlayed).toBe(true);
    });

    it("plays the intro for a game that only starts after the first render", () => {
        // given
        const { result, rerender } = setup({ state: null });
        expect(result.current.clientPhase).toBe("char_select");

        // when
        rerender({ state: makeState({ phase: "playing" }), roomFinished: false });

        // then
        expect(result.current.clientPhase).toBe("vs_intro");
    });

    it("replays the intro for a rematch that goes back to character select", () => {
        // given
        const { result, rerender } = setup();
        act(() => {
            result.current.markIntroPlayed();
        });
        expect(result.current.clientPhase).toBe("playing");

        // when
        rerender({ state: makeState({ phase: "char_select" }), roomFinished: false });
        rerender({ state: makeState({ phase: "playing" }), roomFinished: false });

        // then
        expect(result.current.introPlayed).toBe(false);
        expect(result.current.clientPhase).toBe("vs_intro");
    });

    it("keeps the intro marked as played while a game merely ends", () => {
        // given
        const { result, rerender } = setup();
        act(() => {
            result.current.markIntroPlayed();
        });

        // when
        rerender({ state: makeState({ phase: "finished", winner_slot: 1 }), roomFinished: false });

        // then
        expect(result.current.introPlayed).toBe(true);
        expect(result.current.clientPhase).toBe("finished");
    });

    it("does not rewind the intro of a finished room that returns to character select", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ phase: "playing" }), roomFinished: true });
        expect(result.current.introPlayed).toBe(true);

        // when
        rerender({ state: makeState({ phase: "char_select" }), roomFinished: true });

        // then
        expect(result.current.introPlayed).toBe(true);
        expect(result.current.clientPhase).toBe("finished");
    });

    it("keeps the same markIntroPlayed callback across renders", () => {
        // given
        const { result, rerender } = setup();
        const first = result.current.markIntroPlayed;

        // when
        rerender({ state: makeState({ phase: "playing" }), roomFinished: false });

        // then
        expect(result.current.markIntroPlayed).toBe(first);
    });
});
