import { describe, expect, it } from "vitest";
import {
    BROADCASTER_IDENTITY_PREFIX,
    MONITOR_IDENTITY_PREFIX,
    UNNAMED_VIEWER,
    VIEWER_IDENTITY_PREFIX,
    countViewers,
    isViewerIdentity,
    readViewerMeta,
    selectViewers,
    summariseViewers,
    type ViewerMeta,
    type ViewerParticipant,
} from "./viewers";

function participant(identity: string, name?: string, meta?: ViewerMeta | string): ViewerParticipant {
    return {
        identity,
        name,
        metadata: typeof meta === "string" ? meta : meta ? JSON.stringify(meta) : undefined,
    };
}

describe("isViewerIdentity", () => {
    it("accepts an identity minted with the viewer prefix", () => {
        // given
        const identity = `${VIEWER_IDENTITY_PREFIX}0f6b`;

        // when
        const isViewer = isViewerIdentity(identity);

        // then
        expect(isViewer).toBe(true);
    });

    it("excludes the broadcaster and the stream owner's own monitor join", () => {
        // given
        const identities = [`${BROADCASTER_IDENTITY_PREFIX}0f6b`, `${MONITOR_IDENTITY_PREFIX}0f6b`];

        // when
        const results = identities.map(isViewerIdentity);

        // then
        expect(results).toEqual([false, false]);
    });
});

describe("selectViewers", () => {
    it("keeps only the viewers, in the order they arrived", () => {
        // given
        const participants = [
            participant("broadcaster_1"),
            participant("viewer_1"),
            participant("monitor_1"),
            participant("viewer_2"),
        ];

        // when
        const viewers = selectViewers(participants);

        // then
        expect(viewers.map(viewer => viewer.identity)).toEqual(["viewer_1", "viewer_2"]);
    });
});

describe("countViewers", () => {
    it("counts viewers only", () => {
        // given
        const participants = [participant("broadcaster_1"), participant("viewer_1"), participant("monitor_1")];

        // when
        const count = countViewers(participants);

        // then
        expect(count).toBe(1);
    });
});

describe("readViewerMeta", () => {
    it("parses the metadata through the supplied resolver", () => {
        // given
        const viewer = participant("viewer_1", "Beatrice", { userId: "u1", avatarUrl: "/uploads/a.png" });

        // when
        const meta = readViewerMeta(viewer, value => ({ ...value, avatarUrl: `https://cdn.test${value.avatarUrl}` }));

        // then
        expect(meta).toEqual({ userId: "u1", avatarUrl: "https://cdn.test/uploads/a.png" });
    });

    it("returns nothing for a participant with no metadata", () => {
        // given
        const viewer = participant("viewer_1", "Beatrice");

        // when
        const meta = readViewerMeta(viewer, value => value);

        // then
        expect(meta).toBeNull();
    });

    it("returns nothing when the metadata is not json", () => {
        // given
        const viewer = participant("viewer_1", "Beatrice", "{ broken");

        // when
        const meta = readViewerMeta(viewer, value => value);

        // then
        expect(meta).toBeNull();
    });

    it("returns nothing when the resolver itself throws", () => {
        // given
        const viewer = participant("viewer_1", "Beatrice", { userId: "u1" });

        // when
        const meta = readViewerMeta(viewer, () => {
            throw new Error("resolver exploded");
        });

        // then
        expect(meta).toBeNull();
    });
});

describe("summariseViewers", () => {
    it("splits identified members from anonymous guests", () => {
        // given
        const participants = [
            participant("broadcaster_1", "Beatrice"),
            participant("viewer_1", "Battler", { userId: "u1" }),
            participant("viewer_2"),
            participant("viewer_3"),
        ];

        // when
        const roster = summariseViewers(participants);

        // then
        expect(roster).toEqual({
            total: 3,
            named: [{ userId: "u1", name: "Battler", avatar: undefined }],
            guests: 2,
        });
    });

    it("prefers the livekit participant name, then the metadata username, then a generic label", () => {
        // given
        const participants = [
            participant("viewer_1", "Battler", { userId: "u1", username: "battler18" }),
            participant("viewer_2", undefined, { userId: "u2", username: "bern" }),
            participant("viewer_3", undefined, { userId: "u3" }),
        ];

        // when
        const roster = summariseViewers(participants);

        // then
        expect(roster.named.map(chip => chip.name)).toEqual(["Battler", "bern", UNNAMED_VIEWER]);
    });

    it("collapses one member watching from two tabs into a single chip while still counting both", () => {
        // given
        const participants = [
            participant("viewer_1", "Battler", { userId: "u1" }),
            participant("viewer_2", "Battler on mobile", { userId: "u1" }),
        ];

        // when
        const roster = summariseViewers(participants);

        // then
        expect(roster.total).toBe(2);
        expect(roster.named).toEqual([{ userId: "u1", name: "Battler on mobile", avatar: undefined }]);
    });

    it("counts a viewer with unreadable metadata as a guest", () => {
        // given
        const participants = [participant("viewer_1", "Battler", "{ broken")];

        // when
        const roster = summariseViewers(participants);

        // then
        expect(roster).toEqual({ total: 1, named: [], guests: 1 });
    });

    it("ignores the owner's own monitor join entirely", () => {
        // given
        const participants = [
            participant("monitor_1", "Beatrice", { userId: "owner" }),
            participant("viewer_1", "Battler", { userId: "u1" }),
        ];

        // when
        const roster = summariseViewers(participants);

        // then
        expect(roster.total).toBe(1);
        expect(roster.named.map(chip => chip.userId)).toEqual(["u1"]);
    });

    it("passes each viewer's metadata through the resolver so avatars can be absolutised", () => {
        // given
        const participants = [participant("viewer_1", "Battler", { userId: "u1", avatarUrl: "/uploads/a.png" })];

        // when
        const roster = summariseViewers(participants, meta => ({
            ...meta,
            avatarUrl: `https://cdn.test${meta.avatarUrl}`,
        }));

        // then
        expect(roster.named[0].avatar).toBe("https://cdn.test/uploads/a.png");
    });
});
