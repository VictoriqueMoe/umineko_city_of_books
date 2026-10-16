import { describe, expect, it } from "vitest";
import { chatThreadPath, messagePermalink, type ChatMessageTarget, type ChatThreadKind } from "./permalink";

interface ThreadPathCase {
    name: string;
    kind: ChatThreadKind;
    target?: ChatMessageTarget;
    want: string;
}

describe("chatThreadPath", () => {
    const cases: ThreadPathCase[] = [
        { name: "opens a chat room", kind: "room", want: "/rooms/room-7" },
        { name: "opens a private thread", kind: "dm", want: "/chat/room-7" },
        {
            name: "anchors a chat room on one message",
            kind: "room",
            target: { id: "m9" },
            want: "/rooms/room-7#msg-m9",
        },
        {
            name: "anchors a private thread on one message",
            kind: "dm",
            target: { id: "m9" },
            want: "/chat/room-7#msg-m9",
        },
        {
            name: "carries the message's time ahead of the anchor so old history opens in one step",
            kind: "room",
            target: { id: "m9", createdAt: "2026-08-02T10:00:00Z" },
            want: "/rooms/room-7?at=2026-08-02T10%3A00%3A00Z#msg-m9",
        },
        {
            name: "escapes a time zone offset the way the search links do",
            kind: "dm",
            target: { id: "m9", createdAt: "2026-08-02T10:00:00+01:00" },
            want: "/chat/room-7?at=2026-08-02T10%3A00%3A00%2B01%3A00#msg-m9",
        },
    ];

    for (const testCase of cases) {
        it(testCase.name, () => {
            // given
            const roomId = "room-7";

            // when
            const path = chatThreadPath(testCase.kind, roomId, testCase.target);

            // then
            expect(path).toBe(testCase.want);
        });
    }
});

interface PermalinkCase {
    name: string;
    type: "dm" | "group";
    want: string;
}

describe("messagePermalink", () => {
    const cases: PermalinkCase[] = [
        {
            name: "points a group room message at the rooms page",
            type: "group",
            want: "/rooms/room-7?at=2026-08-02T10%3A00%3A00Z#msg-m9",
        },
        {
            name: "points a private message at the conversations page",
            type: "dm",
            want: "/chat/room-7?at=2026-08-02T10%3A00%3A00Z#msg-m9",
        },
    ];

    for (const testCase of cases) {
        it(testCase.name, () => {
            // given
            const room = { id: "room-7", type: testCase.type };
            const message = { id: "m9", created_at: "2026-08-02T10:00:00Z" };

            // when
            const path = messagePermalink(room, message);

            // then
            expect(path).toBe(testCase.want);
        });
    }
});
