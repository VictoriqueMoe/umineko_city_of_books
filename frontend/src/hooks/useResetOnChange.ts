import { useState, type Dispatch, type SetStateAction } from "react";

export function useResetOnChange(key: unknown, reset: () => void): void {
    const [syncedKey, setSyncedKey] = useState(key);

    if (syncedKey !== key) {
        setSyncedKey(key);
        reset();
    }
}

export function useSyncedState<T>(source: T): [T, Dispatch<SetStateAction<T>>] {
    const [value, setValue] = useState(source);

    useResetOnChange(source, () => setValue(source));

    return [value, setValue];
}
