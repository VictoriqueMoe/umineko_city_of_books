import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { OutgoingChatMessage } from "../../domain/chat/optimisticSend";
import { makeChatMessage, makeChatRoom } from "../../test-utils/fixtures";
import type { ChatMessage, ChatRoom } from "../../types/api";
import type * as ChatMutations from "../mutations/chat";
import { useOptimisticSends, type OptimisticSendsOptions } from "./useOptimisticSends";

const mocks = vi.hoisted(() => ({
    sendChatMessage: vi.fn(),
    sendFirstDMMessage: vi.fn(),
}));

vi.mock("../mutations/chat", async importOriginal => {
    const actual = await importOriginal<typeof ChatMutations>();

    return {
        ...actual,
        useSendChatMessage: () => ({ mutateAsync: mocks.sendChatMessage }),
        useSendFirstDMMessage: () => ({ mutateAsync: mocks.sendFirstDMMessage }),
    };
});

interface Deferred<T> {
    resolve: (value: T) => void;
    reject: (error: unknown) => void;
}

interface FirstDelivery {
    room: ChatRoom;
    message: ChatMessage;
}

function deferSend(): Deferred<ChatMessage> {
    const deferred: Deferred<ChatMessage> = { resolve: () => {}, reject: () => {} };
    const response = new Promise<ChatMessage>((resolve, reject) => {
        deferred.resolve = resolve;
        deferred.reject = reject;
    });
    mocks.sendChatMessage.mockReturnValueOnce(response);

    return deferred;
}

function deferFirstSend(): Deferred<FirstDelivery> {
    const deferred: Deferred<FirstDelivery> = { resolve: () => {}, reject: () => {} };
    const response = new Promise<FirstDelivery>((resolve, reject) => {
        deferred.resolve = resolve;
        deferred.reject = reject;
    });
    mocks.sendFirstDMMessage.mockReturnValueOnce(response);

    return deferred;
}

function outgoing(overrides: Partial<OutgoingChatMessage> = {}): OutgoingChatMessage {
    return { body: "without love it cannot be seen", replyTo: null, attachments: [], ...overrides };
}

function renderSends(overrides: Partial<OptimisticSendsOptions> = {}) {
    const initialProps: OptimisticSendsOptions = {
        roomId: "room-1",
        onSending: vi.fn(),
        onDelivered: vi.fn(),
        ...overrides,
    };

    return renderHook((props: OptimisticSendsOptions) => useOptimisticSends(props), { initialProps });
}

