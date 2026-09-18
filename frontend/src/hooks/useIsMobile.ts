import { useEffect, useState } from "react";
import { Capacitor } from "@capacitor/core";

export const MOBILE_QUERY = "(max-width: 960px)";

function readIsMobile(): boolean {
    if (Capacitor.isNativePlatform()) {
        return true;
    }

    if (typeof window === "undefined") {
        return false;
    }

    return window.matchMedia(MOBILE_QUERY).matches;
}

export function useIsMobile(): boolean {
    const native = Capacitor.isNativePlatform();
    const [isMobile, setIsMobile] = useState(readIsMobile);

    useEffect(() => {
        if (native) {
            return;
        }

        const mql = window.matchMedia(MOBILE_QUERY);

        function onChange() {
            setIsMobile(mql.matches);
        }

        mql.addEventListener("change", onChange);
        return () => {
            mql.removeEventListener("change", onChange);
        };
    }, [native]);

    return native || isMobile;
}
