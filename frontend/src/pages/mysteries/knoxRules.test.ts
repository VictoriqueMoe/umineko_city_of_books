import { describe, it, expect } from "vitest";
import { ALL_KNOX_RULES_ON, KNOX_RULES, swornRules } from "./knoxRules";

describe("KNOX_RULES", () => {
    it("holds exactly ten rules in ordinal order", () => {
        // given
        const expected = [
            "Knox's 1st",
            "Knox's 2nd",
            "Knox's 3rd",
            "Knox's 4th",
            "Knox's 5th",
            "Knox's 6th",
            "Knox's 7th",
            "Knox's 8th",
            "Knox's 9th",
            "Knox's 10th",
        ];

        // when
        const ordinals = KNOX_RULES.map(rule => rule.ordinal);

        // then
        expect(ordinals).toEqual(expected);
    });

    it("names every contract key exactly once", () => {
        // given
        const contractKeys = Object.keys(ALL_KNOX_RULES_ON).sort();

        // when
        const ruleKeys = KNOX_RULES.map(rule => rule.key).sort();

        // then
        expect(ruleKeys).toEqual(contractKeys);
    });

    it("defaults every rule to sworn", () => {
        // given
        const values = Object.values(ALL_KNOX_RULES_ON);

        // then
        expect(values).toHaveLength(10);
        expect(values.every(Boolean)).toBe(true);
    });
});

describe("swornRules", () => {
    it("returns every rule when the whole decalogue is sworn", () => {
        // when
        const sworn = swornRules(ALL_KNOX_RULES_ON);

        // then
        expect(sworn).toHaveLength(10);
    });

    it("drops the rules the game master waived", () => {
        // given
        const contract = { ...ALL_KNOX_RULES_ON, no_supernatural: false, detective_not_culprit: false };

        // when
        const sworn = swornRules(contract);

        // then
        expect(sworn).toHaveLength(8);
        expect(sworn.map(rule => rule.key)).not.toContain("no_supernatural");
        expect(sworn.map(rule => rule.key)).not.toContain("detective_not_culprit");
    });

    it("returns nothing when the game master swears nothing", () => {
        // given
        const contract = Object.fromEntries(
            Object.keys(ALL_KNOX_RULES_ON).map(key => [key, false]),
        ) as unknown as typeof ALL_KNOX_RULES_ON;

        // when
        const sworn = swornRules(contract);

        // then
        expect(sworn).toEqual([]);
    });
});
