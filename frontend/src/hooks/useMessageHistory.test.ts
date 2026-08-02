import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ChatMessage } from "../types/api";
import { useMessageHistory } from "./useMessageHistory";

const mocks = vi.hoisted(() => ({
    fetchRoomMessages: vi.fn(),
    fetchRoomMessagesBefore: vi.fn(),
}));

vi.mock("../api/queries/chat", () => ({
    fetchRoomMessages: mocks.fetchRoomMessages,
    fetchRoomMessagesBefore: mocks.fetchRoomMessagesBefore,
}));

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

interface HistoryProps {
    rid: string | undefined;
    max?: number;
}

function renderHistory(props: HistoryProps) {
    return renderHook(({ rid, max }: HistoryProps) => useMessageHistory(rid, max), { initialProps: props });
}

function makeContainer(opts: { scrollHeight?: number; clientHeight?: number; scrollTop?: number } = {}) {
    const el = document.createElement("div");
    Object.defineProperty(el, "scrollHeight", { configurable: true, value: opts.scrollHeight ?? 1000 });
    Object.defineProperty(el, "clientHeight", { configurable: true, value: opts.clientHeight ?? 400 });
    Object.defineProperty(el, "scrollTop", { configurable: true, writable: true, value: opts.scrollTop ?? 0 });
    Object.defineProperty(el, "scrollTo", { configurable: true, writable: true, value: vi.fn() });
    return el;
}

beforeEach(() => {
    mocks.fetchRoomMessages.mockResolvedValue({ messages: [], total: 0 });
    mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [], total: 0 });
});

describe("useMessageHistory first page", () => {
    it("loads the first page of messages when a room is opened", async () => {
        // given
        const first = makeMessage({ id: "m1" });
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [first], total: 1 });

        // when
        const { result } = renderHistory({ rid: "room-1" });

        // then
        await waitFor(() => {
            expect(result.current.messages).toEqual([first]);
        });
        expect(mocks.fetchRoomMessages).toHaveBeenCalledWith("room-1", 50);
    });

    it("reports that older history exists when the room holds more than one page", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage()], total: 120 });

        // when
        const { result } = renderHistory({ rid: "room-1" });

        // then
        await waitFor(() => {
            expect(result.current.hasMore).toBe(true);
        });
    });

    it("reports no older history when the first page covers the whole room", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage()], total: 1 });

        // when
        const { result } = renderHistory({ rid: "room-1" });

        // then
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        expect(result.current.hasMore).toBe(false);
    });

    it("stays empty and asks for nothing while there is no room", () => {
        // given
        const props: HistoryProps = { rid: undefined };

        // when
        const { result } = renderHistory(props);

        // then
        expect(result.current.messages).toEqual([]);
        expect(mocks.fetchRoomMessages).not.toHaveBeenCalled();
    });

    it("empties the list when the first page cannot be loaded", async () => {
        // given
        mocks.fetchRoomMessages.mockRejectedValue(new Error("network down"));

        // when
        const { result } = renderHistory({ rid: "room-1" });

        // then
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });
        expect(result.current.messages).toEqual([]);
        expect(result.current.hasMore).toBe(false);
    });

    it("shows nothing from the previous room the moment the viewer switches rooms", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m1" })], total: 1 });
        const { result, rerender } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });

        // when
        mocks.fetchRoomMessages.mockReturnValue(new Promise(() => {}));
        rerender({ rid: "room-2" });

        // then
        expect(result.current.messages).toEqual([]);
        expect(result.current.hasMore).toBe(false);
    });

    it("ignores a page that arrives after the viewer moved to another room", async () => {
        // given
        let releaseFirst: (value: { messages: ChatMessage[]; total: number }) => void = () => {};
        const stale = makeMessage({ id: "stale", room_id: "room-1" });
        const fresh = makeMessage({ id: "fresh", room_id: "room-2" });
        mocks.fetchRoomMessages.mockImplementation((rid: string) => {
            if (rid === "room-1") {
                return new Promise<{ messages: ChatMessage[]; total: number }>(resolve => {
                    releaseFirst = resolve;
                });
            }
            return Promise.resolve({ messages: [fresh], total: 1 });
        });
        const { result, rerender } = renderHistory({ rid: "room-1" });

        // when
        rerender({ rid: "room-2" });
        await waitFor(() => {
            expect(result.current.messages).toEqual([fresh]);
        });
        await act(async () => {
            releaseFirst({ messages: [stale], total: 1 });
        });

        // then
        expect(result.current.messages).toEqual([fresh]);
    });
});

