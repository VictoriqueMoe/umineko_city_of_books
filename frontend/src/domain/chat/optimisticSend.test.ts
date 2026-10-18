import { describe, expect, it } from "vitest";
import { makeChatMessage, makePendingSend, makePublicUser } from "../../test-utils/fixtures";
import {
    addPendingSend,
    createClientNonce,
    createPendingSend,
    deliveredNonce,
    failSendingInThread,
    findPendingSend,
    isPendingSend,
    markPendingFailed,
    markPendingSending,
    mergeThread,
    moveThread,
    pendingInThread,
    pendingPreview,
    removePendingSend,
    sameThread,
    threadEntryKey,
    type SendThread,
} from "./optimisticSend";

const NONCE_A = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const NONCE_B = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
const NONCE_C = "cccccccccccccccccccccccccccccccc";

const ROOM_ONE: SendThread = { kind: "room", roomId: "room-1" };
const ROOM_TWO: SendThread = { kind: "room", roomId: "room-2" };
const DRAFT_TO_BATTLER: SendThread = { kind: "draft", recipientId: "u-battler" };

describe("createClientNonce", () => {
    it("produces a 32 character hex nonce the server accepts", () => {
        // when
        const nonce = createClientNonce();

        // then
        expect(nonce).toMatch(/^[0-9a-f]{32}$/);
    });

    it("produces a different nonce on every call", () => {
        // when
        const first = createClientNonce();
        const second = createClientNonce();

        // then
        expect(first).not.toBe(second);
    });
});

describe("createPendingSend", () => {
    it("keeps everything the viewer sent, pairs each attachment with its preview and starts as sending", () => {
        // given
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const replyTo = { id: "m-earlier", senderName: "Battler", body: "an earlier claim" };

        // when
        const pending = createPendingSend({
            nonce: NONCE_A,
            thread: ROOM_ONE,
            outgoing: { body: "I deny it", replyTo, attachments: [{ file, spoiler: true }] },
            previewUrls: ["blob:preview-1"],
            createdAt: "2026-10-16T10:00:00.000Z",
        });

        // then
        expect(pending).toEqual({
            nonce: NONCE_A,
            thread: ROOM_ONE,
            body: "I deny it",
            replyTo,
            attachments: [{ file, spoiler: true, previewUrl: "blob:preview-1" }],
            createdAt: "2026-10-16T10:00:00.000Z",
            delivery: "sending",
            error: null,
        });
    });
});

