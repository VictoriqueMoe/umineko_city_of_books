import { describe, expect, it } from "vitest";
import { makeChatRoom, makeDmRoom } from "../../test-utils/fixtures";
import type { ChatRoom, SiteRole } from "../../types/api";
import {
    isTimeoutActive,
    roomCapabilities,
    roomViewerPolicy,
    type RoomCapabilities,
    type RoomPolicyViewer,
} from "./roomPolicy";

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({ name: "Rokkenjima", member_count: 3, ...overrides });
}

function makePair(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({ member_count: 2, ...overrides });
}

describe("roomViewerPolicy", () => {
    it("makes the room host a moderator of the room", () => {
        // given
        const room = makeRoom({ viewer_role: "host" });

        // when
        const policy = roomViewerPolicy(room, { role: undefined });

        // then
        expect(policy).toEqual({ isHost: true, isSystem: false, isSiteMod: false, canModerateRoom: true });
    });

    it("makes an ordinary member no kind of moderator", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        const policy = roomViewerPolicy(room, { role: undefined });

        // then
        expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: false, canModerateRoom: false });
    });

    const staffRoles: SiteRole[] = ["super_admin", "admin", "moderator"];

    for (const role of staffRoles) {
        it(`makes a ${role} a moderator of a room they only joined`, () => {
            // given
            const room = makeRoom({ viewer_role: "member" });

            // when
            const policy = roomViewerPolicy(room, { role });

            // then
            expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: true, canModerateRoom: true });
        });
    }

    it("reports a system room as one, without withdrawing moderation", () => {
        // given
        const room = makeRoom({ is_system: true, viewer_role: "host" });

        // when
        const policy = roomViewerPolicy(room, { role: "moderator" });

        // then
        expect(policy).toEqual({ isHost: true, isSystem: true, isSiteMod: true, canModerateRoom: true });
    });

    it("treats a room with no viewer_role as one the viewer does not host", () => {
        // given
        const room = makeRoom({ viewer_role: undefined });

        // when
        const policy = roomViewerPolicy(room, { role: "admin" });

        // then
        expect(policy.isHost).toBe(false);
        expect(policy.canModerateRoom).toBe(true);
    });

    it("still reports the site role when the room has not loaded", () => {
        // given
        const room = null;

        // when
        const policy = roomViewerPolicy(room, { role: "admin" });

        // then
        expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: true, canModerateRoom: true });
    });

    it("denies everything when there is no viewer", () => {
        // given
        const room = makeRoom({ viewer_role: "member" });

        // when
        const policy = roomViewerPolicy(room, null);

        // then
        expect(policy).toEqual({ isHost: false, isSystem: false, isSiteMod: false, canModerateRoom: false });
    });
});

describe("roomCapabilities, every flag of each room type", () => {
    const shapeCases: { name: string; room: ChatRoom; expected: RoomCapabilities }[] = [
        {
            name: "a participant looking at their own private thread",
            room: makePair({ viewer_role: "member" }),
            expected: {
                kind: "pair",
                isHost: false,
                isSystem: false,
                isSiteMod: false,
                isMember: true,

                identity: "pair_key",
                displayName: "other_member",
                roles: "none",
                moderation: "site_staff",

                canModerateRoom: false,
                canDeleteOthersMessages: false,
                canPinMessages: true,
                canEditSettings: false,
                canInvite: false,
                canSelfJoin: false,
                canStartWatchParty: false,
                canTimeOutMembers: false,
                canDestroyRoom: false,

                isDiscoverable: false,
                hasPublicPreview: false,
                isAudited: false,
                archivesWhenStale: false,
                leaveMeansHide: true,
                destroysWhenEmpty: true,
                countsTowardsUnreadBadge: true,
                newThreadGatedByDmsEnabled: true,

                blockGatesSending: "symmetric",
                readReceipts: "pairwise",
                bannedWordAction: "delete",
                botTrigger: "every_message",
                botPromptScope: "whole_history",
                botCooldown: true,
            },
        },
        {
            name: "an ordinary member of a public group room",
            room: makeRoom({ viewer_role: "member" }),
            expected: {
                kind: "group",
                isHost: false,
                isSystem: false,
                isSiteMod: false,
                isMember: true,

                identity: "row_id",
                displayName: "owned",
                roles: "host_and_member",
                moderation: "host_and_site_staff",

                canModerateRoom: false,
                canDeleteOthersMessages: false,
                canPinMessages: false,
                canEditSettings: false,
                canInvite: false,
                canSelfJoin: true,
                canStartWatchParty: true,
                canTimeOutMembers: false,
                canDestroyRoom: false,

                isDiscoverable: true,
                hasPublicPreview: true,
                isAudited: true,
                archivesWhenStale: true,
                leaveMeansHide: false,
                destroysWhenEmpty: false,
                countsTowardsUnreadBadge: false,
                newThreadGatedByDmsEnabled: false,

                blockGatesSending: "from_host",
                readReceipts: "none",
                bannedWordAction: "delete_and_kick",
                botTrigger: "mention_or_reply",
                botPromptScope: "reply_chain",
                botCooldown: true,
            },
        },
    ];

    for (const shapeCase of shapeCases) {
        it(`describes ${shapeCase.name}`, () => {
            // given
            const room = shapeCase.room;

            // when
            const caps = roomCapabilities(room, { role: undefined });

            // then
            expect(caps).toEqual(shapeCase.expected);
        });
    }
});

