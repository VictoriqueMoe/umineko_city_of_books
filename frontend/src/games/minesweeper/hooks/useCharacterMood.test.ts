import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MinesweeperState } from "../../../types/api";
import { CharacterDef, CharacterId, Expression, Mood } from "../types";
import { useCharacterMood } from "./useCharacterMood";

const ALL_MOODS: Mood[] = [
    "default",
    "neutral",
    "happy",
    "very_happy",
    "smirk",
    "worried",
    "sweating",
    "angry",
    "furious",
    "surprised",
    "relieved",
    "bored",
    "wink",
    "win",
    "lose",
];

function moodCharacter(id: CharacterId): CharacterDef {
    const expressions: Partial<Record<Mood, Expression>> = {};
    for (const mood of ALL_MOODS) {
        expressions[mood] = { image: `${id}:${mood}`, facing: "center" };
    }

    return { id, name: id, image: `${id}:base`, expressions };
}

const me = moodCharacter(CharacterId.Bernkastel);
const them = moodCharacter(CharacterId.Erika);

function moodOf(expression: Expression): string {
    return expression.image.split(":")[1] ?? "";
}

function makeState(overrides: Partial<MinesweeperState> = {}): MinesweeperState {
    return {
        phase: "playing",
        width: 10,
        height: 11,
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
    mySlot: number;
    myCharacter: CharacterDef | undefined;
    opponentCharacter: CharacterDef | undefined;
}

function setup(overrides: Partial<HookProps> = {}) {
    const initialProps: HookProps = {
        state: makeState(),
        mySlot: 0,
        myCharacter: me,
        opponentCharacter: them,
        ...overrides,
    };

    return renderHook(props => useCharacterMood(props), { initialProps });
}

function advance(ms: number) {
    act(() => {
        vi.advanceTimersByTime(ms);
    });
}

describe("useCharacterMood progress moods", () => {
    const cases: { lead: number; my: Mood; op: Mood }[] = [
        { lead: 16, my: "very_happy", op: "angry" },
        { lead: 15, my: "smirk", op: "sweating" },
        { lead: 9, my: "smirk", op: "sweating" },
        { lead: 8, my: "happy", op: "worried" },
        { lead: 4, my: "happy", op: "worried" },
        { lead: 3, my: "neutral", op: "neutral" },
        { lead: 0, my: "neutral", op: "neutral" },
        { lead: -2, my: "neutral", op: "neutral" },
        { lead: -3, my: "worried", op: "happy" },
        { lead: -7, my: "worried", op: "happy" },
        { lead: -8, my: "sweating", op: "smirk" },
        { lead: -14, my: "sweating", op: "smirk" },
        { lead: -15, my: "angry", op: "very_happy" },
        { lead: -24, my: "angry", op: "very_happy" },
        { lead: -25, my: "furious", op: "very_happy" },
    ];

    for (const testCase of cases) {
        it(`looks ${testCase.my} at a lead of ${testCase.lead} out of a hundred safe cells`, () => {
            // given
            const state = makeState({ revealed_count: [40 + testCase.lead, 40] });

            // when
            const { result } = setup({ state });

            // then
            expect(moodOf(result.current.myExpr)).toBe(testCase.my);
            expect(moodOf(result.current.opExpr)).toBe(testCase.op);
        });
    }

    it("reads the second slot as the player's own progress", () => {
        // given
        const state = makeState({ revealed_count: [40, 56] });

        // when
        const { result } = setup({ state, mySlot: 1 });

        // then
        expect(moodOf(result.current.myExpr)).toBe("very_happy");
        expect(moodOf(result.current.opExpr)).toBe("angry");
    });

    it("stays neutral when the board has no safe cells to clear", () => {
        // given
        const state = makeState({ width: 4, height: 4, mine_count: 16, revealed_count: [0, 0] });

        // when
        const { result } = setup({ state });

        // then
        expect(moodOf(result.current.myExpr)).toBe("neutral");
        expect(moodOf(result.current.opExpr)).toBe("neutral");
    });
});

describe("useCharacterMood outside the playing phase", () => {
    it("shows the default pose while the characters are still being chosen", () => {
        // given
        const state = makeState({ phase: "char_select", revealed_count: [0, 0] });

        // when
        const { result } = setup({ state });

        // then
        expect(moodOf(result.current.myExpr)).toBe("default");
        expect(moodOf(result.current.opExpr)).toBe("default");
    });

    it("shows the default pose when there is no game state yet", () => {
        // given
        const state = null;

        // when
        const { result } = setup({ state });

        // then
        expect(moodOf(result.current.myExpr)).toBe("default");
        expect(moodOf(result.current.opExpr)).toBe("default");
    });

    it("celebrates when the player holds the winning slot", () => {
        // given
        const state = makeState({ phase: "finished", winner_slot: 0, revealed_count: [90, 40] });

        // when
        const { result } = setup({ state, mySlot: 0 });

        // then
        expect(moodOf(result.current.myExpr)).toBe("win");
        expect(moodOf(result.current.opExpr)).toBe("lose");
    });

    it("sulks when the other slot took the win", () => {
        // given
        const state = makeState({ phase: "finished", winner_slot: 1, revealed_count: [90, 40] });

        // when
        const { result } = setup({ state, mySlot: 0 });

        // then
        expect(moodOf(result.current.myExpr)).toBe("lose");
        expect(moodOf(result.current.opExpr)).toBe("win");
    });

    it("shows the default pose when a finished game has no winner recorded", () => {
        // given
        const state = makeState({ phase: "finished", revealed_count: [50, 50] });

        // when
        const { result } = setup({ state });

        // then
        expect(moodOf(result.current.myExpr)).toBe("default");
        expect(moodOf(result.current.opExpr)).toBe("default");
    });

    it("returns an empty expression for a character that has not been picked", () => {
        // given
        const state = makeState({ revealed_count: [56, 40] });

        // when
        const { result } = setup({ state, myCharacter: undefined, opponentCharacter: undefined });

        // then
        expect(result.current.myExpr).toEqual({ image: "", facing: "center" });
        expect(result.current.opExpr).toEqual({ image: "", facing: "center" });
    });

    it("falls back to the default pose when the game leaves the playing phase", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [56, 40] }) });
        expect(moodOf(result.current.myExpr)).toBe("very_happy");

        // when
        rerender({
            state: makeState({ phase: "char_select", revealed_count: [0, 0] }),
            mySlot: 0,
            myCharacter: me,
            opponentCharacter: them,
        });

        // then
        expect(moodOf(result.current.myExpr)).toBe("default");
        expect(moodOf(result.current.opExpr)).toBe("default");
    });
});

