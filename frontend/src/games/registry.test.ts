import { describe, expect, it } from "vitest";
import type { GameType } from "../types/api";
import { GAME_TYPES, gameTypeFor, gameTypeLabel } from "./registry";

const EXPECTED: { type: GameType; label: string }[] = [
    { type: "chess", label: "Chess" },
    { type: "checkers", label: "Checkers" },
    { type: "othello", label: "Othello" },
    { type: "minesweeper", label: "Minesweeper" },
    { type: "snakes_and_ladders", label: "Snakes & Ladders" },
];

describe("GAME_TYPES", () => {
    it("lists every game the site offers, in the order the hub shows them", () => {
        // given
        const expected = EXPECTED.map(entry => entry.type);

        // when
        const listed = GAME_TYPES.map(entry => entry.type);

        // then
        expect(listed).toEqual(expected);
    });

    it("holds one entry per game type", () => {
        // given
        const listed = GAME_TYPES.map(entry => entry.type);

        // when
        const unique = new Set(listed);

        // then
        expect(unique.size).toBe(listed.length);
    });

    it("marks every listed game as playable", () => {
        // given
        const listed = GAME_TYPES;

        // when
        const unavailable = listed.filter(entry => !entry.available);

        // then
        expect(unavailable).toEqual([]);
    });

    it("gives every game a label, a tagline and a set of how to play steps", () => {
        // given
        const listed = GAME_TYPES;

        // when
        const steps = listed.map(entry => entry.howToPlay ?? []);

        // then
        for (const entry of listed) {
            expect(entry.label.length).toBeGreaterThan(0);
            expect(entry.tagline.length).toBeGreaterThan(0);
        }
        for (const stepList of steps) {
            expect(stepList.length).toBeGreaterThan(0);
            for (const step of stepList) {
                expect(step.trim().length).toBeGreaterThan(0);
            }
        }
    });

    for (const expected of EXPECTED) {
        it(`routes ${expected.label} through its own hub, new and detail paths`, () => {
            // given
            const entry = GAME_TYPES.find(candidate => candidate.type === expected.type);

            // when
            const detail = entry?.detailPath("abc-123");

            // then
            expect(entry?.label).toBe(expected.label);
            expect(entry?.hubPath).toBe(`/games/${expected.type}`);
            expect(entry?.newPath).toBe(`/games/${expected.type}/new`);
            expect(detail).toBe(`/games/${expected.type}/abc-123`);
        });
    }
});

describe("gameTypeLabel", () => {
    for (const expected of EXPECTED) {
        it(`gives the display label for ${expected.type}`, () => {
            // given
            const type = expected.type;

            // when
            const label = gameTypeLabel(type);

            // then
            expect(label).toBe(expected.label);
        });
    }

    it("hands back an unrecognised game type unchanged", () => {
        // given
        const type = "mahjong";

        // when
        const label = gameTypeLabel(type);

        // then
        expect(label).toBe("mahjong");
    });

    it("hands back an empty game type unchanged", () => {
        // given
        const type = "";

        // when
        const label = gameTypeLabel(type);

        // then
        expect(label).toBe("");
    });

    it("does not match on a label rather than a type", () => {
        // given
        const type = "Chess";

        // when
        const label = gameTypeLabel(type);

        // then
        expect(label).toBe("Chess");
    });
});

describe("gameTypeFor", () => {
    for (const expected of EXPECTED) {
        it(`returns the whole definition for ${expected.type}`, () => {
            // given
            const type = expected.type;

            // when
            const definition = gameTypeFor(type);

            // then
            expect(definition?.type).toBe(type);
            expect(definition?.label).toBe(expected.label);
            expect(definition).toBe(GAME_TYPES.find(entry => entry.type === type));
        });
    }

    it("returns nothing for a game type that is not on the roster", () => {
        // given
        const type = "mahjong";

        // when
        const definition = gameTypeFor(type);

        // then
        expect(definition).toBeUndefined();
    });

    it("returns nothing for an empty game type", () => {
        // given
        const type = "";

        // when
        const definition = gameTypeFor(type);

        // then
        expect(definition).toBeUndefined();
    });
});