describe("roomCapabilities, who may act on the room", () => {
    type ActionFlags = Pick<
        RoomCapabilities,
        | "canModerateRoom"
        | "canDeleteOthersMessages"
        | "canPinMessages"
        | "canTimeOutMembers"
        | "canEditSettings"
        | "canInvite"
        | "canDestroyRoom"
        | "canStartWatchParty"
        | "canSelfJoin"
    >;

    const actionCases: { name: string; room: ChatRoom; viewer: RoomPolicyViewer | null; expected: ActionFlags }[] = [
        {
            name: "gives a participant nothing but the pin on their own thread",
            room: makePair({ viewer_role: "member" }),
            viewer: { role: undefined },
            expected: {
                canModerateRoom: false,
                canDeleteOthersMessages: false,
                canPinMessages: true,
                canTimeOutMembers: false,
                canEditSettings: false,
                canInvite: false,
                canDestroyRoom: false,
                canStartWatchParty: false,
                canSelfJoin: false,
            },
        },
        {
            name: "lets site staff moderate inside a pair without gaining room settings",
            room: makePair({ viewer_role: "member" }),
            viewer: { role: "moderator" },
            expected: {
                canModerateRoom: true,
                canDeleteOthersMessages: true,
                canPinMessages: true,
                canTimeOutMembers: true,
                canEditSettings: false,
                canInvite: false,
                canDestroyRoom: false,
                canStartWatchParty: false,
                canSelfJoin: false,
            },
        },
        {
            name: "gives a stranger to a pair nothing at all",
            room: makePair({ viewer_role: undefined, is_member: false }),
            viewer: { role: undefined },
            expected: {
                canModerateRoom: false,
                canDeleteOthersMessages: false,
                canPinMessages: false,
                canTimeOutMembers: false,
                canEditSettings: false,
                canInvite: false,
                canDestroyRoom: false,
                canStartWatchParty: false,
                canSelfJoin: false,
            },
        },
        {
            name: "gives a group host the whole room",
            room: makeRoom({ viewer_role: "host" }),
            viewer: { role: undefined },
            expected: {
                canModerateRoom: true,
                canDeleteOthersMessages: true,
                canPinMessages: true,
                canTimeOutMembers: true,
                canEditSettings: true,
                canInvite: true,
                canDestroyRoom: true,
                canStartWatchParty: true,
                canSelfJoin: true,
            },
        },
        {
            name: "gives an ordinary group member only the room they can already join",
            room: makeRoom({ viewer_role: "member" }),
            viewer: { role: undefined },
            expected: {
                canModerateRoom: false,
                canDeleteOthersMessages: false,
                canPinMessages: false,
                canTimeOutMembers: false,
                canEditSettings: false,
                canInvite: false,
                canDestroyRoom: false,
                canStartWatchParty: true,
                canSelfJoin: true,
            },
        },
        {
            name: "gives site staff a group room they only joined",
            room: makeRoom({ viewer_role: "member" }),
            viewer: { role: "admin" },
            expected: {
                canModerateRoom: true,
                canDeleteOthersMessages: true,
                canPinMessages: true,
                canTimeOutMembers: true,
                canEditSettings: true,
                canInvite: true,
                canDestroyRoom: true,
                canStartWatchParty: true,
                canSelfJoin: true,
            },
        },
        {
            name: "withholds settings, invites, timeouts and destruction inside a system room",
            room: makeRoom({ is_system: true, system_kind: "mods", is_public: false, viewer_role: "host" }),
            viewer: { role: "admin" },
            expected: {
                canModerateRoom: true,
                canDeleteOthersMessages: true,
                canPinMessages: true,
                canTimeOutMembers: false,
                canEditSettings: false,
                canInvite: false,
                canDestroyRoom: false,
                canStartWatchParty: false,
                canSelfJoin: false,
            },
        },
        {
            name: "keeps a private group room out of reach of somebody who has not joined",
            room: makeRoom({ is_public: false, is_member: false, viewer_role: undefined }),
            viewer: { role: undefined },
            expected: {
                canModerateRoom: false,
                canDeleteOthersMessages: false,
                canPinMessages: false,
                canTimeOutMembers: false,
                canEditSettings: false,
                canInvite: false,
                canDestroyRoom: false,
                canStartWatchParty: false,
                canSelfJoin: false,
            },
        },
        {
            name: "denies every moderator action when there is no viewer at all",
            room: makeRoom({ viewer_role: "member" }),
            viewer: null,
            expected: {
                canModerateRoom: false,
                canDeleteOthersMessages: false,
                canPinMessages: false,
                canTimeOutMembers: false,
                canEditSettings: false,
                canInvite: false,
                canDestroyRoom: false,
                canStartWatchParty: true,
                canSelfJoin: true,
            },
        },
    ];

    for (const actionCase of actionCases) {
        it(actionCase.name, () => {
            // given
            const { room, viewer } = actionCase;

            // when
            const caps = roomCapabilities(room, viewer);

            // then
            expect(caps).toMatchObject(actionCase.expected);
        });
    }

    it("ignores a host role on a pair, because a pair has no roles", () => {
        // given
        const room = makePair({ viewer_role: "host" });

        // when
        const caps = roomCapabilities(room, { role: undefined });

        // then
        expect(caps.roles).toBe("none");
        expect(caps.canModerateRoom).toBe(false);
        expect(caps.canDeleteOthersMessages).toBe(false);
    });
});

