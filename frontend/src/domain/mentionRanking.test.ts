import { describe, expect, it } from "vitest";
import { compareMentionCandidates, mentionAffinity, rankMentionCandidates } from "./mentionRanking";

interface Candidate {
    username: string;
    viewer_follows: boolean;
    follows_viewer: boolean;
}

function candidate(username: string, viewer_follows: boolean, follows_viewer: boolean): Candidate {
    return { username, viewer_follows, follows_viewer };
}

describe("mentionAffinity", () => {
    it("weights who the viewer follows above who follows the viewer", () => {
        // given
        const following = candidate("beatrice", true, false);
        const follower = candidate("battler", false, true);

        // when
        const scores = [mentionAffinity(following), mentionAffinity(follower)];

        // then
        expect(scores).toEqual([2, 1]);
    });

    it("scores a mutual follow above either half and a stranger at nothing", () => {
        // given
        const mutual = candidate("bernkastel", true, true);
        const stranger = candidate("erika", false, false);

        // when
        const scores = [mentionAffinity(mutual), mentionAffinity(stranger)];

        // then
        expect(scores).toEqual([3, 0]);
    });
});

describe("compareMentionCandidates", () => {
    it("orders by descending affinity", () => {
        // given
        const mutual = candidate("bernkastel", true, true);
        const stranger = candidate("erika", false, false);

        // when
        const order = compareMentionCandidates(mutual, stranger);

        // then
        expect(order).toBeLessThan(0);
    });

    it("reports a tie as zero so the sort keeps the incoming order", () => {
        // given
        const first = candidate("lambdadelta", true, false);
        const second = candidate("featherine", true, false);

        // when
        const order = compareMentionCandidates(first, second);

        // then
        expect(order).toBe(0);
    });
});

describe("rankMentionCandidates", () => {
    it("puts mutual follows first, then the viewer's follows, then followers, then strangers", () => {
        // given
        const candidates = [
            candidate("erika", false, false),
            candidate("battler", false, true),
            candidate("bernkastel", true, true),
            candidate("beatrice", true, false),
        ];

        // when
        const ranked = rankMentionCandidates(candidates);

        // then
        expect(ranked.map(c => c.username)).toEqual(["bernkastel", "beatrice", "battler", "erika"]);
    });

    it("keeps the incoming order between candidates of equal affinity", () => {
        // given
        const candidates = [
            candidate("zepar", false, false),
            candidate("furfur", true, false),
            candidate("gaap", false, false),
            candidate("ronove", true, false),
            candidate("virgilia", false, false),
        ];

        // when
        const ranked = rankMentionCandidates(candidates);

        // then
        expect(ranked.map(c => c.username)).toEqual(["furfur", "ronove", "zepar", "gaap", "virgilia"]);
    });

    it("leaves the caller's array untouched", () => {
        // given
        const candidates = [candidate("erika", false, false), candidate("bernkastel", true, true)];

        // when
        const ranked = rankMentionCandidates(candidates);

        // then
        expect(candidates.map(c => c.username)).toEqual(["erika", "bernkastel"]);
        expect(ranked).not.toBe(candidates);
    });

    it("returns an empty list unchanged", () => {
        // given
        const candidates: Candidate[] = [];

        // when
        const ranked = rankMentionCandidates(candidates);

        // then
        expect(ranked).toEqual([]);
    });
});
