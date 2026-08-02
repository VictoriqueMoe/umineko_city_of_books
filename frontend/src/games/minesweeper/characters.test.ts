import { describe, expect, it } from "vitest";
import { CHARACTERS, findCharacter, resolveExpression } from "./characters";
import { CharacterDef, CharacterId, Expression, Mood } from "./types";

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

function makeCharacter(expressions: Partial<Record<Mood, Expression>>): CharacterDef {
    return {
        id: CharacterId.Bernkastel,
        name: "Test Witch",
        image: "/characters/test/base.png",
        expressions,
    };
}

function expressionFor(id: CharacterId, mood: Mood): Expression {
    const character = CHARACTERS.find(c => c.id === id);
    if (!character) {
        throw new Error(`no character named ${id}`);
    }

    return resolveExpression(character, mood);
}

describe("resolveExpression", () => {
    it("returns the exact expression when the character defines the mood asked for", () => {
        // given
        const character = makeCharacter({
            default: { image: "/characters/test/default.png", facing: "left" },
            furious: { image: "/characters/test/furious.png", facing: "right" },
        });

        // when
        const expression = resolveExpression(character, "furious");

        // then
        expect(expression).toEqual({ image: "/characters/test/furious.png", facing: "right" });
    });

    it("takes the first fallback in the chain when the mood itself is missing", () => {
        // given
        const character = makeCharacter({
            default: { image: "/characters/test/default.png", facing: "center" },
            neutral: { image: "/characters/test/neutral.png", facing: "center" },
            smirk: { image: "/characters/test/smirk.png", facing: "center" },
        });

        // when
        const expression = resolveExpression(character, "happy");

        // then
        expect(expression.image).toBe("/characters/test/smirk.png");
    });

    it("skips over fallbacks the character has not drawn", () => {
        // given
        const character = makeCharacter({
            default: { image: "/characters/test/default.png", facing: "center" },
            neutral: { image: "/characters/test/neutral.png", facing: "center" },
        });

        // when
        const expression = resolveExpression(character, "happy");

        // then
        expect(expression.image).toBe("/characters/test/neutral.png");
    });

    it("lands on the default expression when nothing earlier in the chain exists", () => {
        // given
        const character = makeCharacter({
            default: { image: "/characters/test/default.png", facing: "left" },
        });

        // when
        const expression = resolveExpression(character, "bored");

        // then
        expect(expression).toEqual({ image: "/characters/test/default.png", facing: "left" });
    });

    it("synthesises a centred expression from the base image when the whole chain is missing", () => {
        // given
        const character = makeCharacter({});

        // when
        const expression = resolveExpression(character, "lose");

        // then
        expect(expression).toEqual({ image: "/characters/test/base.png", facing: "center" });
    });

    it("synthesises from the base image for the default mood because it has no fallbacks of its own", () => {
        // given
        const character = makeCharacter({ happy: { image: "/characters/test/happy.png", facing: "center" } });

        // when
        const expression = resolveExpression(character, "default");

        // then
        expect(expression).toEqual({ image: "/characters/test/base.png", facing: "center" });
    });

    it("never returns an expression without an image for any character and mood combination", () => {
        // given
        const resolved: Expression[] = [];

        // when
        for (const character of CHARACTERS) {
            for (const mood of ALL_MOODS) {
                resolved.push(resolveExpression(character, mood));
            }
        }

        // then
        expect(resolved).toHaveLength(CHARACTERS.length * ALL_MOODS.length);
        for (const expression of resolved) {
            expect(expression.image).toMatch(/^\/characters\//);
            expect(["left", "right", "center"]).toContain(expression.facing);
        }
    });
});

describe("resolveExpression fallbacks for the shipped cast", () => {
    const cases: { name: string; id: CharacterId; mood: Mood; image: string }[] = [
        {
            name: "Bernkastel borrows her happy face when she has no very happy one",
            id: CharacterId.Bernkastel,
            mood: "very_happy",
            image: "/characters/bernkastel/bern-happy.png",
        },
        {
            name: "Bernkastel borrows her smirk when she cannot wink",
            id: CharacterId.Bernkastel,
            mood: "wink",
            image: "/characters/bernkastel/bern-smirk.png",
        },
        {
            name: "Erika borrows her furious face when she has no plain angry one",
            id: CharacterId.Erika,
            mood: "angry",
            image: "/characters/erika/erika-furious.png",
        },
        {
            name: "Erika falls through to neutral when she has neither worry nor sweat",
            id: CharacterId.Erika,
            mood: "sweating",
            image: "/characters/erika/erika-neutral.png",
        },
        {
            name: "Erika borrows her happy face for relief",
            id: CharacterId.Erika,
            mood: "relieved",
            image: "/characters/erika/erika-happy.png",
        },
        {
            name: "Dlanor borrows her blush for a win she has no pose for",
            id: CharacterId.Dlanor,
            mood: "win",
            image: "/characters/Dlanor A. Knox/dlanor-blush.png",
        },
        {
            name: "Dlanor borrows her smirk when she cannot wink",
            id: CharacterId.Dlanor,
            mood: "wink",
            image: "/characters/Dlanor A. Knox/dlanor-smirk.png",
        },
        {
            name: "Lambdadelta borrows her upset face for surprise",
            id: CharacterId.Lambdadelta,
            mood: "surprised",
            image: "/characters/lambdadelta/lambdadelta-upset.png",
        },
        {
            name: "Lambdadelta drops all the way back to her default when she has no anger at all",
            id: CharacterId.Lambdadelta,
            mood: "angry",
            image: "/characters/lambdadelta/lambdadelta-default.png",
        },
        {
            name: "Lambdadelta uses her default in place of a neutral face",
            id: CharacterId.Lambdadelta,
            mood: "neutral",
            image: "/characters/lambdadelta/lambdadelta-default.png",
        },
    ];

    for (const testCase of cases) {
        it(testCase.name, () => {
            // given
            const mood = testCase.mood;

            // when
            const expression = expressionFor(testCase.id, mood);

            // then
            expect(expression.image).toBe(testCase.image);
        });
    }
});

describe("CHARACTERS", () => {
    it("ships the four playable witches exactly once each", () => {
        // given
        const ids = CHARACTERS.map(c => c.id);

        // when
        const unique = new Set(ids);

        // then
        expect(ids).toEqual([CharacterId.Bernkastel, CharacterId.Erika, CharacterId.Dlanor, CharacterId.Lambdadelta]);
        expect(unique.size).toBe(ids.length);
    });

    it("gives every character a name, a base image and a default expression", () => {
        // given
        const characters = CHARACTERS;

        // when
        const defaults = characters.map(c => c.expressions.default);

        // then
        for (const character of characters) {
            expect(character.name.length).toBeGreaterThan(0);
            expect(character.image).toMatch(/^\/characters\/.+\.png$/);
        }
        for (const expression of defaults) {
            expect(expression).toBeDefined();
            expect(expression?.image).toMatch(/^\/characters\/.+\.png$/);
        }
    });

    it("points every default expression at the same file as the base image", () => {
        // given
        const characters = CHARACTERS;

        // when
        const mismatched = characters.filter(c => c.expressions.default?.image !== c.image);

        // then
        expect(mismatched).toEqual([]);
    });

    it("gives every character a win and a lose pose or a fallback that reaches one", () => {
        // given
        const characters = CHARACTERS;

        // when
        const winImages = characters.map(c => resolveExpression(c, "win").image);
        const loseImages = characters.map(c => resolveExpression(c, "lose").image);

        // then
        for (const image of winImages) {
            expect(image.length).toBeGreaterThan(0);
        }
        for (const image of loseImages) {
            expect(image.length).toBeGreaterThan(0);
        }
    });
});

describe("findCharacter", () => {
    const cases: { id: CharacterId; name: string }[] = [
        { id: CharacterId.Bernkastel, name: "Bernkastel" },
        { id: CharacterId.Erika, name: "Erika Furudo" },
        { id: CharacterId.Dlanor, name: "Dlanor A. Knox" },
        { id: CharacterId.Lambdadelta, name: "Lambdadelta" },
    ];

    for (const testCase of cases) {
        it(`finds ${testCase.name} by her id`, () => {
            // given
            const id = testCase.id;

            // when
            const character = findCharacter(id);

            // then
            expect(character?.id).toBe(id);
            expect(character?.name).toBe(testCase.name);
        });
    }

    it("finds a character from a plain string id as well as the enum value", () => {
        // given
        const id = "bernkastel";

        // when
        const character = findCharacter(id);

        // then
        expect(character).toBe(CHARACTERS[0]);
    });

    it("returns nothing for a character that is not in the cast", () => {
        // given
        const id = "beatrice";

        // when
        const character = findCharacter(id);

        // then
        expect(character).toBeUndefined();
    });

    it("returns nothing when no character has been picked yet", () => {
        // given
        const id = undefined;

        // when
        const character = findCharacter(id);

        // then
        expect(character).toBeUndefined();
    });

    it("treats an empty id as no character rather than searching for it", () => {
        // given
        const id = "";

        // when
        const character = findCharacter(id);

        // then
        expect(character).toBeUndefined();
    });
});
