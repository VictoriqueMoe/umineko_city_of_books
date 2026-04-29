import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "sidebar-collapsed";

function readInitial(): boolean {
    if (typeof window === "undefined") {
        return false;
    }
    try {
        return window.localStorage.getItem(STORAGE_KEY) === "true";
    } catch {
        return false;
    }
}

export function useSidebarCollapsed(): [boolean, (next: boolean | ((prev: boolean) => boolean)) => void] {
    const [collapsed, setCollapsed] = useState<boolean>(readInitial);

    useEffect(() => {
        try {
            window.localStorage.setItem(STORAGE_KEY, collapsed ? "true" : "false");
        } catch {}
    }, [collapsed]);

    const update = useCallback((next: boolean | ((prev: boolean) => boolean)) => {
        setCollapsed(prev => (typeof next === "function" ? next(prev) : next));
    }, []);

    return [collapsed, update];
}