describe("pending list transitions", () => {
    it("appends a new pending send after the ones already waiting", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A })];

        // when
        const next = addPendingSend(waiting, makePendingSend({ nonce: NONCE_B }));

        // then
        expect(next.map(entry => entry.nonce)).toEqual([NONCE_A, NONCE_B]);
    });

    it("marks a send as failed with the reason and keeps its content", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A, body: "hello" })];

        // when
        const next = markPendingFailed(waiting, NONCE_A, "network is down");

        // then
        expect(next[0]).toMatchObject({ delivery: "failed", error: "network is down", body: "hello" });
    });

    it("puts a failed send back to sending for a retry and clears the reason", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A, delivery: "failed", error: "network is down" })];

        // when
        const next = markPendingSending(waiting, NONCE_A);

        // then
        expect(next[0]).toMatchObject({ nonce: NONCE_A, delivery: "sending", error: null });
    });

    it("removes only the send that was reconciled or discarded", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A }), makePendingSend({ nonce: NONCE_B })];

        // when
        const next = removePendingSend(waiting, NONCE_A);

        // then
        expect(next.map(entry => entry.nonce)).toEqual([NONCE_B]);
    });

    interface UnknownNonceCase {
        name: string;
        apply: (waiting: ReturnType<typeof makePendingSend>[]) => ReturnType<typeof makePendingSend>[];
    }

    const unknownNonceCases: UnknownNonceCase[] = [
        { name: "failing", apply: waiting => markPendingFailed(waiting, NONCE_B, "boom") },
        { name: "retrying", apply: waiting => markPendingSending(waiting, NONCE_B) },
        { name: "removing", apply: waiting => removePendingSend(waiting, NONCE_B) },
    ];

    for (const unknownNonceCase of unknownNonceCases) {
        it(`hands back the same list when ${unknownNonceCase.name} a nonce it does not hold`, () => {
            // given
            const waiting = [makePendingSend({ nonce: NONCE_A })];

            // when
            const next = unknownNonceCase.apply(waiting);

            // then
            expect(next).toBe(waiting);
        });
    }

    it("finds a pending send by its nonce and reports a missing one as null", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A })];

        // when
        const found = findPendingSend(waiting, NONCE_A);
        const missing = findPendingSend(waiting, NONCE_B);

        // then
        expect(found?.nonce).toBe(NONCE_A);
        expect(missing).toBeNull();
    });

    it("keeps only the sends that belong to the open room", () => {
        // given
        const waiting = [
            makePendingSend({ nonce: NONCE_A, thread: ROOM_ONE }),
            makePendingSend({ nonce: NONCE_B, thread: ROOM_TWO }),
        ];

        // when
        const inRoom = pendingInThread(waiting, ROOM_TWO);

        // then
        expect(inRoom.map(entry => entry.nonce)).toEqual([NONCE_B]);
    });

    it("keeps a draft's sends to that draft and out of every room", () => {
        // given
        const waiting = [
            makePendingSend({ nonce: NONCE_A, thread: ROOM_ONE }),
            makePendingSend({ nonce: NONCE_B, thread: DRAFT_TO_BATTLER }),
        ];

        // when
        const inDraft = pendingInThread(waiting, DRAFT_TO_BATTLER);

        // then
        expect(inDraft.map(entry => entry.nonce)).toEqual([NONCE_B]);
    });

    it("shows no sends while no thread is open", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A })];

        // when
        const inRoom = pendingInThread(waiting, null);

        // then
        expect(inRoom).toEqual([]);
    });
});

describe("sameThread", () => {
    interface SameThreadCase {
        name: string;
        a: SendThread;
        b: SendThread;
        want: boolean;
    }

    const cases: SameThreadCase[] = [
        { name: "the same room", a: ROOM_ONE, b: { kind: "room", roomId: "room-1" }, want: true },
        { name: "two different rooms", a: ROOM_ONE, b: ROOM_TWO, want: false },
        {
            name: "drafts to the same recipient",
            a: DRAFT_TO_BATTLER,
            b: { kind: "draft", recipientId: "u-battler" },
            want: true,
        },
        {
            name: "drafts to different recipients",
            a: DRAFT_TO_BATTLER,
            b: { kind: "draft", recipientId: "u-ange" },
            want: false,
        },
        {
            name: "a draft and a room sharing an id",
            a: { kind: "draft", recipientId: "room-1" },
            b: ROOM_ONE,
            want: false,
        },
    ];

    for (const sameThreadCase of cases) {
        it(`treats ${sameThreadCase.name} as ${sameThreadCase.want ? "one thread" : "different threads"}`, () => {
            // when
            const same = sameThread(sameThreadCase.a, sameThreadCase.b);

            // then
            expect(same).toBe(sameThreadCase.want);
        });
    }
});

describe("failSendingInThread", () => {
    it("fails every send still on its way in the thread, keeping their content, and leaves other threads alone", () => {
        // given
        const waiting = [
            makePendingSend({ nonce: NONCE_A, thread: DRAFT_TO_BATTLER, body: "first" }),
            makePendingSend({ nonce: NONCE_B, thread: DRAFT_TO_BATTLER, body: "second" }),
            makePendingSend({ nonce: NONCE_C, thread: ROOM_ONE, body: "elsewhere" }),
        ];

        // when
        const next = failSendingInThread(waiting, DRAFT_TO_BATTLER, "recipient has DMs disabled");

        // then
        expect(next.map(entry => [entry.body, entry.delivery, entry.error])).toEqual([
            ["first", "failed", "recipient has DMs disabled"],
            ["second", "failed", "recipient has DMs disabled"],
            ["elsewhere", "sending", null],
        ]);
    });

    it("keeps the reason a send already failed with", () => {
        // given
        const waiting = [
            makePendingSend({ nonce: NONCE_A, thread: DRAFT_TO_BATTLER, delivery: "failed", error: "earlier" }),
            makePendingSend({ nonce: NONCE_B, thread: DRAFT_TO_BATTLER }),
        ];

        // when
        const next = failSendingInThread(waiting, DRAFT_TO_BATTLER, "later");

        // then
        expect(next.map(entry => entry.error)).toEqual(["earlier", "later"]);
    });

    it("hands back the same list when nothing in the thread is on its way", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A, thread: ROOM_ONE })];

        // when
        const next = failSendingInThread(waiting, DRAFT_TO_BATTLER, "boom");

        // then
        expect(next).toBe(waiting);
    });
});

