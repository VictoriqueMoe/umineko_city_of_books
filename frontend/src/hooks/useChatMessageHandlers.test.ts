import { act, renderHook, waitFor } from "@testing-library/react";
import type { Dispatch, SetStateAction } from "react";
import { describe, expect, it, type Mock, vi } from "vitest";
import type { ChatMessage, UserProfile } from "../types/api";
import { makeUser } from "../test-utils/fixtures";
import { useChatMessageHandlers } from "./useChatMessageHandlers";

const mocks = vi.hoisted(() => ({
    deleteMessage: vi.fn(),
    editMessage: vi.fn(),
}));

vi.mock("../api/mutations/chat", () => ({
    useDeleteChatMessage: () => ({ mutateAsync: mocks.deleteMessage }),
    useEditChatMessage: () => ({ mutateAsync: mocks.editMessage }),
}));

type MessagesDispatch = Dispatch<SetStateAction<ChatMessage[]>>;

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: { id: "u1", username: "beatrice", display_name: "Beatrice" },
        body: "the golden truth",
        is_system: false,
        created_at: "2026-01-01T00:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}

function applyFirstUpdate(setMessages: Mock<MessagesDispatch>, previous: ChatMessage[]): ChatMessage[] {
    const updater = setMessages.mock.calls[0][0] as (current: ChatMessage[]) => ChatMessage[];

    return updater(previous);
}

interface HandlerOptions {
    user?: UserProfile | null;
    messages: ChatMessage[];
    setMessages: MessagesDispatch;
    setEditingMessageId?: (id: string | null) => void;
    onError?: (message: string) => void;
    editLastBlocked?: boolean;
}

function renderHandlers(options: HandlerOptions) {
    return renderHook(() =>
        useChatMessageHandlers({
            user: options.user === undefined ? makeUser({ id: "u1" }) : options.user,
            messages: options.messages,
            setMessages: options.setMessages,
            setEditingMessageId: options.setEditingMessageId ?? (() => {}),
            onError: options.onError,
            editLastBlocked: options.editLastBlocked,
        }),
    );
}

describe("handleDeleteMessage", () => {
    it("drops the deleted message and leaves the rest of the room alone", async () => {
        // given
        const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })];
        const setMessages = vi.fn<MessagesDispatch>();
        const { result } = renderHandlers({ messages, setMessages });

        // when
        await act(async () => {
            await result.current.handleDeleteMessage(messages[0]);
        });

        // then
        expect(mocks.deleteMessage).toHaveBeenCalledWith("m1");
        expect(applyFirstUpdate(setMessages, messages).map(m => m.id)).toEqual(["m2"]);
    });

    it("reports the failure and keeps the message when the request is rejected", async () => {
        // given
        mocks.deleteMessage.mockRejectedValue(new Error("you may not delete that"));
        const onError = vi.fn();
        const setMessages = vi.fn<MessagesDispatch>();
        const message = makeMessage();
        const { result } = renderHandlers({ messages: [message], setMessages, onError });

        // when
        await act(async () => {
            await result.current.handleDeleteMessage(message);
        });

        // then
        expect(onError).toHaveBeenCalledWith("you may not delete that");
        expect(setMessages).not.toHaveBeenCalled();
    });

    it("falls back to a generic message when the failure is not an error", async () => {
        // given
        mocks.deleteMessage.mockRejectedValue("kaboom");
        const onError = vi.fn();
        const message = makeMessage();
        const { result } = renderHandlers({ messages: [message], setMessages: vi.fn<MessagesDispatch>(), onError });

        // when
        await act(async () => {
            await result.current.handleDeleteMessage(message);
        });

        // then
        expect(onError).toHaveBeenCalledWith("Failed to delete message");
    });

    it("swallows the failure when no error handler was supplied", async () => {
        // given
        mocks.deleteMessage.mockRejectedValue(new Error("nope"));
        const setMessages = vi.fn<MessagesDispatch>();
        const message = makeMessage();
        const { result } = renderHandlers({ messages: [message], setMessages });

        // when
        await act(async () => {
            await result.current.handleDeleteMessage(message);
        });

        // then
        expect(setMessages).not.toHaveBeenCalled();
    });
});

