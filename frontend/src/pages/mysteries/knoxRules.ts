import type { KnoxContract } from "../../types/api";

export interface KnoxRule {
    key: keyof KnoxContract;
    ordinal: string;
    label: string;
    sworn: string;
}

export const KNOX_RULES: KnoxRule[] = [
    {
        key: "culprit_named_early",
        ordinal: "Knox's 1st",
        label: "Culprit named early",
        sworn: "It Is Forbidden for the culprit to be anyone who was not mentioned in the early part of the tale.",
    },
    {
        key: "no_supernatural",
        ordinal: "Knox's 2nd",
        label: "No supernatural solution",
        sworn: "It Is Forbidden for the supernatural to be the answer. Magic may decorate this tale. It may not solve it.",
    },
    {
        key: "passages_declared",
        ordinal: "Knox's 3rd",
        label: "Hidden passages declared",
        sworn: "It Is Forbidden for more than one hidden passage to exist, and It Is Forbidden for any passage to go undeclared.",
    },
    {
        key: "no_unknown_poison",
        ordinal: "Knox's 4th",
        label: "No unknown poisons or gadgets",
        sworn: "It Is Forbidden for an unknown poison, or a device requiring a long scientific explanation, to carry the solution.",
    },
    {
        key: "no_outsider",
        ordinal: "Knox's 5th",
        label: "No stranger from nowhere",
        sworn: "It Is Forbidden for the answer to rest upon a stranger who never once walked into this tale.",
    },
    {
        key: "no_lucky_accident",
        ordinal: "Knox's 6th",
        label: "No lucky accidents",
        sworn: "It Is Forbidden for accident, or for intuition that cannot be accounted for, to hand the detective the truth.",
    },
    {
        key: "detective_not_culprit",
        ordinal: "Knox's 7th",
        label: "The detective is not the culprit",
        sworn: "It Is Forbidden for the detective to be the culprit.",
    },
    {
        key: "clues_shown",
        ordinal: "Knox's 8th",
        label: "Clues shown as they are found",
        sworn: "It Is Forbidden for the detective to act upon a clue that has not been shown to you at the same moment.",
    },
    {
        key: "narrator_hides_nothing",
        ordinal: "Knox's 9th",
        label: "The narrator hides nothing",
        sworn: "It Is Forbidden for the narrator to conceal a thought that passes through their mind.",
    },
    {
        key: "no_unannounced_twins",
        ordinal: "Knox's 10th",
        label: "No unannounced twins or doubles",
        sworn: "It Is Forbidden for a twin, or any double, to appear without due warning.",
    },
];

export const ALL_KNOX_RULES_ON: KnoxContract = {
    culprit_named_early: true,
    no_supernatural: true,
    passages_declared: true,
    no_unknown_poison: true,
    no_outsider: true,
    no_lucky_accident: true,
    detective_not_culprit: true,
    clues_shown: true,
    narrator_hides_nothing: true,
    no_unannounced_twins: true,
};

export function swornRules(contract: KnoxContract): KnoxRule[] {
    return KNOX_RULES.filter(rule => contract[rule.key]);
}
