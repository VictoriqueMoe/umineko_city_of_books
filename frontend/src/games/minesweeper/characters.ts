import { CharacterDef, CharacterId, Expression, Mood } from "./types";

const MOOD_FALLBACKS: Record<Mood, Mood[]> = {
    default: [],
    neutral: ["default"],
    happy: ["smirk", "neutral", "default"],
    very_happy: ["happy", "smirk", "default"],
    smirk: ["happy", "neutral", "default"],
    worried: ["sweating", "neutral", "default"],
    sweating: ["worried", "neutral", "default"],
    angry: ["furious", "sweating", "neutral", "default"],
    furious: ["angry", "sweating", "neutral", "default"],
    surprised: ["worried", "neutral", "default"],
    relieved: ["happy", "neutral", "default"],
    bored: ["neutral", "default"],
    wink: ["smirk", "happy", "default"],
    win: ["very_happy", "happy", "default"],
    lose: ["furious", "angry", "worried", "default"],
};

export function resolveExpression(character: CharacterDef, mood: Mood): Expression {
    if (character.expressions[mood]) {
        return character.expressions[mood] as Expression;
    }
    const fallbacks = MOOD_FALLBACKS[mood];
    for (let i = 0; i < fallbacks.length; i++) {
        const fb = fallbacks[i];
        if (character.expressions[fb]) {
            return character.expressions[fb] as Expression;
        }
    }
    return character.expressions.default ?? { image: character.image, facing: "center" };
}

export const CHARACTERS: CharacterDef[] = [
    {
        id: CharacterId.Bernkastel,
        name: "Bernkastel",
        image: "/characters/bernkastel/bern-default.png",
        expressions: {
            default: { image: "/characters/bernkastel/bern-default.png", facing: "right" },
            neutral: { image: "/characters/bernkastel/bern-neutral.png", facing: "center" },
            happy: { image: "/characters/bernkastel/bern-happy.png", facing: "center" },
            smirk: { image: "/characters/bernkastel/bern-smirk.png", facing: "center" },
            worried: { image: "/characters/bernkastel/bern-worried.png", facing: "center" },
            sweating: { image: "/characters/bernkastel/bern-sweating.png", facing: "center" },
            angry: { image: "/characters/bernkastel/bern-angry.png", facing: "center" },
            furious: { image: "/characters/bernkastel/bern_furious.png", facing: "center" },
            surprised: { image: "/characters/bernkastel/bern-supprised.png", facing: "center" },
            relieved: { image: "/characters/bernkastel/bern_relived.png", facing: "center" },
            bored: { image: "/characters/bernkastel/bern-board.png", facing: "center" },
            win: { image: "/characters/bernkastel/bern-win.png", facing: "center" },
            lose: { image: "/characters/bernkastel/bern-lose.png", facing: "center" },
        },
    },
    {
        id: CharacterId.Erika,
        name: "Erika Furudo",
        image: "/characters/erika/erika-default.png",
        expressions: {
            default: { image: "/characters/erika/erika-default.png", facing: "left" },
            neutral: { image: "/characters/erika/erika-neutral.png", facing: "right" },
            happy: { image: "/characters/erika/erika-happy.png", facing: "right" },
            very_happy: { image: "/characters/erika/erika-very_happy.png", facing: "right" },
            smirk: { image: "/characters/erika/eria-smirk.png", facing: "right" },
            surprised: { image: "/characters/erika/erika-suprised.png", facing: "right" },
            furious: { image: "/characters/erika/erika-furious.png", facing: "right" },
            wink: { image: "/characters/erika/erika-wink.png", facing: "right" },
            win: { image: "/characters/erika/erika-win.png", facing: "right" },
            lose: { image: "/characters/erika/erika-lose.png", facing: "right" },
        },
    },
    {
        id: CharacterId.Dlanor,
        name: "Dlanor A. Knox",
        image: "/characters/Dlanor A. Knox/dlanor-default.png",
        expressions: {
            default: { image: "/characters/Dlanor A. Knox/dlanor-default.png", facing: "right" },
            neutral: { image: "/characters/Dlanor A. Knox/dlanor-neutral.png", facing: "right" },
            happy: { image: "/characters/Dlanor A. Knox/dlanor-blush.png", facing: "right" },
            smirk: { image: "/characters/Dlanor A. Knox/dlanor-smirk.png", facing: "right" },
            angry: { image: "/characters/Dlanor A. Knox/dlanor=angry.png", facing: "right" },
            furious: { image: "/characters/Dlanor A. Knox/dlanor-very-annoyed.png", facing: "right" },
            surprised: { image: "/characters/Dlanor A. Knox/dlanor-surprised.png", facing: "right" },
            bored: { image: "/characters/Dlanor A. Knox/dlanor-annoyed.png", facing: "right" },
            lose: { image: "/characters/Dlanor A. Knox/dlanor-contempt.png", facing: "right" },
        },
    },
    {
        id: CharacterId.Lambdadelta,
        name: "Lambdadelta",
        image: "/characters/lambdadelta/lambdadelta-default.png",
        expressions: {
            default: { image: "/characters/lambdadelta/lambdadelta-default.png", facing: "left" },
            happy: { image: "/characters/lambdadelta/lambdadelta-happy.png", facing: "right" },
            very_happy: { image: "/characters/lambdadelta/lambdadelta-very_happy.png", facing: "right" },
            smirk: { image: "/characters/lambdadelta/lambdadelta-smirk.png", facing: "right" },
            worried: { image: "/characters/lambdadelta/lambdadelta-upset.png", facing: "right" },
            win: { image: "/characters/lambdadelta/lambdadelta-win.png", facing: "right" },
            lose: { image: "/characters/lambdadelta/lambdadelta-lose.png", facing: "right" },
        },
    },
];

export function findCharacter(id: string | CharacterId | undefined): CharacterDef | undefined {
    if (!id) {
        return undefined;
    }
    for (let i = 0; i < CHARACTERS.length; i++) {
        if (CHARACTERS[i].id === id) {
            return CHARACTERS[i];
        }
    }
    return undefined;
}
