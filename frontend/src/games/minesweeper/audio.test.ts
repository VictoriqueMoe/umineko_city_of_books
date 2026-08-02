import { describe, expect, it } from "vitest";
import { getGameOverAudio, getStartAudio } from "./audio";
import { CharacterId } from "./types";

const VOICE = "https://quotes.auaurora.moe/api/v1/umineko/audio/voice";

const DLANOR_START = `${VOICE}/combined?segments=47:54600001,47:54600002`;
const DLANOR_WIN = `${VOICE}/combined?segments=47:54600315,47:54600316,47:54600317`;
const DLANOR_LOSE = `${VOICE}/47/84600008`;
const DLANOR_LOSE_TO_ERIKA = `${VOICE}/47/64600077`;

const BERN_WIN = `${VOICE}/combined?segments=28:72100547,28:72100548,28:72100549`;
const BERN_LOSE = `${VOICE}/28/82100692`;
const BERN_LOSE_TO_LAMBDA = `${VOICE}/28/82100517`;

const ERIKA_WIN = `${VOICE}/combined?segments=46:64501228,46:64501229,46:64501230`;
const ERIKA_LOSE = `${VOICE}/combined?segments=46:54500569,46:54500571,46:54500572`;

const LAMBDA_WIN = `${VOICE}/combined?segments=29:92200077,29:92200078,29:92200079`;
const LAMBDA_LOSE = `${VOICE}/combined?segments=29:82200264,29:82200265`;

describe("getStartAudio", () => {
    it("returns the opening line for the only character who has one", () => {
        // given
        const character = CharacterId.Dlanor;

        // when
        const url = getStartAudio(character);

        // then
        expect(url).toBe(DLANOR_START);
    });

    it("returns nothing when no character has been chosen", () => {
        // given
        const character = "" as const;

        // when
        const url = getStartAudio(character);

        // then
        expect(url).toBeNull();
    });

    it("returns nothing for a character the audio table does not cover", () => {
        // given
        const character = "featherine" as CharacterId;

        // when
        const url = getStartAudio(character);

        // then
        expect(url).toBeNull();
    });

    const silent: CharacterId[] = [CharacterId.Bernkastel, CharacterId.Erika, CharacterId.Lambdadelta];

    for (const character of silent) {
        it(`gives ${character} nothing to say at the start of a match`, () => {
            // given
            const chosen = character;

            // when
            const url = getStartAudio(chosen);

            // then
            expect(url).toBeNull();
        });
    }
});

describe("getGameOverAudio", () => {
    it("plays the matchup line when the winner is one the character has a special reply for", () => {
        // given
        const me = CharacterId.Bernkastel;

        // when
        const url = getGameOverAudio(me, CharacterId.Lambdadelta, false);

        // then
        expect(url).toBe(BERN_LOSE_TO_LAMBDA);
    });

    it("falls back to the default win line when the matchup only overrides the loss", () => {
        // given
        const me = CharacterId.Bernkastel;

        // when
        const url = getGameOverAudio(me, CharacterId.Lambdadelta, true);

        // then
        expect(url).toBe(BERN_WIN);
    });

    it("plays Dlanor's dedicated line for losing to Erika", () => {
        // given
        const me = CharacterId.Dlanor;

        // when
        const url = getGameOverAudio(me, CharacterId.Erika, false);

        // then
        expect(url).toBe(DLANOR_LOSE_TO_ERIKA);
    });

    it("uses the default lose line against an opponent with no matchup entry", () => {
        // given
        const me = CharacterId.Dlanor;

        // when
        const url = getGameOverAudio(me, CharacterId.Bernkastel, false);

        // then
        expect(url).toBe(DLANOR_LOSE);
    });

    it("returns nothing when the player has not chosen a character", () => {
        // given
        const me = "" as const;

        // when
        const url = getGameOverAudio(me, CharacterId.Erika, true);

        // then
        expect(url).toBeNull();
    });

    it("returns nothing when the opponent has not chosen a character", () => {
        // given
        const opponent = "" as const;

        // when
        const url = getGameOverAudio(CharacterId.Erika, opponent, true);

        // then
        expect(url).toBeNull();
    });

    it("returns nothing for a character the audio table does not cover", () => {
        // given
        const me = "featherine" as CharacterId;

        // when
        const url = getGameOverAudio(me, CharacterId.Erika, true);

        // then
        expect(url).toBeNull();
    });

    it("uses the default lines for an opponent the audio table does not cover", () => {
        // given
        const opponent = "featherine" as CharacterId;

        // when
        const won = getGameOverAudio(CharacterId.Bernkastel, opponent, true);
        const lost = getGameOverAudio(CharacterId.Bernkastel, opponent, false);

        // then
        expect(won).toBe(BERN_WIN);
        expect(lost).toBe(BERN_LOSE);
    });

    const defaultCases: { me: CharacterId; opponent: CharacterId; won: boolean; url: string }[] = [
        { me: CharacterId.Bernkastel, opponent: CharacterId.Erika, won: true, url: BERN_WIN },
        { me: CharacterId.Bernkastel, opponent: CharacterId.Erika, won: false, url: BERN_LOSE },
        { me: CharacterId.Erika, opponent: CharacterId.Dlanor, won: true, url: ERIKA_WIN },
        { me: CharacterId.Erika, opponent: CharacterId.Dlanor, won: false, url: ERIKA_LOSE },
        { me: CharacterId.Lambdadelta, opponent: CharacterId.Bernkastel, won: true, url: LAMBDA_WIN },
        { me: CharacterId.Lambdadelta, opponent: CharacterId.Bernkastel, won: false, url: LAMBDA_LOSE },
        { me: CharacterId.Dlanor, opponent: CharacterId.Lambdadelta, won: true, url: DLANOR_WIN },
        { me: CharacterId.Dlanor, opponent: CharacterId.Lambdadelta, won: false, url: DLANOR_LOSE },
    ];

    for (const testCase of defaultCases) {
        const outcome = testCase.won ? "beating" : "losing to";
        it(`plays ${testCase.me}'s default line for ${outcome} ${testCase.opponent}`, () => {
            // given
            const me = testCase.me;

            // when
            const url = getGameOverAudio(me, testCase.opponent, testCase.won);

            // then
            expect(url).toBe(testCase.url);
        });
    }

    it("plays a line for every pairing of characters in the cast", () => {
        // given
        const cast = [CharacterId.Bernkastel, CharacterId.Erika, CharacterId.Dlanor, CharacterId.Lambdadelta];

        // when
        const urls: (string | null)[] = [];
        for (const me of cast) {
            for (const opponent of cast) {
                urls.push(getGameOverAudio(me, opponent, true));
                urls.push(getGameOverAudio(me, opponent, false));
            }
        }

        // then
        expect(urls).toHaveLength(cast.length * cast.length * 2);
        for (const url of urls) {
            expect(url).toMatch(/^https:\/\/quotes\.auaurora\.moe\//);
        }
    });
});