describe("moveThread", () => {
    it("moves every send of a draft into the room it became, in order, and leaves other threads alone", () => {
        // given
        const waiting = [
            makePendingSend({ nonce: NONCE_A, thread: DRAFT_TO_BATTLER }),
            makePendingSend({ nonce: NONCE_B, thread: ROOM_ONE }),
            makePendingSend({ nonce: NONCE_C, thread: DRAFT_TO_BATTLER, delivery: "failed", error: "boom" }),
        ];

        // when
        const next = moveThread(waiting, DRAFT_TO_BATTLER, ROOM_TWO);

        // then
        expect(next.map(entry => [entry.nonce, entry.thread, entry.delivery])).toEqual([
            [NONCE_A, ROOM_TWO, "sending"],
            [NONCE_B, ROOM_ONE, "sending"],
            [NONCE_C, ROOM_TWO, "failed"],
        ]);
    });

    it("hands back the same list when the draft holds no sends", () => {
        // given
        const waiting = [makePendingSend({ nonce: NONCE_A, thread: ROOM_ONE })];

        // when
        const next = moveThread(waiting, DRAFT_TO_BATTLER, ROOM_TWO);

        // then
        expect(next).toBe(waiting);
    });
});

describe("deliveredNonce", () => {
    const viewer = makePublicUser({ id: "viewer" });

    interface DeliveredCase {
        name: string;
        senderId: string;
        nonce: string | undefined;
        want: string | null;
    }

    const deliveredCases: DeliveredCase[] = [
        {
            name: "the viewer's own message with a nonce reconciles it",
            senderId: "viewer",
            nonce: NONCE_A,
            want: NONCE_A,
        },
        {
            name: "someone else's message never reconciles the viewer's sends",
            senderId: "other",
            nonce: NONCE_A,
            want: null,
        },
        { name: "a message without a nonce reconciles nothing", senderId: "viewer", nonce: undefined, want: null },
    ];

    for (const deliveredCase of deliveredCases) {
        it(deliveredCase.name, () => {
            // given
            const message = makeChatMessage({
                sender: makePublicUser({ id: deliveredCase.senderId }),
                client_nonce: deliveredCase.nonce,
            });

            // when
            const nonce = deliveredNonce(message, viewer.id);

            // then
            expect(nonce).toBe(deliveredCase.want);
        });
    }
});

