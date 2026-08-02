import { describe, expect, it } from "vitest";
import { makeUser } from "../test-utils/fixtures";
import { createTestQueryClient } from "../test-utils/render";
import type { UserProfile } from "../types/api";
import { patchUserInCache } from "./userCache";

interface FeedPayload {
    items: { id: string; author: UserProfile }[];
    meta: { total: number };
}

const beatrice = () => makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });
const ange = () => makeUser({ id: "u2", username: "ange", display_name: "Ange" });

describe("patchUserInCache", () => {
    it("patches a user sitting at the root of a cache entry", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["profile", "u1"], beatrice());

        // when
        patchUserInCache(qc, "u1", { display_name: "Golden Witch" });

        // then
        expect(qc.getQueryData<UserProfile>(["profile", "u1"])?.display_name).toBe("Golden Witch");
    });

    it("patches users buried inside arrays and nested objects", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData<FeedPayload>(["theories"], {
            items: [
                { id: "t1", author: beatrice() },
                { id: "t2", author: ange() },
            ],
            meta: { total: 2 },
        });

        // when
        patchUserInCache(qc, "u1", { banned: true, ban_reason: "practising sorcery" });

        // then
        const feed = qc.getQueryData<FeedPayload>(["theories"]);
        expect(feed?.items[0].author.banned).toBe(true);
        expect(feed?.items[0].author.ban_reason).toBe("practising sorcery");
        expect(feed?.items[1].author.banned).toBeUndefined();
    });

    it("patches every copy of that user across every query", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["profile", "u1"], beatrice());
        qc.setQueryData(["members"], [beatrice(), ange()]);

        // when
        patchUserInCache(qc, "u1", { avatar_url: "/uploads/new.png" });

        // then
        expect(qc.getQueryData<UserProfile>(["profile", "u1"])?.avatar_url).toBe("/uploads/new.png");
        expect(qc.getQueryData<UserProfile[]>(["members"])?.[0].avatar_url).toBe("/uploads/new.png");
        expect(qc.getQueryData<UserProfile[]>(["members"])?.[1].avatar_url).toBe("");
    });

    it("patches the moderation fields a lock or role change needs", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["profile", "u1"], beatrice());

        // when
        patchUserInCache(qc, "u1", { locked: true, lock_reason: "too many golden butterflies", role: "moderator" });

        // then
        const patched = qc.getQueryData<UserProfile>(["profile", "u1"]);
        expect(patched?.locked).toBe(true);
        expect(patched?.lock_reason).toBe("too many golden butterflies");
        expect(patched?.role).toBe("moderator");
    });

    it("leaves a query that never mentions the user completely untouched", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["members"], [ange()]);
        const before = qc.getQueryData(["members"]);

        // when
        patchUserInCache(qc, "u1", { banned: true });

        // then
        expect(qc.getQueryData(["members"])).toBe(before);
    });

    it("leaves the branches that do not mention the user identical", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData<FeedPayload>(["theories"], {
            items: [
                { id: "t1", author: beatrice() },
                { id: "t2", author: ange() },
            ],
            meta: { total: 2 },
        });
        const before = qc.getQueryData<FeedPayload>(["theories"]);

        // when
        patchUserInCache(qc, "u1", { banned: true });

        // then
        const after = qc.getQueryData<FeedPayload>(["theories"]);
        expect(after).not.toBe(before);
        expect(after?.items[1]).toBe(before?.items[1]);
        expect(after?.meta).toBe(before?.meta);
    });

    it("ignores patch fields that were left undefined", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["profile", "u1"], beatrice());
        const before = qc.getQueryData(["profile", "u1"]);

        // when
        patchUserInCache(qc, "u1", { display_name: undefined, banned: undefined });

        // then
        expect(qc.getQueryData(["profile", "u1"])).toBe(before);
    });

    it("rewrites nothing when the cache already holds the patched value", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["profile", "u1"], beatrice());
        const before = qc.getQueryData(["profile", "u1"]);

        // when
        patchUserInCache(qc, "u1", { display_name: "Beatrice" });

        // then
        expect(qc.getQueryData(["profile", "u1"])).toBe(before);
    });

    it("ignores objects that share the id but are not users", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["theory", "u1"], { id: "u1", title: "The witch did it", body: "" });
        const before = qc.getQueryData(["theory", "u1"]);

        // when
        patchUserInCache(qc, "u1", { banned: true });

        // then
        expect(qc.getQueryData(["theory", "u1"])).toBe(before);
        expect(qc.getQueryData(["theory", "u1"])).not.toHaveProperty("banned");
    });

    it("walks past nulls and primitives without tripping over them", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["mixed"], { author: null, note: "nothing here", count: 3, member: beatrice() });

        // when
        patchUserInCache(qc, "u1", { banned: true });

        // then
        const patched = qc.getQueryData<{ author: null; note: string; count: number; member: UserProfile }>(["mixed"]);
        expect(patched?.author).toBeNull();
        expect(patched?.note).toBe("nothing here");
        expect(patched?.count).toBe(3);
        expect(patched?.member.banned).toBe(true);
    });

    it("does nothing when no query holds that user", () => {
        // given
        const qc = createTestQueryClient();
        qc.setQueryData(["members"], [ange()]);

        // when
        patchUserInCache(qc, "u404", { banned: true });

        // then
        expect(qc.getQueryData<UserProfile[]>(["members"])?.[0].banned).toBeUndefined();
    });
});
