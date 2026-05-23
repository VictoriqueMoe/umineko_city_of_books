import { CharacterId } from "./types";

interface AudioConfig {
    start: string;
    default: {
        win: string;
        lose: string;
    };
    matchups?: {
        [opponentId in CharacterId]?: {
            win?: string;
            lose?: string;
        };
    };
}

const CHARACTER_AUDIO: Partial<Record<CharacterId, AudioConfig>> = {
    [CharacterId.Bernkastel]: {
        start: "",
        default: {
            lose: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/28/82100692",
            win: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/combined?segments=28:72100547,28:72100548,28:72100549",
        },
        matchups: {
            [CharacterId.Lambdadelta]: {
                lose: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/28/82100517",
            },
        },
    },
    [CharacterId.Erika]: {
        start: "",
        default: {
            win: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/combined?segments=46:64501228,46:64501229,46:64501230",
            lose: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/combined?segments=46:54500569,46:54500571,46:54500572",
        },
        matchups: {},
    },
    [CharacterId.Lambdadelta]: {
        start: "",
        default: {
            win: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/combined?segments=29:92200077,29:92200078,29:92200079",
            lose: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/combined?segments=29:82200264,29:82200265",
        },
        matchups: {},
    },
    [CharacterId.Dlanor]: {
        start: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/combined?segments=47:54600001,47:54600002",
        default: {
            win: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/combined?segments=47:54600315,47:54600316,47:54600317",
            lose: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/47/84600008",
        },
        matchups: {
            [CharacterId.Erika]: {
                lose: "https://quotes.auaurora.moe/api/v1/umineko/audio/voice/47/64600077",
            },
        },
    },
};

export function getStartAudio(character: CharacterId | ""): string | null {
    if (!character) {
        return null;
    }
    return CHARACTER_AUDIO[character]?.start ?? null;
}

export function getGameOverAudio(
    myCharacter: CharacterId | "",
    opponentCharacter: CharacterId | "",
    won: boolean,
): string | null {
    if (!myCharacter || !opponentCharacter) {
        return null;
    }
    const config = CHARACTER_AUDIO[myCharacter];
    if (!config) {
        return null;
    }
    const key = won ? "win" : "lose";
    const matchup = config.matchups?.[opponentCharacter];
    return matchup?.[key] ?? config.default?.[key] ?? null;
}