describe("roomCapabilities, the owner's answered questions", () => {
    it("lets either participant pin their own thread while a group room still wants a host", () => {
        // given
        const pair = makePair({ viewer_role: "member" });
        const group = makeRoom({ viewer_role: "member" });

        // when
        const pairCaps = roomCapabilities(pair, { role: undefined });
        const groupCaps = roomCapabilities(group, { role: undefined });

        // then
        expect(pairCaps.canPinMessages).toBe(true);
        expect(pairCaps.canModerateRoom).toBe(false);
        expect(groupCaps.canPinMessages).toBe(false);
    });

    it("keeps read receipts out of group rooms deliberately, so the group rendering stays unreachable", () => {
        // given
        const pair = makePair();
        const group = makeRoom();

        // when
        const pairCaps = roomCapabilities(pair, { role: undefined });
        const groupCaps = roomCapabilities(group, { role: undefined });

        // then
        expect(pairCaps.readReceipts).toBe("pairwise");
        expect(groupCaps.readReceipts).toBe("none");
    });

    it("counts only a pair towards the direct-message bell", () => {
        // given
        const pair = makePair();
        const group = makeRoom();

        // when
        const pairCaps = roomCapabilities(pair, { role: undefined });
        const groupCaps = roomCapabilities(group, { role: undefined });

        // then
        expect(pairCaps.countsTowardsUnreadBadge).toBe(true);
        expect(groupCaps.countsTowardsUnreadBadge).toBe(false);
    });

    it("gates a new pair on the recipient's dms_enabled setting and never a group room", () => {
        // given
        const pair = makePair();
        const group = makeRoom();

        // when
        const pairCaps = roomCapabilities(pair, { role: undefined });
        const groupCaps = roomCapabilities(group, { role: undefined });

        // then
        expect(pairCaps.newThreadGatedByDmsEnabled).toBe(true);
        expect(groupCaps.newThreadGatedByDmsEnabled).toBe(false);
    });

    it("deletes a banned word inside a pair instead of evicting the sender from their own conversation", () => {
        // given
        const pair = makePair();
        const group = makeRoom();

        // when
        const pairCaps = roomCapabilities(pair, { role: undefined });
        const groupCaps = roomCapabilities(group, { role: undefined });

        // then
        expect(pairCaps.bannedWordAction).toBe("delete");
        expect(groupCaps.bannedWordAction).toBe("delete_and_kick");
    });

    it("applies the bot cooldown in a pair as well as a group room", () => {
        // given
        const pair = makePair();
        const group = makeRoom();

        // when
        const pairCaps = roomCapabilities(pair, { role: undefined });
        const groupCaps = roomCapabilities(group, { role: undefined });

        // then
        expect(pairCaps.botCooldown).toBe(true);
        expect(groupCaps.botCooldown).toBe(true);
    });

    it("states that moderation inside a pair belongs to site staff and nobody else", () => {
        // given
        const pair = makePair({ viewer_role: "member" });

        // when
        const staffCaps = roomCapabilities(pair, { role: "moderator" });
        const participantCaps = roomCapabilities(pair, { role: undefined });

        // then
        expect(staffCaps.moderation).toBe("site_staff");
        expect(staffCaps.canDeleteOthersMessages).toBe(true);
        expect(participantCaps.canDeleteOthersMessages).toBe(false);
    });

    it("never audits a pair, whatever the viewer is", () => {
        // given
        const pair = makePair();

        // when
        const caps = roomCapabilities(pair, { role: "super_admin" });

        // then
        expect(caps.isAudited).toBe(false);
    });
});