describe("useMessageHistory setMessages", () => {
    it("replaces the whole list when handed a plain array", async () => {
        // given
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });
        const replacement = [makeMessage({ id: "m9" })];

        // when
        act(() => {
            result.current.setMessages(replacement);
        });

        // then
        expect(result.current.messages).toEqual(replacement);
    });

    it("hands the messages already loaded to an updater function", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m1" })], total: 1 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });

        // when
        act(() => {
            result.current.setMessages(prev => [...prev, makeMessage({ id: "m2" })]);
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1", "m2"]);
    });

    it("trims the oldest messages once the list passes the cap", async () => {
        // given
        const { result } = renderHistory({ rid: "room-1", max: 2 });
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });

        // when
        act(() => {
            result.current.setMessages([
                makeMessage({ id: "m1" }),
                makeMessage({ id: "m2" }),
                makeMessage({ id: "m3" }),
            ]);
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m2", "m3"]);
        expect(result.current.hasMore).toBe(true);
    });

    it("keeps every message when no cap was given", async () => {
        // given
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });

        // when
        act(() => {
            result.current.setMessages([
                makeMessage({ id: "m1" }),
                makeMessage({ id: "m2" }),
                makeMessage({ id: "m3" }),
            ]);
        });

        // then
        expect(result.current.messages).toHaveLength(3);
        expect(result.current.hasMore).toBe(false);
    });
});

describe("useMessageHistory addMessage", () => {
    it("appends a message the viewer has not seen yet", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m1" })], total: 1 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });

        // when
        act(() => {
            result.current.addMessage(makeMessage({ id: "m2", body: "the red truth" }));
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1", "m2"]);
    });

    it("replaces the copy it already holds rather than showing the message twice", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m1", body: "sending" })], total: 1 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });

        // when
        act(() => {
            result.current.addMessage(makeMessage({ id: "m1", body: "delivered" }));
        });

        // then
        expect(result.current.messages).toHaveLength(1);
        expect(result.current.messages[0].body).toBe("delivered");
    });
});

describe("useMessageHistory older pages", () => {
    async function renderScrolledToTop(messages: ChatMessage[], total: number) {
        mocks.fetchRoomMessages.mockResolvedValue({ messages, total });
        const rendered = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(rendered.result.current.messages).toHaveLength(messages.length);
        });

        const container = makeContainer({ scrollTop: 0, scrollHeight: 1000, clientHeight: 400 });
        act(() => {
            rendered.result.current.containerRef(container);
        });

        return { ...rendered, container };
    }

    it("asks for the page before the oldest message it already holds", async () => {
        // given
        const oldest = makeMessage({ id: "m5", created_at: "2026-01-05T00:00:00Z" });
        const { result } = await renderScrolledToTop([oldest], 90);

        // when
        await act(async () => {
            result.current.handleScroll();
        });

        // then
        expect(mocks.fetchRoomMessagesBefore).toHaveBeenCalledWith("room-1", "2026-01-05T00:00:00Z|m5", 50);
    });

    it("prepends the older page ahead of the messages already shown", async () => {
        // given
        const { result } = await renderScrolledToTop([makeMessage({ id: "m5" })], 90);
        mocks.fetchRoomMessagesBefore.mockResolvedValue({
            messages: [makeMessage({ id: "m3" }), makeMessage({ id: "m4" })],
            total: 90,
        });

        // when
        await act(async () => {
            result.current.handleScroll();
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m3", "m4", "m5"]);
    });

    it("drops older messages the viewer already holds when merging the page", async () => {
        // given
        const { result } = await renderScrolledToTop([makeMessage({ id: "m4" }), makeMessage({ id: "m5" })], 90);
        mocks.fetchRoomMessagesBefore.mockResolvedValue({
            messages: [makeMessage({ id: "m3" }), makeMessage({ id: "m4" })],
            total: 90,
        });

        // when
        await act(async () => {
            result.current.handleScroll();
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m3", "m4", "m5"]);
    });

    it("stops offering older history once a page comes back empty", async () => {
        // given
        const { result } = await renderScrolledToTop([makeMessage({ id: "m5" })], 90);
        mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [], total: 90 });

        // when
        await act(async () => {
            result.current.handleScroll();
        });

        // then
        expect(result.current.hasMore).toBe(false);
    });

    it("does not load an older page when there is no more history", async () => {
        // given
        const { result } = await renderScrolledToTop([makeMessage({ id: "m5" })], 1);

        // when
        await act(async () => {
            result.current.handleScroll();
        });

        // then
        expect(mocks.fetchRoomMessagesBefore).not.toHaveBeenCalled();
    });

    it("does not load an older page while the viewer is nowhere near the top", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m5" })], total: 90 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        const container = makeContainer({ scrollTop: 600, scrollHeight: 1000, clientHeight: 400 });
        act(() => {
            result.current.containerRef(container);
        });

        // when
        await act(async () => {
            result.current.handleScroll();
        });

        // then
        expect(mocks.fetchRoomMessagesBefore).not.toHaveBeenCalled();
    });
});

