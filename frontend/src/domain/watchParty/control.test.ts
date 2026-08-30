import { describe, expect, it } from "vitest";
import type { SiteRole, User, WatchPartyParticipant } from "../../types/api";
import {
    effectiveRank,
    watchPartyControlContext,
    watchPartyRowControls,
    type WatchPartyControlContext,
    type WatchPartyControlViewer,
    type WatchPartyRowControls,
} from "./control";

const VIEWER_ID = "user-viewer";
const OTHER_ID = "user-other";
const OWNER_ID = "user-owner";

function makeUser(id: string, role?: SiteRole): User {
    return { id, username: id, display_name: id, role };
}

function participant(id: string, hasControl = false, role?: SiteRole): WatchPartyParticipant {
    return { user: makeUser(id, role), has_control: hasControl, joined_at: "2026-08-01T10:00:00Z" };
}

function viewer(overrides: Partial<WatchPartyControlViewer> = {}): WatchPartyControlViewer {
    return { userId: VIEWER_ID, role: undefined, hasControl: false, ...overrides };
}

interface RankCase {
    name: string;
    role: SiteRole | undefined;
    isOwner: boolean;
    expected: number;
}

const rankCases: RankCase[] = [
    { name: "a super admin outranks everyone", role: "super_admin", isOwner: false, expected: 4 },
    { name: "an admin sits below a super admin", role: "admin", isOwner: false, expected: 3 },
    { name: "a moderator sits below an admin", role: "moderator", isOwner: false, expected: 2 },
    { name: "a watcher with no site role has no rank", role: undefined, isOwner: false, expected: 0 },
    { name: "owning the party lifts a plain watcher to one", role: undefined, isOwner: true, expected: 1 },
    { name: "owning the party does not lift a moderator", role: "moderator", isOwner: true, expected: 2 },
    { name: "owning the party does not lift a super admin", role: "super_admin", isOwner: true, expected: 4 },
];

describe("effectiveRank", () => {
    it.each(rankCases)("$name", ({ role, isOwner, expected }) => {
        // given the role and the ownership from the table row

        // when
        const rank = effectiveRank(role, isOwner);

        // then
        expect(rank).toBe(expected);
    });
});

describe("watchPartyControlContext", () => {
    it("lets anybody take control while nobody holds it", () => {
        // given
        const participants = [participant(VIEWER_ID), participant(OTHER_ID)];

        // when
        const context = watchPartyControlContext(participants, viewer(), OWNER_ID);

        // then
        expect(context).toEqual({ viewerRank: 0, canOutrankController: true });
    });

    it("lets a moderator outrank a plain watcher who holds control", () => {
        // given
        const participants = [participant(OTHER_ID, true)];

        // when
        const context = watchPartyControlContext(participants, viewer({ role: "moderator" }), OWNER_ID);

        // then
        expect(context).toEqual({ viewerRank: 2, canOutrankController: true });
    });

    it("refuses to outrank a controller of equal rank", () => {
        // given
        const participants = [participant(OTHER_ID, true, "moderator")];

        // when
        const context = watchPartyControlContext(participants, viewer({ role: "moderator" }), OWNER_ID);

        // then
        expect(context).toEqual({ viewerRank: 2, canOutrankController: false });
    });

    it("refuses to outrank a controller of higher rank", () => {
        // given
        const participants = [participant(OTHER_ID, true, "admin")];

        // when
        const context = watchPartyControlContext(participants, viewer({ role: "moderator" }), OWNER_ID);

        // then
        expect(context).toEqual({ viewerRank: 2, canOutrankController: false });
    });

    it("gives the party owner a rank of one when they hold no site role", () => {
        // given
        const participants = [participant(OTHER_ID, true)];

        // when
        const context = watchPartyControlContext(participants, viewer({ userId: OWNER_ID }), OWNER_ID);

        // then
        expect(context).toEqual({ viewerRank: 1, canOutrankController: true });
    });

    it("reads control from the first holder in the roster", () => {
        // given
        const participants = [participant(OTHER_ID, true, "admin"), participant(OWNER_ID, true)];

        // when
        const context = watchPartyControlContext(participants, viewer({ role: "moderator" }), OWNER_ID);

        // then
        expect(context).toEqual({ viewerRank: 2, canOutrankController: false });
    });
});