beforeEach(() => {
    mocks.sendChatMessage.mockReset();
    mocks.sendFirstDMMessage.mockReset();
    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:preview");
    vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("useOptimisticSends", () => {
    it("shows the message at once, posts it with a fresh nonce and asks for the thread to scroll", () => {
        // given
        deferSend();
        const onSending = vi.fn();
        const { result } = renderSends({ onSending });
        const replyTo = { id: "m-earlier", senderName: "Battler", body: "an earlier claim" };

        // when
        act(() => {
            result.current.sender?.send(outgoing({ replyTo }));
        });

        // then
        const [pending] = result.current.pending;
        expect(pending).toMatchObject({
            thread: { kind: "room", roomId: "room-1" },
            body: "without love it cannot be seen",
            delivery: "sending",
            replyTo,
        });
        expect(pending.nonce).toMatch(/^[0-9a-f]{32}$/);
        expect(mocks.sendChatMessage).toHaveBeenCalledExactlyOnceWith({
            roomId: "room-1",
            body: "without love it cannot be seen",
            reply_to_id: "m-earlier",
            files: [],
            spoilers: [],
            client_nonce: pending.nonce,
        });
        expect(onSending).toHaveBeenCalledOnce();
    });

    it("lets a second message go out while the first is still in flight", () => {
        // given
        deferSend();
        deferSend();
        const { result } = renderSends();

        // when
        act(() => {
            result.current.sender?.send(outgoing({ body: "first" }));
        });
        act(() => {
            result.current.sender?.send(outgoing({ body: "second" }));
        });

        // then
        expect(result.current.pending.map(entry => entry.body)).toEqual(["first", "second"]);
        expect(mocks.sendChatMessage).toHaveBeenCalledTimes(2);
    });

    it("previews each attachment and sends the files with their spoiler flags", () => {
        // given
        deferSend();
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const { result } = renderSends();

        // when
        act(() => {
            result.current.sender?.send(outgoing({ body: "", attachments: [{ file, spoiler: true }] }));
        });

        // then
        expect(result.current.pending[0].attachments).toEqual([{ file, spoiler: true, previewUrl: "blob:preview" }]);
        expect(mocks.sendChatMessage).toHaveBeenCalledWith(
            expect.objectContaining({ files: [file], spoilers: [true] }),
        );
    });

    it("drops the pending send, frees its previews and hands over the confirmed message when the server answers", async () => {
        // given
        const send = deferSend();
        const onDelivered = vi.fn();
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const { result } = renderSends({ onDelivered });
        act(() => {
            result.current.sender?.send(outgoing({ attachments: [{ file, spoiler: false }] }));
        });
        const confirmed = makeChatMessage({ id: "m-real", client_nonce: result.current.pending[0].nonce });

        // when
        await act(async () => {
            send.resolve(confirmed);
        });

        // then
        expect(result.current.pending).toEqual([]);
        expect(onDelivered).toHaveBeenCalledExactlyOnceWith(confirmed);
        expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:preview");
    });

    it("keeps a failed send with all its content and the reason it failed", async () => {
        // given
        const send = deferSend();
        const { result } = renderSends();
        act(() => {
            result.current.sender?.send(outgoing({ body: "hello" }));
        });

        // when
        await act(async () => {
            send.reject(new Error("network is down"));
        });

        // then
        expect(result.current.pending).toHaveLength(1);
        expect(result.current.pending[0]).toMatchObject({
            body: "hello",
            delivery: "failed",
            error: "network is down",
        });
    });

    it("retries a failed send under the same nonce so the server can never store it twice", async () => {
        // given
        const first = deferSend();
        const { result } = renderSends();
        act(() => {
            result.current.sender?.send(outgoing());
        });
        const nonce = result.current.pending[0].nonce;
        await act(async () => {
            first.reject(new Error("network is down"));
        });
        const second = deferSend();

        // when
        act(() => {
            result.current.retry(nonce);
        });

        // then
        expect(result.current.pending[0]).toMatchObject({ nonce, delivery: "sending", error: null });
        expect(mocks.sendChatMessage).toHaveBeenLastCalledWith(expect.objectContaining({ client_nonce: nonce }));
        await act(async () => {
            second.resolve(makeChatMessage({ client_nonce: nonce }));
        });
        expect(result.current.pending).toEqual([]);
    });

    it("ignores a retry for a send that is still on its way", () => {
        // given
        deferSend();
        const { result } = renderSends();
        act(() => {
            result.current.sender?.send(outgoing());
        });

        // when
        act(() => {
            result.current.retry(result.current.pending[0].nonce);
        });

        // then
        expect(mocks.sendChatMessage).toHaveBeenCalledOnce();
    });

    it("discards a failed send and frees its previews", async () => {
        // given
        const send = deferSend();
        const file = new File(["portrait"], "beatrice.png", { type: "image/png" });
        const { result } = renderSends();
        act(() => {
            result.current.sender?.send(outgoing({ attachments: [{ file, spoiler: false }] }));
        });
        await act(async () => {
            send.reject(new Error("network is down"));
        });

        // when
        act(() => {
            result.current.discard(result.current.pending[0].nonce);
        });

        // then
        expect(result.current.pending).toEqual([]);
        expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:preview");
    });

    it("drops the pending send as soon as its echo is reconciled, before the request returns", async () => {
        // given
        const send = deferSend();
        const onDelivered = vi.fn();
        const { result } = renderSends({ onDelivered });
        act(() => {
            result.current.sender?.send(outgoing());
        });
        const nonce = result.current.pending[0].nonce;

        // when
        act(() => {
            result.current.reconcile(nonce);
        });

        // then
        expect(result.current.pending).toEqual([]);
        await act(async () => {
            send.resolve(makeChatMessage({ client_nonce: nonce }));
        });
        expect(result.current.pending).toEqual([]);
        expect(onDelivered).toHaveBeenCalledOnce();
    });

    it("never marks a send failed once its echo has already reconciled it", async () => {
        // given
        const send = deferSend();
        const { result } = renderSends();
        act(() => {
            result.current.sender?.send(outgoing());
        });
        act(() => {
            result.current.reconcile(result.current.pending[0].nonce);
        });

        // when
        await act(async () => {
            send.reject(new Error("the response was lost"));
        });

        // then
        expect(result.current.pending).toEqual([]);
    });

    it("keeps each room's pending sends to that room and brings them back on return", () => {
        // given
        deferSend();
        const { result, rerender } = renderSends();
        act(() => {
            result.current.sender?.send(outgoing({ body: "for room one" }));
        });

        // when
        rerender({ roomId: "room-2", onSending: vi.fn(), onDelivered: vi.fn() });
        const elsewhere = result.current.pending;
        rerender({ roomId: "room-1", onSending: vi.fn(), onDelivered: vi.fn() });

        // then
        expect(elsewhere).toEqual([]);
        expect(result.current.pending.map(entry => entry.body)).toEqual(["for room one"]);
    });

    it("hands back the same empty list across renders while nothing is pending", () => {
        // given
        const { result, rerender } = renderSends();
        const before = result.current.pending;

        // when
        rerender({ roomId: "room-1", onSending: vi.fn(), onDelivered: vi.fn() });

        // then
        expect(result.current.pending).toBe(before);
    });

    it("offers no sender while no room is open", () => {
        // when
        const { result } = renderSends({ roomId: undefined });

        // then
        expect(result.current.sender).toBeNull();
        expect(result.current.draftSender).toBeNull();
    });
});

describe("useOptimisticSends in a draft to someone never messaged", () => {
    const RECIPIENT = "u-battler";

    const IN_DRAFT: Partial<OptimisticSendsOptions> = { roomId: undefined, draftRecipientId: RECIPIENT };

    function landed(roomId: string, nonce: string): FirstDelivery {
        return {
            room: makeChatRoom({ id: roomId }),
            message: makeChatMessage({ id: "m-first", room_id: roomId, client_nonce: nonce }),
        };
    }

    it("shows the first message at once and opens the conversation with its nonce", () => {
        // given
        deferFirstSend();
        const { result } = renderSends(IN_DRAFT);

        // when
        act(() => {
            result.current.draftSender?.send(outgoing({ body: "are you there" }));
        });

        // then
        const [pending] = result.current.pending;
        expect(pending).toMatchObject({
            thread: { kind: "draft", recipientId: RECIPIENT },
            body: "are you there",
            delivery: "sending",
        });
        expect(mocks.sendFirstDMMessage).toHaveBeenCalledExactlyOnceWith({
            recipientId: RECIPIENT,
            body: "are you there",
            files: [],
            spoilers: [],
            client_nonce: pending.nonce,
        });
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
        expect(result.current.sender).toBeNull();
        expect(result.current.draftSender?.recipientId).toBe(RECIPIENT);
    });

    it("holds later messages while the conversation opens, then sends them through the room in order", async () => {
        // given three messages sent before the conversation exists
        const first = deferFirstSend();
        deferSend();
        deferSend();
        const onAdopted = vi.fn();
        const { result, rerender } = renderSends({ ...IN_DRAFT, onAdopted });
        for (const body of ["first", "second", "third"]) {
            act(() => {
                result.current.draftSender?.send(outgoing({ body }));
            });
        }
        const [firstNonce, secondNonce, thirdNonce] = result.current.pending.map(entry => entry.nonce);
        expect(mocks.sendFirstDMMessage).toHaveBeenCalledOnce();
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
        const delivery = landed("room-9", firstNonce);

        // when the conversation is opened
        await act(async () => {
            first.resolve(delivery);
        });

        // then the room is adopted once and the held messages follow through it in order
        expect(onAdopted).toHaveBeenCalledExactlyOnceWith(delivery.message, delivery.room);
        expect(mocks.sendFirstDMMessage).toHaveBeenCalledOnce();
        const roomSends = mocks.sendChatMessage.mock.calls.map(([input]) => [
            input.roomId,
            input.body,
            input.client_nonce,
        ]);
        expect(roomSends).toEqual([
            ["room-9", "second", secondNonce],
            ["room-9", "third", thirdNonce],
        ]);
        rerender({ roomId: "room-9", onSending: vi.fn(), onDelivered: vi.fn(), onAdopted });
        expect(result.current.pending.map(entry => [entry.nonce, entry.thread])).toEqual([
            [secondNonce, { kind: "room", roomId: "room-9" }],
            [thirdNonce, { kind: "room", roomId: "room-9" }],
        ]);
        expect(result.current.sender).toMatchObject({ roomId: "room-9", adoptedFrom: RECIPIENT });
    });

    it("fails the held messages with the first when the conversation cannot be opened, keeping their content", async () => {
        // given
        const first = deferFirstSend();
        const onAdopted = vi.fn();
        const { result } = renderSends({ ...IN_DRAFT, onAdopted });
        act(() => {
            result.current.draftSender?.send(outgoing({ body: "first" }));
        });
        act(() => {
            result.current.draftSender?.send(outgoing({ body: "second" }));
        });

        // when
        await act(async () => {
            first.reject(new Error("recipient has DMs disabled"));
        });

        // then
        expect(result.current.pending.map(entry => [entry.body, entry.delivery, entry.error])).toEqual([
            ["first", "failed", "recipient has DMs disabled"],
            ["second", "failed", "recipient has DMs disabled"],
        ]);
        expect(mocks.sendChatMessage).not.toHaveBeenCalled();
        expect(onAdopted).not.toHaveBeenCalled();
    });

    it("retries a failed first message under the same nonce so a retry never opens a second conversation", async () => {
        // given
        const first = deferFirstSend();
        const onAdopted = vi.fn();
        const { result } = renderSends({ ...IN_DRAFT, onAdopted });
        act(() => {
            result.current.draftSender?.send(outgoing({ body: "first" }));
        });
        const nonce = result.current.pending[0].nonce;
        await act(async () => {
            first.reject(new Error("network is down"));
        });
        const second = deferFirstSend();

        // when
        act(() => {
            result.current.retry(nonce);
        });

        // then
        expect(result.current.pending[0]).toMatchObject({ nonce, delivery: "sending", error: null });
        expect(mocks.sendFirstDMMessage).toHaveBeenLastCalledWith(expect.objectContaining({ client_nonce: nonce }));
        await act(async () => {
            second.resolve(landed("room-9", nonce));
        });
        expect(onAdopted).toHaveBeenCalledOnce();
        expect(result.current.pending).toEqual([]);
    });

    it("holds a retried message behind a first message that is still opening the conversation", async () => {
        // given both messages failed together
        const first = deferFirstSend();
        const { result } = renderSends(IN_DRAFT);
        act(() => {
            result.current.draftSender?.send(outgoing({ body: "first" }));
        });
        act(() => {
            result.current.draftSender?.send(outgoing({ body: "second" }));
        });
        const [firstNonce, secondNonce] = result.current.pending.map(entry => entry.nonce);
        await act(async () => {
            first.reject(new Error("network is down"));
        });
        const reopened = deferFirstSend();
        deferSend();

        // when both are retried while the first is opening the conversation again
        act(() => {
            result.current.retry(firstNonce);
        });
        act(() => {
            result.current.retry(secondNonce);
        });

        // then only the first opens the conversation and the second follows through the room
        expect(mocks.sendFirstDMMessage).toHaveBeenCalledTimes(2);
        await act(async () => {
            reopened.resolve(landed("room-9", firstNonce));
        });
        expect(mocks.sendFirstDMMessage).toHaveBeenCalledTimes(2);
        expect(mocks.sendChatMessage).toHaveBeenCalledExactlyOnceWith(
            expect.objectContaining({ roomId: "room-9", body: "second", client_nonce: secondNonce }),
        );
    });
});
