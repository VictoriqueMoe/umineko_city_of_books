import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import type { NotificationContextValue, WSMessageHandler } from "../../../context/notificationContextValue";
import { providerWrapper } from "../../../test-utils/render";
import { PONG_INPUT_INTERVAL_MS, PONG_INPUT_KEEPALIVE_MS, PONG_INPUT_TYPE } from "../types";
import { usePongInput } from "./usePongInput";

const COURT_HEIGHT = 800;
const PADDLE_HEIGHT = 130;

const pending = new Map<number, FrameRequestCallback>();
let handleSeq = 0;

function frame(now: number): void {
    const due = [...pending.values()];
    pending.clear();
    act(() => {
        for (const callback of due) {
            callback(now);
        }
    });
}

interface Harness {
    notification: Partial<NotificationContextValue>;
    sendWSMessage: Mock<(msg: object) => void>;
}

function makeHarness(): Harness {
    const addWSListener = vi.fn((_handler: WSMessageHandler) => () => {});
    const sendWSMessage: Mock<(msg: object) => void> = vi.fn();
    return { notification: { addWSListener, sendWSMessage, wsEpoch: 1 }, sendWSMessage };
}

function setup(harness: Harness, enabled = true, roomId: string | null = "room-1") {
    return renderHook(
        () =>
            usePongInput({
                roomId: roomId ?? undefined,
                enabled,
                courtHeight: COURT_HEIGHT,
                paddleHeight: PADDLE_HEIGHT,
                initialY: 400,
            }),
        { wrapper: providerWrapper({ notification: harness.notification }) },
    );
}

function sentPayloads(sendWSMessage: Mock<(msg: object) => void>): { y: number; seq: number }[] {
    return sendWSMessage.mock.calls.map(
        ([msg]) => (msg as { data: { payload: { y: number; seq: number } } }).data.payload,
    );
}

beforeEach(() => {
    pending.clear();
    handleSeq = 0;
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
        handleSeq += 1;
        pending.set(handleSeq, callback);
        return handleSeq;
    });
    vi.stubGlobal("cancelAnimationFrame", (handle: number) => {
        pending.delete(handle);
    });
});

afterEach(() => {
    vi.unstubAllGlobals();
});