describe("useMessageHistory loadUntilMessage", () => {
    it("gives up immediately when there is no room", async () => {
        // given
        const { result } = renderHistory({ rid: undefined });

        // when
        let found = true;
        await act(async () => {
            found = await result.current.loadUntilMessage("m1");
        });

        // then
        expect(found).toBe(false);
        expect(mocks.fetchRoomMessagesBefore).not.toHaveBeenCalled();
    });

    it("stops at once when the message is already loaded", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m1" })], total: 90 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });

        // when
        let found = false;
        await act(async () => {
            found = await result.current.loadUntilMessage("m1");
        });

        // then
        expect(found).toBe(true);
        expect(mocks.fetchRoomMessagesBefore).not.toHaveBeenCalled();
    });

    it("walks back from the oldest message it already holds", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({
            messages: [makeMessage({ id: "m5", created_at: "2026-01-05T00:00:00Z" })],
            total: 900,
        });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        mocks.fetchRoomMessagesBefore.mockResolvedValueOnce({
            messages: [makeMessage({ id: "m4", created_at: "2026-01-04T00:00:00Z" })],
            total: 900,
        });
        mocks.fetchRoomMessagesBefore.mockResolvedValueOnce({
            messages: [makeMessage({ id: "m3", created_at: "2026-01-03T00:00:00Z" })],
            total: 900,
        });

        // when
        await act(async () => {
            await result.current.loadUntilMessage("ghost", undefined, 2);
        });

        // then
        expect(mocks.fetchRoomMessagesBefore).toHaveBeenNthCalledWith(1, "room-1", "2026-01-05T00:00:00Z|m5", 50);
        expect(mocks.fetchRoomMessagesBefore).toHaveBeenNthCalledWith(2, "room-1", "2026-01-04T00:00:00Z|m4", 50);
    });

    it("reports the message as found as soon as a page brings it in", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m5" })], total: 900 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [makeMessage({ id: "m4" })], total: 900 });

        // when
        let found = false;
        await act(async () => {
            found = await result.current.loadUntilMessage("m4");
        });

        // then
        expect(found).toBe(true);
        expect(mocks.fetchRoomMessagesBefore).toHaveBeenCalledTimes(1);
    });

    it("merges every older page it walks through while hunting for the message", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m5" })], total: 90 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        mocks.fetchRoomMessagesBefore.mockResolvedValueOnce({ messages: [makeMessage({ id: "m4" })], total: 90 });
        mocks.fetchRoomMessagesBefore.mockResolvedValueOnce({ messages: [makeMessage({ id: "m3" })], total: 90 });

        // when
        await act(async () => {
            await result.current.loadUntilMessage("m1", undefined, 2);
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m3", "m4", "m5"]);
    });

    it("gives up and stops offering history when a page comes back empty", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m5" })], total: 90 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [], total: 90 });

        // when
        let found = true;
        await act(async () => {
            found = await result.current.loadUntilMessage("ghost");
        });

        // then
        expect(found).toBe(false);
        expect(result.current.hasMore).toBe(false);
    });

    it("stops after the page limit it was given", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m5" })], total: 900 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        mocks.fetchRoomMessagesBefore.mockResolvedValue({ messages: [makeMessage({ id: "m4" })], total: 900 });

        // when
        let found = true;
        await act(async () => {
            found = await result.current.loadUntilMessage("ghost", undefined, 1);
        });

        // then
        expect(found).toBe(false);
        expect(mocks.fetchRoomMessagesBefore).toHaveBeenCalledTimes(1);
    });

    it("jumps straight to the timestamp it was given and keeps the merged list in order", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({
            messages: [makeMessage({ id: "m5", created_at: "2026-01-05T00:00:00Z" })],
            total: 90,
        });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        mocks.fetchRoomMessagesBefore.mockResolvedValue({
            messages: [makeMessage({ id: "m1", created_at: "2026-01-01T00:00:00Z" })],
            total: 90,
        });

        // when
        await act(async () => {
            await result.current.loadUntilMessage("m1", "2026-01-02T00:00:00Z");
        });

        // then
        expect(mocks.fetchRoomMessagesBefore).toHaveBeenCalledWith(
            "room-1",
            "2026-01-02T00:00:00Z|ffffffff-ffff-ffff-ffff-ffffffffffff",
            50,
        );
        expect(mocks.fetchRoomMessagesBefore).toHaveBeenCalledTimes(1);
        expect(result.current.messages.map(m => m.id)).toEqual(["m1", "m5"]);
    });
});

