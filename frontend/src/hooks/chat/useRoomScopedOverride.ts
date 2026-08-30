import { type Dispatch, type SetStateAction, useCallback, useState } from "react";

interface Override<T> {
    roomId: string | null;
    value: T | null;
}

const NO_OVERRIDE = { roomId: null, value: null };

export function useRoomScopedOverride<T>(roomId: string | undefined, base: T): [T, Dispatch<SetStateAction<T>>] {
    const [override, setOverride] = useState<Override<T>>(NO_OVERRIDE);

    const held = override.roomId === roomId ? override.value : null;
    const value = held === null ? base : held;

    const set: Dispatch<SetStateAction<T>> = useCallback(
        updater => {
            setOverride(prev => {
                const current = prev.roomId === roomId && prev.value !== null ? prev.value : base;
                const next = typeof updater === "function" ? (updater as (previous: T) => T)(current) : updater;

                return { roomId: roomId ?? null, value: next };
            });
        },
        [roomId, base],
    );

    return [value, set];
}
