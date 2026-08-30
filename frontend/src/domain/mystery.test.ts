import { describe, expect, it } from "vitest";
import type { MysteryAttempt, MysteryClue } from "../types/api";
import {
    cluesForPlayer,
    findWinningAttempt,
    globalClues,
    groupAttemptsByAuthor,
    mysteryPermissions,
    timerColour,
    winningAuthorIds,
} from "./mystery";

const battler = { id: "player-1", username: "battler", display_name: "Battler" };
const ange = { id: "player-2", username: "ange", display_name: "Ange" };
const beatrice = { id: "gm-1", username: "beatrice", display_name: "Beatrice" };

const NOW = Date.parse("2026-07-31T12:00:00Z");

function makeAttempt(overrides: Partial<MysteryAttempt> = {}): MysteryAttempt {
    return {
        id: "attempt-1",
        author: battler,
        body: "The chain was fixed after the fact.",
        is_winner: false,
        vote_score: 0,
        created_at: "2026-07-01T11:00:00Z",
        ...overrides,
    };
}

function makeClue(overrides: Partial<MysteryClue> = {}): MysteryClue {
    return { id: 1, body: "The door was chained", truth_type: "red", sort_order: 0, ...overrides };
}

describe("findWinningAttempt", () => {
    it("finds the crowned attempt at the top of the board", () => {
        // given
        const attempts = [makeAttempt({ id: "a1" }), makeAttempt({ id: "a2", is_winner: true })];

        // when
        const winner = findWinningAttempt(attempts);

        // then
        expect(winner?.id).toBe("a2");
    });

    it("digs the crowned attempt out of a nested thread", () => {
        // given
        const attempts = [
            makeAttempt({
                id: "a1",
                replies: [makeAttempt({ id: "a2", replies: [makeAttempt({ id: "a3", is_winner: true })] })],
            }),
        ];

        // when
        const winner = findWinningAttempt(attempts);

        // then
        expect(winner?.id).toBe("a3");
    });

    it("prefers the first crowned attempt it meets", () => {
        // given
        const attempts = [makeAttempt({ id: "a1", is_winner: true }), makeAttempt({ id: "a2", is_winner: true })];

        // when
        const winner = findWinningAttempt(attempts);

        // then
        expect(winner?.id).toBe("a1");
    });

    it("returns nothing when nobody has been crowned", () => {
        // given
        const attempts = [makeAttempt({ id: "a1", replies: [makeAttempt({ id: "a2" })] })];

        // when
        const winner = findWinningAttempt(attempts);

        // then
        expect(winner).toBeNull();
    });

    it("returns nothing for an empty board", () => {
        // given
        const attempts: MysteryAttempt[] = [];

        // when
        const winner = findWinningAttempt(attempts);

        // then
        expect(winner).toBeNull();
    });
});

describe("groupAttemptsByAuthor", () => {
    it("gathers each author's attempts into one group", () => {
        // given
        const attempts = [
            makeAttempt({ id: "a1" }),
            makeAttempt({ id: "a2", author: ange }),
            makeAttempt({ id: "a3" }),
        ];

        // when
        const groups = groupAttemptsByAuthor(attempts, { freeForAll: false, viewerId: null });

        // then
        expect(groups.map(g => g.author.id)).toEqual(["player-1", "player-2"]);
        expect(groups[0].attempts.map(a => a.id)).toEqual(["a1", "a3"]);
    });

    it("keeps the authors in the order they first moved outside a free-for-all", () => {
        // given
        const attempts = [makeAttempt({ id: "a1", author: ange }), makeAttempt({ id: "a2" })];

        // when
        const groups = groupAttemptsByAuthor(attempts, { freeForAll: false, viewerId: battler.id });

        // then
        expect(groups.map(g => g.author.id)).toEqual(["player-2", "player-1"]);
    });

    it("floats the viewer's own group to the top of a free-for-all", () => {
        // given
        const attempts = [makeAttempt({ id: "a1", author: ange }), makeAttempt({ id: "a2" })];

        // when
        const groups = groupAttemptsByAuthor(attempts, { freeForAll: true, viewerId: battler.id });

        // then
        expect(groups.map(g => g.author.id)).toEqual(["player-1", "player-2"]);
    });

    it("leaves a free-for-all in first-move order for a signed out reader", () => {
        // given
        const attempts = [makeAttempt({ id: "a1", author: ange }), makeAttempt({ id: "a2" })];

        // when
        const groups = groupAttemptsByAuthor(attempts, { freeForAll: true, viewerId: null });

        // then
        expect(groups.map(g => g.author.id)).toEqual(["player-2", "player-1"]);
    });

    it("returns nothing for an empty board", () => {
        // given
        const attempts: MysteryAttempt[] = [];

        // when
        const groups = groupAttemptsByAuthor(attempts, { freeForAll: true, viewerId: battler.id });

        // then
        expect(groups).toEqual([]);
    });
});