describe("roomCapabilities, the block gate", () => {
    const gateCases: { name: string; room: ChatRoom; expected: RoomCapabilities["blockGatesSending"] }[] = [
        {
            name: "a pair gates in either direction",
            room: makePair(),
            expected: "symmetric",
        },
        {
            name: "an ordinary group room gates from the host only",
            room: makeRoom(),
            expected: "from_host",
        },
        {
            name: "a live stream room keeps the host gate",
            room: makeRoom({ is_system: true, system_kind: "live_stream" }),
            expected: "from_host",
        },
        {
            name: "a watch party room keeps the host gate",
            room: makeRoom({ is_system: true, system_kind: "watch_party" }),
            expected: "from_host",
        },
        {
            name: "a staff system room has no host to gate on",
            room: makeRoom({ is_system: true, system_kind: "mods" }),
            expected: "none",
        },
        {
            name: "a system room with no kind has no host to gate on",
            room: makeRoom({ is_system: true }),
            expected: "none",
        },
    ];

    for (const gateCase of gateCases) {
        it(gateCase.name, () => {
            // given
            const room = gateCase.room;

            // when
            const caps = roomCapabilities(room, { role: undefined });

            // then
            expect(caps.blockGatesSending).toBe(gateCase.expected);
        });
    }
});

describe("roomCapabilities, discovery and lifecycle", () => {
    type LifecycleFlags = Pick<
        RoomCapabilities,
        | "isDiscoverable"
        | "hasPublicPreview"
        | "isAudited"
        | "archivesWhenStale"
        | "leaveMeansHide"
        | "destroysWhenEmpty"
    >;

    const lifecycleCases: { name: string; room: ChatRoom; expected: LifecycleFlags }[] = [
        {
            name: "a public group room is listed, previewed, audited and archived when it goes quiet",
            room: makeRoom(),
            expected: {
                isDiscoverable: true,
                hasPublicPreview: true,
                isAudited: true,
                archivesWhenStale: true,
                leaveMeansHide: false,
                destroysWhenEmpty: false,
            },
        },
        {
            name: "a private group room is hidden from the list, the preview and the audit log",
            room: makeRoom({ is_public: false }),
            expected: {
                isDiscoverable: false,
                hasPublicPreview: false,
                isAudited: false,
                archivesWhenStale: true,
                leaveMeansHide: false,
                destroysWhenEmpty: false,
            },
        },
        {
            name: "an archived public group room drops out of the list but keeps its preview",
            room: makeRoom({ archived_at: "2026-08-01T00:00:00Z" }),
            expected: {
                isDiscoverable: false,
                hasPublicPreview: true,
                isAudited: true,
                archivesWhenStale: true,
                leaveMeansHide: false,
                destroysWhenEmpty: false,
            },
        },
        {
            name: "a system room is never listed, previewed, audited or archived",
            room: makeRoom({ is_system: true, system_kind: "mods" }),
            expected: {
                isDiscoverable: false,
                hasPublicPreview: false,
                isAudited: false,
                archivesWhenStale: false,
                leaveMeansHide: false,
                destroysWhenEmpty: false,
            },
        },
        {
            name: "a pair hides itself when one side leaves and dies once nobody is left",
            room: makePair(),
            expected: {
                isDiscoverable: false,
                hasPublicPreview: false,
                isAudited: false,
                archivesWhenStale: false,
                leaveMeansHide: true,
                destroysWhenEmpty: true,
            },
        },
        {
            name: "a pair stays private even if the row claims to be public",
            room: makePair({ is_public: true }),
            expected: {
                isDiscoverable: false,
                hasPublicPreview: false,
                isAudited: false,
                archivesWhenStale: false,
                leaveMeansHide: true,
                destroysWhenEmpty: true,
            },
        },
    ];

    for (const lifecycleCase of lifecycleCases) {
        it(lifecycleCase.name, () => {
            // given
            const room = lifecycleCase.room;

            // when
            const caps = roomCapabilities(room, { role: undefined });

            // then
            expect(caps).toMatchObject(lifecycleCase.expected);
        });
    }
});