describe("mergeThread", () => {
    const viewer = makePublicUser({ id: "viewer" });

    function idsOf(entries: ReturnType<typeof mergeThread>): string[] {
        return entries.map(entry => (isPendingSend(entry) ? entry.nonce : entry.id));
    }

    it("returns the server list itself when nothing is pending", () => {
        // given
        const messages = [makeChatMessage({ id: "m1" })];

        // when
        const merged = mergeThread(messages, [], viewer.id);

        // then
        expect(merged).toBe(messages);
    });

    it("places pending sends after every confirmed message", () => {
        // given
        const messages = [makeChatMessage({ id: "m1" })];
        const pending = [makePendingSend({ nonce: NONCE_A })];

        // when
        const merged = mergeThread(messages, pending, viewer.id);

        // then
        expect(idsOf(merged)).toEqual(["m1", NONCE_A]);
    });

    it("never shows a send twice once its confirmed message is in the list", () => {
        // given
        const messages = [makeChatMessage({ id: "m1", sender: viewer, client_nonce: NONCE_A })];
        const pending = [makePendingSend({ nonce: NONCE_A }), makePendingSend({ nonce: NONCE_B })];

        // when
        const merged = mergeThread(messages, pending, viewer.id);

        // then
        expect(idsOf(merged)).toEqual(["m1", NONCE_B]);
    });

    it("keeps a pending send when only someone else's message carries its nonce", () => {
        // given
        const messages = [
            makeChatMessage({ id: "m1", sender: makePublicUser({ id: "other" }), client_nonce: NONCE_A }),
        ];

        // when
        const merged = mergeThread(messages, [makePendingSend({ nonce: NONCE_A })], viewer.id);

        // then
        expect(idsOf(merged)).toEqual(["m1", NONCE_A]);
    });

    it("returns the server list itself when every pending send has already landed", () => {
        // given
        const messages = [makeChatMessage({ id: "m1", sender: viewer, client_nonce: NONCE_A })];

        // when
        const merged = mergeThread(messages, [makePendingSend({ nonce: NONCE_A })], viewer.id);

        // then
        expect(merged).toBe(messages);
    });
});

describe("threadEntryKey", () => {
    it("gives a confirmed message the key its pending send had, so the bubble is not remounted", () => {
        // given
        const pending = makePendingSend({ nonce: NONCE_A });
        const confirmed = makeChatMessage({ id: "m1", client_nonce: NONCE_A });

        // when
        const pendingKey = threadEntryKey(pending);
        const confirmedKey = threadEntryKey(confirmed);

        // then
        expect(confirmedKey).toBe(pendingKey);
    });

    it("keys a message sent without a nonce by its id", () => {
        // given
        const message = makeChatMessage({ id: "m1" });

        // when
        const key = threadEntryKey(message);

        // then
        expect(key).toBe("m1");
    });
});

describe("pendingPreview", () => {
    const viewer = makePublicUser({ id: "viewer", display_name: "Beatrice" });

    it("renders the pending send as the viewer's message with its reply and attachment previews", () => {
        // given
        const image = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const clip = new File(["clip"], "cat.mp4", { type: "video/mp4" });
        const pending = makePendingSend({
            nonce: NONCE_A,
            body: "look",
            replyTo: { id: "m-earlier", senderName: "Battler", body: "an earlier claim" },
            attachments: [
                { file: image, spoiler: false, previewUrl: "blob:image" },
                { file: clip, spoiler: true, previewUrl: "blob:clip" },
            ],
        });

        // when
        const preview = pendingPreview(pending, viewer, null);

        // then
        expect(preview).toMatchObject({
            id: NONCE_A,
            room_id: "room-1",
            sender: viewer,
            body: "look",
            client_nonce: NONCE_A,
            reply_to: { id: "m-earlier", sender_name: "Battler", body: "an earlier claim" },
            media: [
                { id: 0, media_url: "blob:image", media_type: "image", filename: "beatrice.png", is_spoiler: false },
                { id: 1, media_url: "blob:clip", media_type: "video", filename: "cat.mp4", is_spoiler: true },
            ],
            reactions: [],
            pinned: false,
        });
        expect(preview.sender_nickname).toBeUndefined();
    });

    it("belongs to no room while it is the first message of a draft", () => {
        // given
        const pending = makePendingSend({ nonce: NONCE_A, thread: DRAFT_TO_BATTLER });

        // when
        const preview = pendingPreview(pending, viewer, null);

        // then
        expect(preview.room_id).toBe("");
        expect(preview.client_nonce).toBe(NONCE_A);
    });

    it("wears the viewer's room nickname and member avatar where the room gives them one", () => {
        // given
        const pending = makePendingSend({ nonce: NONCE_A });

        // when
        const preview = pendingPreview(pending, viewer, { nickname: "Golden Witch", member_avatar_url: "/a.png" });

        // then
        expect(preview.sender_nickname).toBe("Golden Witch");
        expect(preview.sender_member_avatar_url).toBe("/a.png");
    });
});
