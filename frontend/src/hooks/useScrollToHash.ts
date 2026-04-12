import { useEffect } from "react";

export function useScrollToHash(ready: boolean, elementId: string | null) {
    useEffect(() => {
        if (!ready || !elementId) {
            return;
        }

        const t = setTimeout(() => {
            const el = document.getElementById(elementId);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
            }
        }, 300);
        return () => clearTimeout(t);
    }, [ready, elementId]);
}
