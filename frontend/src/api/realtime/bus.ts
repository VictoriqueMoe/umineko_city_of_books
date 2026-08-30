import type { RealtimeEvent, RealtimeEventName, RealtimeEventOf } from "./events";

export type RealtimeEventHandler<K extends RealtimeEventName = RealtimeEventName> = (event: RealtimeEventOf<K>) => void;

export type RealtimeUnsubscribe = () => void;

type StoredHandler = (event: RealtimeEvent) => void;

const handlersByName = new Map<RealtimeEventName, Set<StoredHandler>>();

function asNameList<K extends RealtimeEventName>(names: K | readonly K[]): readonly K[] {
    return typeof names === "string" ? [names] : names;
}

function rethrowAsync(error: unknown): void {
    queueMicrotask(() => {
        throw error;
    });
}

export function subscribe<K extends RealtimeEventName>(
    names: K | readonly K[],
    handler: RealtimeEventHandler<K>,
): RealtimeUnsubscribe {
    const wanted = asNameList(names);
    const stored = handler as StoredHandler;

    for (const name of wanted) {
        const existing = handlersByName.get(name);
        if (existing) {
            existing.add(stored);
            continue;
        }
        handlersByName.set(name, new Set([stored]));
    }

    let released = false;

    return () => {
        if (released) {
            return;
        }
        released = true;

        for (const name of wanted) {
            const set = handlersByName.get(name);
            if (!set) {
                continue;
            }
            set.delete(stored);
            if (set.size === 0) {
                handlersByName.delete(name);
            }
        }
    };
}

export function dispatch(event: RealtimeEvent): void {
    const subscribed = handlersByName.get(event.type);
    if (!subscribed || subscribed.size === 0) {
        return;
    }

    const snapshot = Array.from(subscribed);

    for (const handler of snapshot) {
        try {
            handler(event);
        } catch (error) {
            rethrowAsync(error);
        }
    }
}
