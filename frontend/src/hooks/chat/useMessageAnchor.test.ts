import { act, fireEvent, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeChatMessage } from "../../test-utils/fixtures";
import { providerWrapper } from "../../test-utils/render";
import type { ChatMessage } from "../../types/api";
import { useMessageAnchor } from "./useMessageAnchor";

const SETTLE_MS = 300;
const ANCHOR_HOLD_MS = 3000;

const observers: FakeIntersectionObserver[] = [];
const mounted: HTMLElement[] = [];

class FakeIntersectionObserver {
    readonly observed: Element[] = [];
    disconnected = false;
    private readonly callback: IntersectionObserverCallback;

    constructor(callback: IntersectionObserverCallback) {
        this.callback = callback;
        observers.push(this);
    }

    observe(node: Element) {
        this.observed.push(node);
    }

    unobserve() {}

    disconnect() {
        this.disconnected = true;
    }

    takeRecords(): IntersectionObserverEntry[] {
        return [];
    }

    report(intersectionRatio: number) {
        if (this.disconnected) {
            return;
        }

        const entries: IntersectionObserverEntry[] = [];
        for (const target of this.observed) {
            entries.push({
                target,
                intersectionRatio,
                isIntersecting: intersectionRatio > 0,
            } as IntersectionObserverEntry);
        }

        this.callback(entries, this as unknown as IntersectionObserver);
    }
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        sender: { id: "u2", username: "battler", display_name: "Battler" },
        body: "without love it cannot be seen",
        created_at: "2026-08-02T10:00:00Z",
        ...overrides,
    });
}

interface AnchorOptions {
    messages?: ChatMessage[];
    found?: boolean;
    route?: string;
}

function renderAnchor(options: AnchorOptions = {}) {
    const loadUntilMessage = vi.fn().mockResolvedValue(options.found ?? false);
    const onError = vi.fn();
    const rendered = renderHook(
        () => useMessageAnchor({ messages: options.messages ?? [], loadUntilMessage, onError }),
        { wrapper: providerWrapper({ route: options.route ?? "/rooms/room-1" }) },
    );

    return { ...rendered, loadUntilMessage, onError };
}

function bubble(messageId: string) {
    const container = document.createElement("div");
    const element = document.createElement("div");
    element.id = `chat-msg-${messageId}`;
    const scrollIntoView = vi.fn();
    element.scrollIntoView = scrollIntoView;
    container.appendChild(element);

    return { container, element, scrollIntoView };
}

function mount(container: HTMLElement) {
    document.body.appendChild(container);
    mounted.push(container);
}

function anchorObserver(): FakeIntersectionObserver {
    const observer = observers[observers.length - 1];
    if (!observer) {
        throw new Error("the anchor never started watching the linked message");
    }

    return observer;
}

function settle(ms = SETTLE_MS) {
    act(() => {
        vi.advanceTimersByTime(ms);
    });
}

beforeEach(() => {
    observers.length = 0;
    vi.stubGlobal("IntersectionObserver", FakeIntersectionObserver);
});

afterEach(() => {
    for (const node of mounted) {
        node.remove();
    }
    mounted.length = 0;
});

describe("useMessageAnchor", () => {
    it("highlights a message that is already on screen", async () => {
        // given
        const { result } = renderAnchor({ messages: [makeMessage({ id: "m1" })] });
        const target = bubble("m1");
        mount(target.container);

        // when
        await act(async () => {
            await result.current.handleJumpToMessage("m1");
        });

        // then
        expect(result.current.highlightedMsgId).toBe("m1");
    });

    it("says so when the message cannot be found anywhere in the history", async () => {
        // given
        const { result, onError } = renderAnchor();

        // when
        await act(async () => {
            await result.current.handleJumpToMessage("ghost");
        });

        // then
        expect(onError).toHaveBeenCalledWith("Couldn't locate that message.");
    });
});