describe("winningAuthorIds", () => {
    it("names every author who has already been crowned", () => {
        // given
        const attempts = [
            makeAttempt({ id: "a1", is_winner: true }),
            makeAttempt({ id: "a2", author: ange }),
            makeAttempt({ id: "a3", is_winner: true }),
        ];

        // when
        const winners = winningAuthorIds(attempts);

        // then
        expect(Array.from(winners)).toEqual(["player-1"]);
    });

    it("looks only at the top of the board, never inside a thread", () => {
        // given
        const attempts = [makeAttempt({ id: "a1", replies: [makeAttempt({ id: "a2", is_winner: true })] })];

        // when
        const winners = winningAuthorIds(attempts);

        // then
        expect(winners.size).toBe(0);
    });

    it("names nobody when no attempt has been crowned", () => {
        // given
        const attempts = [makeAttempt({ id: "a1" })];

        // when
        const winners = winningAuthorIds(attempts);

        // then
        expect(winners.size).toBe(0);
    });
});

describe("globalClues", () => {
    it("keeps the red truths the whole board may read", () => {
        // given
        const clues = [makeClue({ id: 1 }), makeClue({ id: 2, player_id: battler.id })];

        // when
        const shared = globalClues(clues);

        // then
        expect(shared.map(c => c.id)).toEqual([1]);
    });

    it("returns nothing when every red truth is whispered", () => {
        // given
        const clues = [makeClue({ id: 2, player_id: battler.id })];

        // when
        const shared = globalClues(clues);

        // then
        expect(shared).toEqual([]);
    });
});

describe("cluesForPlayer", () => {
    it("keeps only the red truths whispered to that player", () => {
        // given
        const clues = [
            makeClue({ id: 1 }),
            makeClue({ id: 2, player_id: battler.id }),
            makeClue({ id: 3, player_id: ange.id }),
        ];

        // when
        const mine = cluesForPlayer(clues, battler.id);

        // then
        expect(mine.map(c => c.id)).toEqual([2]);
    });

    it("returns nothing when nothing was whispered to that player", () => {
        // given
        const clues = [makeClue({ id: 1 })];

        // when
        const mine = cluesForPlayer(clues, battler.id);

        // then
        expect(mine).toEqual([]);
    });
});

describe("mysteryPermissions", () => {
    it("gives the game master everything over their own mystery", () => {
        // given
        const viewer = { id: beatrice.id, role: null };

        // when
        const permissions = mysteryPermissions(viewer, { author: beatrice });

        // then
        expect(permissions).toEqual({
            isAuthor: true,
            canEdit: true,
            canDelete: true,
            canSeeAsGameMaster: true,
        });
    });

    it("lets a moderator edit and delete a mystery they did not set, but not read it as its game master", () => {
        // given
        const viewer = { id: "mod-1", role: "moderator" as const };

        // when
        const permissions = mysteryPermissions(viewer, { author: beatrice });

        // then
        expect(permissions).toEqual({
            isAuthor: false,
            canEdit: true,
            canDelete: true,
            canSeeAsGameMaster: false,
        });
    });

    it("lets a super admin read any mystery as its game master", () => {
        // given
        const viewer = { id: "root-1", role: "super_admin" as const };

        // when
        const permissions = mysteryPermissions(viewer, { author: beatrice });

        // then
        expect(permissions.canSeeAsGameMaster).toBe(true);
    });

    it("gives an ordinary piece nothing over somebody else's mystery", () => {
        // given
        const viewer = { id: battler.id, role: null };

        // when
        const permissions = mysteryPermissions(viewer, { author: beatrice });

        // then
        expect(permissions).toEqual({
            isAuthor: false,
            canEdit: false,
            canDelete: false,
            canSeeAsGameMaster: false,
        });
    });

    it("gives a signed out visitor nothing", () => {
        // given
        const viewer = null;

        // when
        const permissions = mysteryPermissions(viewer, { author: beatrice });

        // then
        expect(permissions).toEqual({
            isAuthor: false,
            canEdit: false,
            canDelete: false,
            canSeeAsGameMaster: false,
        });
    });
});

describe("timerColour", () => {
    it("turns green the moment the mystery is solved, however old it is", () => {
        // given
        const createdAt = "2026-01-01T12:00:00Z";

        // when
        const colour = timerColour(createdAt, true, NOW);

        // then
        expect(colour).toBe("#66bb6a");
    });

    it("stays blue for a mystery set today", () => {
        // given
        const createdAt = "2026-07-31T00:00:00Z";

        // when
        const colour = timerColour(createdAt, false, NOW);

        // then
        expect(colour).toBe("#64b5f6");
    });

    it("turns yellow once a mystery has stood for a day", () => {
        // given
        const createdAt = "2026-07-30T11:00:00Z";

        // when
        const colour = timerColour(createdAt, false, NOW);

        // then
        expect(colour).toBe("#ffd54f");
    });

    it("turns orange once a mystery has stood for a week", () => {
        // given
        const createdAt = "2026-07-20T12:00:00Z";

        // when
        const colour = timerColour(createdAt, false, NOW);

        // then
        expect(colour).toBe("#ffb74d");
    });

    it("turns red once a mystery has stood for a month", () => {
        // given
        const createdAt = "2026-06-01T12:00:00Z";

        // when
        const colour = timerColour(createdAt, false, NOW);

        // then
        expect(colour).toBe("#e57373");
    });

    it("falls back to blue when the mystery has no readable date", () => {
        // given
        const createdAt = "who knows";

        // when
        const colour = timerColour(createdAt, false, NOW);

        // then
        expect(colour).toBe("#64b5f6");
    });
});
