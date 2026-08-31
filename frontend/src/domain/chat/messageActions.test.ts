import { describe, expect, it } from "vitest";
import { messageActions, type MessageActionContext, type MessageActionId } from "./messageActions";

function context(overrides: Partial<MessageActionContext> = {}): MessageActionContext {
    return {
        pinned: false,
        isOwn: false,
        editing: false,
        senderIsStaff: false,
        canToggleReaction: false,
        canPin: false,
        canModerate: false,
        canEdit: true,
        hasReply: false,
        hasPinToggle: false,
        hasEdit: false,
        hasDelete: false,
        ...overrides,
    };
}

interface GateCase {
    name: string;
    overrides: Partial<MessageActionContext>;
    want: MessageActionId[];
}

const gateCases: GateCase[] = [
    { name: "offers nothing when the caller wired nothing up", overrides: {}, want: [] },
    {
        name: "offers reacting once the room allows it and a handler exists",
        overrides: { canToggleReaction: true },
        want: ["react"],
    },
    { name: "offers replying whenever a reply handler exists", overrides: { hasReply: true }, want: ["reply"] },
    {
        name: "withholds pinning from someone who may not pin",
        overrides: { canPin: false, hasPinToggle: true },
        want: [],
    },
    {
        name: "withholds pinning when the caller forgot the handler",
        overrides: { canPin: true, hasPinToggle: false },
        want: [],
    },
    {
        name: "offers pinning to someone who may pin",
        overrides: { canPin: true, hasPinToggle: true },
        want: ["pin"],
    },
    {
        name: "withholds editing on someone else's message",
        overrides: { isOwn: false, hasEdit: true },
        want: [],
    },
    {
        name: "withholds editing while the room no longer allows it",
        overrides: { isOwn: true, canEdit: false, hasEdit: true },
        want: [],
    },
    {
        name: "withholds editing while the message is already being edited",
        overrides: { isOwn: true, hasEdit: true, editing: true },
        want: [],
    },
    { name: "offers editing on your own message", overrides: { isOwn: true, hasEdit: true }, want: ["edit"] },
    { name: "offers deleting your own message", overrides: { isOwn: true, hasDelete: true }, want: ["delete"] },
    {
        name: "withholds deleting someone else's message from an ordinary member",
        overrides: { isOwn: false, canModerate: false, hasDelete: true },
        want: [],
    },
    {
        name: "offers deleting someone else's message to a moderator",
        overrides: { isOwn: false, canModerate: true, hasDelete: true },
        want: ["delete"],
    },
    {
        name: "stops a moderator deleting a staff member's message",
        overrides: { isOwn: false, canModerate: true, senderIsStaff: true, hasDelete: true },
        want: [],
    },
    {
        name: "still lets staff delete their own message",
        overrides: { isOwn: true, senderIsStaff: true, hasDelete: true },
        want: ["delete"],
    },
    {
        name: "withholds deleting when the caller forgot the handler",
        overrides: { isOwn: true, hasDelete: false },
        want: [],
    },
    {
        name: "gives a moderator on their own message the whole set in a fixed order",
        overrides: {
            isOwn: true,
            canToggleReaction: true,
            canPin: true,
            canModerate: true,
            hasReply: true,
            hasPinToggle: true,
            hasEdit: true,
            hasDelete: true,
        },
        want: ["react", "reply", "pin", "edit", "delete"],
    },
    {
        name: "gives a live stream viewer the thin set the panel wired up",
        overrides: { isOwn: true, hasReply: true, hasEdit: true },
        want: ["reply", "edit"],
    },
];

describe("messageActions", () => {
    for (const gateCase of gateCases) {
        it(gateCase.name, () => {
            // given
            const input = context(gateCase.overrides);

            // when
            const actions = messageActions(input);

            // then
            expect(actions.map(action => action.id)).toEqual(gateCase.want);
        });
    }

    interface LabelCase {
        name: string;
        pinned: boolean;
        want: string;
    }

    const labelCases: LabelCase[] = [
        { name: "offers to pin a message that is not pinned", pinned: false, want: "Pin message" },
        { name: "offers to unpin a message that is pinned", pinned: true, want: "Unpin message" },
    ];

    for (const labelCase of labelCases) {
        it(labelCase.name, () => {
            // given
            const input = context({ canPin: true, hasPinToggle: true, pinned: labelCase.pinned });

            // when
            const actions = messageActions(input);

            // then
            expect(actions[0].label).toBe(labelCase.want);
        });
    }

    it("marks deleting as the only destructive action", () => {
        // given
        const input = context({
            isOwn: true,
            canToggleReaction: true,
            canPin: true,
            hasReply: true,
            hasPinToggle: true,
            hasEdit: true,
            hasDelete: true,
        });

        // when
        const actions = messageActions(input);

        // then
        expect(actions.filter(action => action.destructive).map(action => action.id)).toEqual(["delete"]);
    });

    it("marks reacting as the only action that opens a further surface", () => {
        // given
        const input = context({
            isOwn: true,
            canToggleReaction: true,
            canPin: true,
            hasReply: true,
            hasPinToggle: true,
            hasEdit: true,
            hasDelete: true,
        });

        // when
        const actions = messageActions(input);

        // then
        expect(actions.filter(action => action.opensPicker).map(action => action.id)).toEqual(["react"]);
    });

    it("labels every action it offers", () => {
        // given
        const input = context({
            isOwn: true,
            canToggleReaction: true,
            canPin: true,
            hasReply: true,
            hasPinToggle: true,
            hasEdit: true,
            hasDelete: true,
        });

        // when
        const actions = messageActions(input);

        // then
        expect(actions.map(action => action.label)).toEqual([
            "React",
            "Reply",
            "Pin message",
            "Edit message",
            "Delete message",
        ]);
    });
});