describe("useMessageAnchor arriving from a message link", () => {
    const linked = "/rooms/room-1?at=2026-08-02T10%3A00%3A00Z#msg-m1";

    it("scrolls to and highlights the linked message once the thread settles", () => {
        // given
        vi.useFakeTimers();
        const target = bubble("m1");
        mount(target.container);
        const { result } = renderAnchor({ messages: [makeMessage({ id: "m1" })], route: linked });

        // when
        settle();

        // then
        expect(target.scrollIntoView).toHaveBeenCalledExactlyOnceWith({ behavior: "auto", block: "center" });
        expect(result.current.highlightedMsgId).toBe("m1");
        expect(anchorObserver().observed).toEqual([target.element]);
    });

    interface ShiftCase {
        name: string;
        ratio: number;
        wantScrolls: number;
    }

    const shiftCases: ShiftCase[] = [
        { name: "brings it back when late layout pushes it out of view", ratio: 0, wantScrolls: 2 },
        { name: "brings it back when late layout leaves only a sliver of it showing", ratio: 0.2, wantScrolls: 2 },
        { name: "leaves it alone while most of it is still showing", ratio: 0.6, wantScrolls: 1 },
    ];

    for (const shiftCase of shiftCases) {
        it(shiftCase.name, () => {
            // given
            vi.useFakeTimers();
            const target = bubble("m1");
            mount(target.container);
            renderAnchor({ messages: [makeMessage({ id: "m1" })], route: linked });
            settle();

            // when
            anchorObserver().report(shiftCase.ratio);

            // then
            expect(target.scrollIntoView).toHaveBeenCalledTimes(shiftCase.wantScrolls);
        });
    }

    interface TakeoverCase {
        name: string;
        input: (container: HTMLElement) => void;
    }

    const takeoverCases: TakeoverCase[] = [
        { name: "a mouse wheel", input: container => fireEvent.wheel(container) },
        { name: "a touch drag", input: container => fireEvent.touchMove(container) },
        { name: "a key press", input: container => fireEvent.keyDown(container, { key: "PageDown" }) },
        { name: "a pointer press", input: container => fireEvent.pointerDown(container) },
    ];

    for (const takeoverCase of takeoverCases) {
        it(`stops pulling the reader back once they scroll with ${takeoverCase.name}`, () => {
            // given
            vi.useFakeTimers();
            const target = bubble("m1");
            mount(target.container);
            renderAnchor({ messages: [makeMessage({ id: "m1" })], route: linked });
            settle();

            // when
            takeoverCase.input(target.container);
            anchorObserver().report(0);

            // then
            expect(anchorObserver().disconnected).toBe(true);
            expect(target.scrollIntoView).toHaveBeenCalledOnce();
        });
    }

    it("stops pulling the reader back once they jump to another message", async () => {
        // given
        vi.useFakeTimers();
        const target = bubble("m1");
        const other = bubble("m2");
        mount(target.container);
        mount(other.container);
        const { result } = renderAnchor({
            messages: [makeMessage({ id: "m1" }), makeMessage({ id: "m2" })],
            route: linked,
        });
        settle();
        const landing = anchorObserver();

        // when
        await act(async () => {
            await result.current.handleJumpToMessage("m2");
        });
        landing.report(0);

        // then
        expect(landing.disconnected).toBe(true);
        expect(target.scrollIntoView).toHaveBeenCalledOnce();
        expect(other.scrollIntoView).toHaveBeenCalledOnce();
    });

    it("lets the layout go after a few seconds", () => {
        // given
        vi.useFakeTimers();
        const target = bubble("m1");
        mount(target.container);
        renderAnchor({ messages: [makeMessage({ id: "m1" })], route: linked });
        settle();

        // when
        settle(ANCHOR_HOLD_MS);
        anchorObserver().report(0);

        // then
        expect(anchorObserver().disconnected).toBe(true);
        expect(target.scrollIntoView).toHaveBeenCalledOnce();
    });

    it("waits for a linked message that is loaded but not drawn yet instead of giving up on it", async () => {
        // given
        vi.useFakeTimers();
        const target = bubble("m1");
        const { result } = renderAnchor({ messages: [makeMessage({ id: "m1" })], route: linked });
        settle();
        expect(result.current.highlightedMsgId).toBeNull();

        // when
        await act(async () => {
            mount(target.container);
        });

        // then
        expect(target.scrollIntoView).toHaveBeenCalledOnce();
        expect(result.current.highlightedMsgId).toBe("m1");
    });

    it("fetches back to the linked message's time and says so when it is nowhere in the history", async () => {
        // given
        const { loadUntilMessage, onError } = renderAnchor({
            messages: [makeMessage({ id: "m0" })],
            route: linked,
            found: false,
        });

        // when
        await act(async () => {
            await Promise.resolve();
        });

        // then
        expect(loadUntilMessage).toHaveBeenCalledExactlyOnceWith("m1", "2026-08-02T10:00:00Z");
        expect(onError).toHaveBeenCalledExactlyOnceWith("Couldn't locate that message.");
    });

    it("lets go of the linked message when the thread goes away", () => {
        // given
        vi.useFakeTimers();
        const target = bubble("m1");
        mount(target.container);
        const { unmount } = renderAnchor({ messages: [makeMessage({ id: "m1" })], route: linked });
        settle();

        // when
        unmount();

        // then
        expect(anchorObserver().disconnected).toBe(true);
    });
});