describe("useCharacterMood reactions", () => {
    function rerenderWith(
        rerender: (props: HookProps) => void,
        counts: [number, number],
        overrides: Partial<MinesweeperState> = {},
    ) {
        rerender({
            state: makeState({ revealed_count: counts, ...overrides }),
            mySlot: 0,
            myCharacter: me,
            opponentCharacter: them,
        });
    }

    it("looks surprised when the opponent uncovers a big chunk in one go", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 10] }) });

        // when
        rerenderWith(rerender, [10, 17]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("surprised");
        expect(moodOf(result.current.opExpr)).toBe("surprised");
    });

    it("ignores an opponent reveal that is not big enough to startle", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 10] }) });

        // when
        rerenderWith(rerender, [10, 15]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("worried");
        expect(moodOf(result.current.opExpr)).toBe("happy");
    });

    it("does not startle before the player has revealed anything at all", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [0, 0] }) });

        // when
        rerenderWith(rerender, [0, 10]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("sweating");
        expect(moodOf(result.current.opExpr)).toBe("smirk");
    });

    it("looks relieved when the player retakes the lead after falling behind", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 20] }) });

        // when
        rerenderWith(rerender, [21, 20]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("relieved");
        expect(moodOf(result.current.opExpr)).toBe("worried");
    });

    it("does not feel relief while it is still behind", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 20] }) });

        // when
        rerenderWith(rerender, [15, 20]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("worried");
        expect(moodOf(result.current.opExpr)).toBe("happy");
    });

    it("prefers surprise over relief when both would apply at once", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 20] }) });

        // when
        rerenderWith(rerender, [40, 30]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("surprised");
    });

    it("drops the reaction and returns to the running score after a couple of seconds", () => {
        // given
        vi.useFakeTimers();
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 10] }) });
        rerenderWith(rerender, [10, 17]);
        expect(moodOf(result.current.myExpr)).toBe("surprised");

        // when
        advance(2500);

        // then
        expect(moodOf(result.current.myExpr)).toBe("worried");
        expect(moodOf(result.current.opExpr)).toBe("happy");
    });

    it("keeps the reaction on screen until its time is up", () => {
        // given
        vi.useFakeTimers();
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 10] }) });
        rerenderWith(rerender, [10, 17]);

        // when
        advance(2499);

        // then
        expect(moodOf(result.current.myExpr)).toBe("surprised");
    });

    it("gives a fresh two and a half seconds to a reaction that replaces another", () => {
        // given
        vi.useFakeTimers();
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 20] }) });
        rerenderWith(rerender, [10, 27]);
        expect(moodOf(result.current.myExpr)).toBe("surprised");

        // when
        advance(2000);
        rerenderWith(rerender, [40, 27]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("relieved");
        advance(2000);
        expect(moodOf(result.current.myExpr)).toBe("relieved");
        advance(500);
        expect(moodOf(result.current.myExpr)).toBe("smirk");
    });

    it("clears a pending reaction when the game finishes", () => {
        // given
        vi.useFakeTimers();
        const { result, rerender } = setup({ state: makeState({ revealed_count: [10, 10] }) });
        rerenderWith(rerender, [10, 17]);

        // when
        rerenderWith(rerender, [10, 17], { phase: "finished", winner_slot: 1 });

        // then
        expect(moodOf(result.current.myExpr)).toBe("lose");
        expect(moodOf(result.current.opExpr)).toBe("win");
    });
});