interface RowCase {
    name: string;
    participant: WatchPartyParticipant;
    viewer: WatchPartyControlViewer;
    ownerUserId?: string;
    context: WatchPartyControlContext;
    expected: WatchPartyRowControls;
}

const rowCases: RowCase[] = [
    {
        name: "the viewer may reclaim control from someone they outrank",
        participant: participant(VIEWER_ID),
        viewer: viewer(),
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: "Reclaim control", transferTarget: VIEWER_ID, canKick: false },
    },
    {
        name: "the viewer may not reclaim control from someone they do not outrank",
        participant: participant(VIEWER_ID),
        viewer: viewer(),
        context: { viewerRank: 2, canOutrankController: false },
        expected: { transferLabel: null, transferTarget: null, canKick: false },
    },
    {
        name: "the viewer is offered nothing on their own row while they hold control",
        participant: participant(VIEWER_ID, true),
        viewer: viewer({ hasControl: true }),
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: null, transferTarget: null, canKick: false },
    },
    {
        name: "the viewer may reclaim control from another watcher they outrank",
        participant: participant(OTHER_ID, true),
        viewer: viewer(),
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: "Reclaim", transferTarget: VIEWER_ID, canKick: true },
    },
    {
        name: "the viewer may not reclaim control from another watcher they do not outrank",
        participant: participant(OTHER_ID, true, "admin"),
        viewer: viewer(),
        context: { viewerRank: 2, canOutrankController: false },
        expected: { transferLabel: null, transferTarget: null, canKick: false },
    },
    {
        name: "the controller may pass control to a watcher without it",
        participant: participant(OTHER_ID),
        viewer: viewer({ hasControl: true }),
        context: { viewerRank: 0, canOutrankController: false },
        expected: { transferLabel: "Pass control", transferTarget: OTHER_ID, canKick: false },
    },
    {
        name: "a watcher who outranks the controller may pass control without holding it",
        participant: participant(OTHER_ID),
        viewer: viewer({ role: "moderator" }),
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: "Pass control", transferTarget: OTHER_ID, canKick: true },
    },
    {
        name: "a watcher with neither control nor rank is offered no transfer",
        participant: participant(OTHER_ID),
        viewer: viewer(),
        context: { viewerRank: 0, canOutrankController: false },
        expected: { transferLabel: null, transferTarget: null, canKick: false },
    },
    {
        name: "a moderator may kick a plain watcher",
        participant: participant(OTHER_ID),
        viewer: viewer({ role: "moderator", hasControl: true }),
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: "Pass control", transferTarget: OTHER_ID, canKick: true },
    },
    {
        name: "a moderator may not kick another moderator",
        participant: participant(OTHER_ID, false, "moderator"),
        viewer: viewer({ role: "moderator", hasControl: true }),
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: "Pass control", transferTarget: OTHER_ID, canKick: false },
    },
    {
        name: "a moderator may not kick an admin who owns the party",
        participant: participant(OWNER_ID, false, "admin"),
        viewer: viewer({ role: "moderator", hasControl: true }),
        ownerUserId: OWNER_ID,
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: "Pass control", transferTarget: OWNER_ID, canKick: false },
    },
    {
        name: "a moderator may kick the party owner when the owner holds no site role",
        participant: participant(OWNER_ID),
        viewer: viewer({ role: "moderator", hasControl: true }),
        ownerUserId: OWNER_ID,
        context: { viewerRank: 2, canOutrankController: true },
        expected: { transferLabel: "Pass control", transferTarget: OWNER_ID, canKick: true },
    },
    {
        name: "nobody may kick themselves",
        participant: participant(VIEWER_ID, false, "moderator"),
        viewer: viewer({ role: "super_admin" }),
        context: { viewerRank: 4, canOutrankController: true },
        expected: { transferLabel: "Reclaim control", transferTarget: VIEWER_ID, canKick: false },
    },
];

describe("watchPartyRowControls", () => {
    it.each(rowCases)("$name", ({ participant: row, viewer: rowViewer, ownerUserId, context, expected }) => {
        // given the row, the viewer and the control context from the table row

        // when
        const controls = watchPartyRowControls(row, rowViewer, ownerUserId ?? OWNER_ID, context);

        // then
        expect(controls).toEqual(expected);
    });
});