describe("useMessageHistory resync", () => {
    it("adds only the messages that arrived while the socket was down", async () => {
        // given
        const known = makeMessage({ id: "m1" });
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [known], total: 1 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        const missed = makeMessage({ id: "m2" });
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [known, missed], total: 2 });

        // when
        await act(async () => {
            await result.current.resync();
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1", "m2"]);
    });

    it("leaves the list untouched when the refetch brings nothing new", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m1" })], total: 1 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        const before = result.current.messages;

        // when
        await act(async () => {
            await result.current.resync();
        });

        // then
        expect(result.current.messages).toBe(before);
    });

    it("does nothing when there is no room to resync", async () => {
        // given
        const { result } = renderHistory({ rid: undefined });

        // when
        await act(async () => {
            await result.current.resync();
        });

        // then
        expect(mocks.fetchRoomMessages).not.toHaveBeenCalled();
    });

    it("swallows a failed resync and keeps what it already has", async () => {
        // given
        mocks.fetchRoomMessages.mockResolvedValue({ messages: [makeMessage({ id: "m1" })], total: 1 });
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(result.current.messages).toHaveLength(1);
        });
        mocks.fetchRoomMessages.mockRejectedValue(new Error("socket still down"));

        // when
        await act(async () => {
            await result.current.resync();
        });

        // then
        expect(result.current.messages.map(m => m.id)).toEqual(["m1"]);
    });
});

describe("useMessageHistory scrolling", () => {
    it("scrolls the container to the bottom when forced", async () => {
        // given
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });
        const container = makeContainer({ scrollTop: 900, scrollHeight: 1000, clientHeight: 400 });
        act(() => {
            result.current.containerRef(container);
        });

        // when
        act(() => {
            result.current.scrollToBottom({ force: true });
        });

        // then
        await waitFor(() => {
            expect(container.scrollTo).toHaveBeenCalledWith({ top: 1000, behavior: "smooth" });
        });
    });

    it("stays put when the viewer has scrolled up and the scroll was not forced", async () => {
        // given
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });
        const container = makeContainer({ scrollTop: 0, scrollHeight: 1000, clientHeight: 400 });
        act(() => {
            result.current.containerRef(container);
            result.current.handleScroll();
        });

        // when
        act(() => {
            result.current.scrollToBottom();
        });

        // then
        expect(container.scrollTo).not.toHaveBeenCalled();
    });

    it("jumps to the bottom instantly without animating", async () => {
        // given
        const { result } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });
        const container = makeContainer({ scrollTop: 0, scrollHeight: 1000, clientHeight: 400 });
        act(() => {
            result.current.containerRef(container);
        });

        // when
        act(() => {
            result.current.scrollToBottomInstant({ force: true });
        });

        // then
        expect(container.scrollTop).toBe(1000);
        expect(container.scrollTo).not.toHaveBeenCalled();
    });
});

describe("useMessageHistory cleanup", () => {
    it("disconnects the resize observer when the room view goes away", async () => {
        // given
        const disconnectSpy = vi.fn();
        class FakeResizeObserver {
            observe = vi.fn();
            unobserve = vi.fn();
            disconnect = disconnectSpy;
        }
        vi.stubGlobal("ResizeObserver", FakeResizeObserver);
        const { result, unmount } = renderHistory({ rid: "room-1" });
        await waitFor(() => {
            expect(mocks.fetchRoomMessages).toHaveBeenCalled();
        });
        act(() => {
            result.current.containerRef(makeContainer());
        });

        // when
        unmount();

        // then
        expect(disconnectSpy).toHaveBeenCalled();
    });
});