describe("useCharacterMood idling", () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    function rerenderWith(rerender: (props: HookProps) => void, counts: [number, number]) {
        rerender({
            state: makeState({ revealed_count: counts }),
            mySlot: 0,
            myCharacter: me,
            opponentCharacter: them,
        });
    }

    it("settles back to the default pose after a long spell of no progress", () => {
        // given
        const { result } = setup({ state: makeState({ revealed_count: [56, 40] }) });
        expect(moodOf(result.current.myExpr)).toBe("very_happy");

        // when
        advance(8000);

        // then
        expect(moodOf(result.current.myExpr)).toBe("default");
        expect(moodOf(result.current.opExpr)).toBe("default");
    });

    it("restarts the idle countdown every time the board moves", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [56, 40] }) });
        advance(5000);

        // when
        rerenderWith(rerender, [58, 40]);
        advance(5000);

        // then
        expect(moodOf(result.current.myExpr)).toBe("very_happy");
    });

    it("wakes from the idle default as soon as the board moves again", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [56, 40] }) });
        advance(8000);
        expect(moodOf(result.current.myExpr)).toBe("default");

        // when
        rerenderWith(rerender, [58, 40]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("very_happy");
        expect(moodOf(result.current.opExpr)).toBe("angry");
    });

    it("settles back to the default pose again after a second idle spell", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [56, 40] }) });
        advance(8000);
        rerenderWith(rerender, [58, 40]);
        expect(moodOf(result.current.myExpr)).toBe("very_happy");

        // when
        advance(8000);

        // then
        expect(moodOf(result.current.myExpr)).toBe("default");
        expect(moodOf(result.current.opExpr)).toBe("default");
    });

    it("wakes from the idle default when the opponent surges ahead", () => {
        // given
        const { result, rerender } = setup({ state: makeState({ revealed_count: [56, 40] }) });
        advance(8000);
        expect(moodOf(result.current.myExpr)).toBe("default");

        // when
        rerenderWith(rerender, [56, 47]);

        // then
        expect(moodOf(result.current.myExpr)).toBe("surprised");
        advance(2500);
        expect(moodOf(result.current.myExpr)).toBe("smirk");
    });

    it("clears every pending timer when it unmounts", () => {
        // given
        const { rerender, unmount } = setup({ state: makeState({ revealed_count: [10, 10] }) });
        rerenderWith(rerender, [10, 17]);
        expect(vi.getTimerCount()).toBeGreaterThan(0);

        // when
        unmount();

        // then
        expect(vi.getTimerCount()).toBe(0);
    });
});