describe("roomCapabilities, a room that has not loaded", () => {
    const absentRooms: { name: string; room: ChatRoom | null | undefined }[] = [
        { name: "null", room: null },
        { name: "undefined", room: undefined },
    ];

    for (const absentRoom of absentRooms) {
        it(`reads ${absentRoom.name} as a group room the viewer is not in`, () => {
            // given
            const absent = absentRoom.room;

            // when
            const caps = roomCapabilities(absent, { role: undefined });

            // then
            expect(caps.kind).toBe("group");
            expect(caps.isMember).toBe(false);
            expect(caps.canModerateRoom).toBe(false);
            expect(caps.canSelfJoin).toBe(false);
            expect(caps.readReceipts).toBe("none");
        });
    }

    it("still reports the site role before the room arrives", () => {
        // given
        const room = null;

        // when
        const caps = roomCapabilities(room, { role: "admin" });

        // then
        expect(caps.isSiteMod).toBe(true);
        expect(caps.canModerateRoom).toBe(true);
        expect(caps.canEditSettings).toBe(true);
    });
});

describe("roomCapabilities keeps the four fields roomViewerPolicy already returned", () => {
    const migrationCases: { name: string; room: ChatRoom | null; viewer: RoomPolicyViewer | null }[] = [
        { name: "a group host", room: makeRoom({ viewer_role: "host" }), viewer: { role: undefined } },
        { name: "an ordinary group member", room: makeRoom({ viewer_role: "member" }), viewer: { role: undefined } },
        { name: "site staff in a group room", room: makeRoom({ viewer_role: "member" }), viewer: { role: "admin" } },
        { name: "a system room host", room: makeRoom({ is_system: true, viewer_role: "host" }), viewer: null },
        { name: "a participant in a pair", room: makePair({ viewer_role: "member" }), viewer: { role: undefined } },
        { name: "site staff in a pair", room: makePair({ viewer_role: "member" }), viewer: { role: "moderator" } },
        { name: "a room that has not loaded", room: null, viewer: { role: "admin" } },
    ];

    for (const migrationCase of migrationCases) {
        it(`agrees with the old policy for ${migrationCase.name}`, () => {
            // given
            const { room, viewer } = migrationCase;

            // when
            const caps = roomCapabilities(room, viewer);
            const policy = roomViewerPolicy(room, viewer);

            // then
            expect(caps).toMatchObject(policy);
        });
    }
});

describe("isTimeoutActive", () => {
    const now = Date.parse("2026-08-28T12:00:00Z");

    it("is active while the timeout is still in the future", () => {
        // given
        const until = "2026-08-28T12:00:01Z";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(true);
    });

    it("is over once the instant is reached, not a tick later", () => {
        // given
        const until = "2026-08-28T12:00:00Z";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(false);
    });

    it("is over when the timeout is in the past", () => {
        // given
        const until = "2026-08-28T11:59:59Z";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(false);
    });

    it("reads a timezone-less server timestamp as UTC rather than local time", () => {
        // given
        const until = "2026-08-28 12:00:01";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(true);
    });

    const absentValues: (string | null | undefined)[] = [undefined, null, "", "   "];

    for (const until of absentValues) {
        it(`is inactive for ${JSON.stringify(until)}`, () => {
            // given
            const timeoutUntil = until;

            // when
            const active = isTimeoutActive(timeoutUntil, now);

            // then
            expect(active).toBe(false);
        });
    }

    it("is inactive for an unparseable timestamp rather than throwing", () => {
        // given
        const until = "not a date";

        // when
        const active = isTimeoutActive(until, now);

        // then
        expect(active).toBe(false);
    });

    it("takes now as a parameter, so the same timeout expires as the clock is advanced", () => {
        // given
        const until = "2026-08-28T12:30:00Z";

        // when
        const before = isTimeoutActive(until, now);
        const after = isTimeoutActive(until, now + 31 * 60 * 1000);

        // then
        expect(before).toBe(true);
        expect(after).toBe(false);
    });
});
