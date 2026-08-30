import { useEffect, useRef, useSyncExternalStore } from "react";
import { subscribe, type RealtimeEventHandler } from "./bus";
import type { RealtimeEventName, RealtimeEventOf } from "./events";
import { ensureRealtimePipeline, getRealtimeEpoch, subscribeRealtimeEpoch } from "./pipeline";

const NAME_SEPARATOR = "\n";

function subscriptionKey<K extends RealtimeEventName>(names: K | readonly K[]): string {
    return typeof names === "string" ? names : names.join(NAME_SEPARATOR);
}

export function useRealtimeEvent<K extends RealtimeEventName>(
    names: K | readonly K[],
    handler: RealtimeEventHandler<K>,
): void {
    const namesRef = useRef(names);
    const handlerRef = useRef(handler);

    useEffect(() => {
        namesRef.current = names;
        handlerRef.current = handler;
    });

    const key = subscriptionKey(names);

    useEffect(() => {
        ensureRealtimePipeline();

        return subscribe(namesRef.current, (event: RealtimeEventOf<K>) => {
            handlerRef.current(event);
        });
    }, [key]);
}

export function useRealtimeStatus(): number {
    return useSyncExternalStore(subscribeRealtimeEpoch, getRealtimeEpoch);
}
