import { useEffect } from "react";

export function useWarnBeforeLeaving(active: boolean): void {
    useEffect(() => {
        if (!active) {
            return;
        }

        function warn(event: BeforeUnloadEvent) {
            event.preventDefault();
            event.returnValue = "";
        }

        window.addEventListener("beforeunload", warn);

        return () => {
            window.removeEventListener("beforeunload", warn);
        };
    }, [active]);
}