describe("handleEditMessage", () => {
    it("sends the new body and applies the server copy to the matching message only", async () => {
        // given
        const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2", body: "untouched" })];
        mocks.editMessage.mockResolvedValue(
            makeMessage({ id: "m1", body: "the red truth", edited_at: "2026-01-02T00:00:00Z" }),
        );
        const setMessages = vi.fn<MessagesDispatch>();
        const { result } = renderHandlers({ messages, setMessages });

        // when
        await act(async () => {
            await result.current.handleEditMessage(messages[0], "the red truth");
        });

        // then
        expect(mocks.editMessage).toHaveBeenCalledWith({ messageId: "m1", body: "the red truth" });
        expect(applyFirstUpdate(setMessages, messages)).toEqual([
            expect.objectContaining({ id: "m1", body: "the red truth", edited_at: "2026-01-02T00:00:00Z" }),
            expect.objectContaining({ id: "m2", body: "untouched" }),
        ]);
    });

    it("reports the failure and rethrows so the editor can stay open", async () => {
        // given
        mocks.editMessage.mockRejectedValue(new Error("message too old"));
        const onError = vi.fn();
        const setMessages = vi.fn<MessagesDispatch>();
        const message = makeMessage();
        const { result } = renderHandlers({ messages: [message], setMessages, onError });

        // when
        const attempt = act(async () => {
            await result.current.handleEditMessage(message, "too late");
        });

        // then
        await expect(attempt).rejects.toThrow("message too old");
        expect(onError).toHaveBeenCalledWith("message too old");
        expect(setMessages).not.toHaveBeenCalled();
    });

    it("falls back to a generic message when the failure is not an error", async () => {
        // given
        mocks.editMessage.mockRejectedValue("kaboom");
        const onError = vi.fn();
        const message = makeMessage();
        const { result } = renderHandlers({ messages: [message], setMessages: vi.fn<MessagesDispatch>(), onError });

        // when
        const attempt = act(async () => {
            await result.current.handleEditMessage(message, "whatever");
        });

        // then
        await expect(attempt).rejects.toBe("kaboom");
        expect(onError).toHaveBeenCalledWith("Failed to edit message");
    });
});

describe("handleEditLast", () => {
    it("opens the newest message the viewer sent", () => {
        // given
        const messages = [
            makeMessage({ id: "m1" }),
            makeMessage({ id: "m2", sender: { id: "u2", username: "battler", display_name: "Battler" } }),
            makeMessage({ id: "m3" }),
        ];
        const setEditingMessageId = vi.fn();
        const { result } = renderHandlers({ messages, setMessages: vi.fn<MessagesDispatch>(), setEditingMessageId });

        // when
        act(() => {
            result.current.handleEditLast();
        });

        // then
        expect(setEditingMessageId).toHaveBeenCalledWith("m3");
    });

    it("skips system messages when looking for something to edit", () => {
        // given
        const messages = [makeMessage({ id: "m1" }), makeMessage({ id: "m2", is_system: true })];
        const setEditingMessageId = vi.fn();
        const { result } = renderHandlers({ messages, setMessages: vi.fn<MessagesDispatch>(), setEditingMessageId });

        // when
        act(() => {
            result.current.handleEditLast();
        });

        // then
        expect(setEditingMessageId).toHaveBeenCalledWith("m1");
    });

    it("does nothing when nobody is signed in", () => {
        // given
        const setEditingMessageId = vi.fn();
        const { result } = renderHandlers({
            user: null,
            messages: [makeMessage()],
            setMessages: vi.fn<MessagesDispatch>(),
            setEditingMessageId,
        });

        // when
        act(() => {
            result.current.handleEditLast();
        });

        // then
        expect(setEditingMessageId).not.toHaveBeenCalled();
    });

    it("does nothing while editing the last message is blocked", () => {
        // given
        const setEditingMessageId = vi.fn();
        const { result } = renderHandlers({
            messages: [makeMessage()],
            setMessages: vi.fn<MessagesDispatch>(),
            setEditingMessageId,
            editLastBlocked: true,
        });

        // when
        act(() => {
            result.current.handleEditLast();
        });

        // then
        expect(setEditingMessageId).not.toHaveBeenCalled();
    });

    it("does nothing when the viewer has said nothing in the room", () => {
        // given
        const messages = [
            makeMessage({ id: "m1", sender: { id: "u2", username: "battler", display_name: "Battler" } }),
        ];
        const setEditingMessageId = vi.fn();
        const { result } = renderHandlers({ messages, setMessages: vi.fn<MessagesDispatch>(), setEditingMessageId });

        // when
        act(() => {
            result.current.handleEditLast();
        });

        // then
        expect(setEditingMessageId).not.toHaveBeenCalled();
    });

    it("scrolls the message it opened into view", async () => {
        // given
        const scrollIntoView = vi.fn();
        const target = document.createElement("div");
        target.id = "chat-msg-m1";
        target.scrollIntoView = scrollIntoView;
        document.body.appendChild(target);
        const { result } = renderHandlers({
            messages: [makeMessage({ id: "m1" })],
            setMessages: vi.fn<MessagesDispatch>(),
        });

        // when
        act(() => {
            result.current.handleEditLast();
        });

        // then
        await waitFor(() => {
            expect(scrollIntoView).toHaveBeenCalledWith({ behavior: "smooth", block: "center" });
        });
        target.remove();
    });
});
