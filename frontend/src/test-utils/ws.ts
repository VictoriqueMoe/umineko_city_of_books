import { act } from "@testing-library/react";
import { vi, type Mock } from "vitest";
import { dispatch, type RealtimeEventHandler, type RealtimeUnsubscribe } from "../api/realtime/bus";
import type { RealtimeEvent, RealtimeEventName } from "../api/realtime/events";
import type { RealtimeCommand } from "../api/realtime/outbound";

export class FakeWebSocket {
    static readonly OPEN = 1;
    static readonly CLOSED = 3;
    static instances: FakeWebSocket[] = [];

    readyState = FakeWebSocket.OPEN;
    closeCount = 0;
    sent: string[] = [];
    onopen: (() => void) | null = null;
    onmessage: ((event: { data: string }) => void) | null = null;
    onclose: (() => void) | null = null;
    onerror: (() => void) | null = null;

    constructor(readonly url: string) {
        FakeWebSocket.instances.push(this);
    }

    send(payload: string): void {
        this.sent.push(payload);
    }

    close(): void {
        this.closeCount += 1;
        this.readyState = FakeWebSocket.CLOSED;
    }
}

export interface RealtimeTestEvent {
    type: RealtimeEventName;
    data?: unknown;
}

export type RealtimeTestNames = RealtimeEventName | readonly RealtimeEventName[];

interface Subscription {
    names: readonly RealtimeEventName[];
    handler: RealtimeEventHandler;
}

export interface WSHarness {
    subscribe: Mock<(names: RealtimeTestNames, handler: RealtimeEventHandler) => RealtimeUnsubscribe>;
    unsubscribe: Mock<() => void>;
    sendRealtime: Mock<(command: RealtimeCommand) => void>;
    getEpoch: () => number;
    subscribeEpoch: (listener: () => void) => () => void;
    emit: (event: RealtimeTestEvent) => void;
    reconnect: () => void;
    flush: () => Promise<void>;
}

export function emitRealtimeEvent(event: RealtimeTestEvent): void {
    act(() => {
        dispatch(event as RealtimeEvent);
    });
}

export async function emitSettledRealtimeEvent(event: RealtimeTestEvent): Promise<void> {
    await act(async () => {
        dispatch(event as RealtimeEvent);

        await new Promise(resolve => setTimeout(resolve, 0));
    });
}

export function makeWSHarness(): WSHarness {
    const subscriptions: Subscription[] = [];
    const epochListeners = new Set<() => void>();

    let epoch = 0;

    const unsubscribe: Mock<() => void> = vi.fn();

    const subscribe = vi.fn((names: RealtimeTestNames, handler: RealtimeEventHandler): RealtimeUnsubscribe => {
        const entry: Subscription = { names: typeof names === "string" ? [names] : names, handler };
        subscriptions.push(entry);

        let released = false;

        return () => {
            unsubscribe();

            if (released) {
                return;
            }
            released = true;

            const index = subscriptions.indexOf(entry);
            if (index >= 0) {
                subscriptions.splice(index, 1);
            }
        };
    });

    const sendRealtime: Mock<(command: RealtimeCommand) => void> = vi.fn();

    function getEpoch(): number {
        return epoch;
    }

    function subscribeEpoch(listener: () => void): () => void {
        epochListeners.add(listener);

        return () => {
            epochListeners.delete(listener);
        };
    }

    function emit(event: RealtimeTestEvent): void {
        act(() => {
            for (const entry of subscriptions.slice()) {
                if (!entry.names.includes(event.type)) {
                    continue;
                }
                entry.handler(event as RealtimeEvent);
            }
        });
    }

    function reconnect(): void {
        act(() => {
            epoch += 1;
            for (const listener of Array.from(epochListeners)) {
                listener();
            }
        });
    }

    async function flush(): Promise<void> {
        await act(async () => {});
    }

    return { subscribe, unsubscribe, sendRealtime, getEpoch, subscribeEpoch, emit, reconnect, flush };
}