describe("usePongInput", () => {
    it("sends the opening target on the first animation frame after seeding", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);

        // when
        frame(1000);

        // then
        expect(harness.sendWSMessage).toHaveBeenCalledTimes(1);
        expect(harness.sendWSMessage).toHaveBeenCalledWith({
            type: PONG_INPUT_TYPE,
            data: { room_id: "room-1", payload: { y: 400, seq: 1 } },
        });
    });

    it("sends nothing while the target is still the unseeded default", () => {
        // given
        const harness = makeHarness();
        setup(harness);

        // when
        frame(1000);
        frame(1000 + PONG_INPUT_KEEPALIVE_MS + 10);

        // then
        expect(harness.sendWSMessage).not.toHaveBeenCalled();
    });

    it("takes the seed from the first authoritative paddle rather than the stale opener", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);

        // when
        result.current.seedTargetY(620);
        result.current.seedTargetY(180);
        frame(1000);

        // then
        expect(result.current.targetRef.current).toBe(620);
        expect(sentPayloads(harness.sendWSMessage)).toEqual([{ y: 620, seq: 1 }]);
    });

    it("keeps a real move in front of a later seed", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);

        // when
        result.current.setTargetY(300);
        result.current.seedTargetY(620);
        frame(1000);

        // then
        expect(sentPayloads(harness.sendWSMessage)).toEqual([{ y: 300, seq: 1 }]);
    });

    it("coalesces a moving target down to the input interval", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);

        // when
        result.current.setTargetY(500);
        frame(1000 + PONG_INPUT_INTERVAL_MS - 10);
        frame(1000 + PONG_INPUT_INTERVAL_MS + 10);

        // then
        expect(sentPayloads(harness.sendWSMessage)).toEqual([
            { y: 400, seq: 1 },
            { y: 500, seq: 2 },
        ]);
    });

    it("stays quiet while the target moves less than the epsilon", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);

        // when
        result.current.setTargetY(402);
        frame(1000 + PONG_INPUT_INTERVAL_MS + 10);

        // then
        expect(harness.sendWSMessage).toHaveBeenCalledTimes(1);
    });

    it("resends a parked target once the keepalive has run out", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);
        result.current.setTargetY(402);

        // when
        frame(1000 + PONG_INPUT_KEEPALIVE_MS + 10);

        // then
        expect(sentPayloads(harness.sendWSMessage)).toEqual([
            { y: 400, seq: 1 },
            { y: 402, seq: 2 },
        ]);
    });

    it("re-anchors the server paddle when the socket epoch bumps", () => {
        // given
        const harness = makeHarness();
        const { rerender, result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);

        // when
        harness.notification.wsEpoch = 2;
        rerender();
        frame(1010);

        // then
        expect(sentPayloads(harness.sendWSMessage)).toEqual([
            { y: 400, seq: 1 },
            { y: 400, seq: 2 },
        ]);
    });

    it("sends nothing at all when input is disabled", () => {
        // given
        const harness = makeHarness();
        setup(harness, false);

        // when
        frame(1000);
        frame(2000);

        // then
        expect(harness.sendWSMessage).not.toHaveBeenCalled();
    });

    it("sends nothing at all without a room", () => {
        // given
        const harness = makeHarness();
        setup(harness, true, null);

        // when
        frame(1000);

        // then
        expect(harness.sendWSMessage).not.toHaveBeenCalled();
    });

    it("clamps a pointer beyond the bottom of the court to the paddle limit", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);

        // when
        result.current.setTargetFromPointer(4000, { top: 0, height: 400 });
        frame(1000);

        // then
        expect(sentPayloads(harness.sendWSMessage)).toEqual([{ y: COURT_HEIGHT - PADDLE_HEIGHT / 2, seq: 1 }]);
    });

    it("clamps a pointer above the top of the court to the paddle limit", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);

        // when
        result.current.setTargetFromPointer(-500, { top: 0, height: 400 });
        frame(1000);

        // then
        expect(sentPayloads(harness.sendWSMessage)).toEqual([{ y: PADDLE_HEIGHT / 2, seq: 1 }]);
    });

    it("maps a pointer inside the canvas through to court units", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);

        // when
        result.current.setTargetFromPointer(140, { top: 40, height: 400 });
        frame(1000);

        // then
        expect(sentPayloads(harness.sendWSMessage)).toEqual([{ y: 200, seq: 1 }]);
    });

    it("advances the target while an arrow key is held", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);

        // when
        act(() => {
            window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown" }));
        });
        frame(1000 + PONG_INPUT_INTERVAL_MS + 10);

        // then
        const payloads = sentPayloads(harness.sendWSMessage);
        expect(payloads).toHaveLength(2);
        expect(payloads[1].y).toBeGreaterThan(400);
    });

    it("stops advancing the target once the key is released", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        frame(1000);
        act(() => {
            window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowUp" }));
        });
        frame(1060);
        const moved = result.current.targetRef.current;

        // when
        act(() => {
            window.dispatchEvent(new KeyboardEvent("keyup", { key: "ArrowUp" }));
        });
        frame(1120);

        // then
        expect(moved).toBeLessThan(400);
        expect(result.current.targetRef.current).toBe(moved);
    });

    const editableCases = [
        { name: "an input", tag: "input", key: "w" },
        { name: "a textarea", tag: "textarea", key: "s" },
        { name: "a select", tag: "select", key: "ArrowDown" },
        { name: "a contenteditable box", tag: "div", key: "s" },
    ];

    it.each(editableCases)("leaves $key alone when it was typed into $name", ({ tag, key }) => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);
        const node = document.createElement(tag);
        if (tag === "div") {
            Object.defineProperty(node, "isContentEditable", { value: true });
        }
        document.body.appendChild(node);

        // when
        const event = new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true });
        act(() => {
            node.dispatchEvent(event);
        });
        frame(1000 + PONG_INPUT_INTERVAL_MS + 10);

        // then
        expect(event.defaultPrevented).toBe(false);
        expect(result.current.targetRef.current).toBe(400);
        expect(harness.sendWSMessage).toHaveBeenCalledTimes(1);
        node.remove();
    });

    it("still claims a paddle key typed outside an editable node", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);

        // when
        const event = new KeyboardEvent("keydown", { key: "s", bubbles: true, cancelable: true });
        act(() => {
            window.dispatchEvent(event);
        });
        frame(1060);

        // then
        expect(event.defaultPrevented).toBe(true);
        expect(result.current.targetRef.current).toBeGreaterThan(400);
    });

    it("releases a held key when the window loses focus", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);
        act(() => {
            window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown" }));
        });
        frame(1060);
        const moved = result.current.targetRef.current;

        // when
        act(() => {
            window.dispatchEvent(new Event("blur"));
        });
        frame(1120);

        // then
        expect(moved).toBeGreaterThan(400);
        expect(result.current.targetRef.current).toBe(moved);
    });

    it("releases a held key when focus moves into the chat box", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);
        act(() => {
            window.dispatchEvent(new KeyboardEvent("keydown", { key: "s" }));
        });
        frame(1060);
        const moved = result.current.targetRef.current;
        const input = document.createElement("input");
        document.body.appendChild(input);

        // when
        act(() => {
            input.dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
        });
        frame(1120);

        // then
        expect(moved).toBeGreaterThan(400);
        expect(result.current.targetRef.current).toBe(moved);
        input.remove();
    });

    it("releases a held key when the tab is hidden", () => {
        // given
        const harness = makeHarness();
        const { result } = setup(harness);
        result.current.seedTargetY(400);
        frame(1000);
        act(() => {
            window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowUp" }));
        });
        frame(1060);
        const moved = result.current.targetRef.current;

        // when
        Object.defineProperty(document, "hidden", { configurable: true, get: () => true });
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });
        frame(1120);
        Reflect.deleteProperty(document, "hidden");

        // then
        expect(moved).toBeLessThan(400);
        expect(result.current.targetRef.current).toBe(moved);
    });
});
