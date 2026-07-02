import { useEffect } from "react";
import { useNavigate } from "react-router";
import { Capacitor } from "@capacitor/core";
import { isInternalOrigin } from "../../utils/siteOrigin";

export function NativeLinkInterceptor() {
    const navigate = useNavigate();

    useEffect(() => {
        if (!Capacitor.isNativePlatform()) {
            return;
        }

        function handleClick(event: MouseEvent) {
            if (
                event.defaultPrevented ||
                event.button !== 0 ||
                event.metaKey ||
                event.ctrlKey ||
                event.shiftKey ||
                event.altKey
            ) {
                return;
            }

            const target = event.target as Element | null;
            const anchor = target?.closest("a");
            if (!anchor || anchor.hasAttribute("download")) {
                return;
            }

            const href = anchor.getAttribute("href");
            if (!href) {
                return;
            }

            let url: URL;
            try {
                url = new URL(href, window.location.href);
            } catch {
                return;
            }

            if ((url.protocol !== "http:" && url.protocol !== "https:") || !isInternalOrigin(url.origin)) {
                return;
            }

            event.preventDefault();
            navigate(url.pathname + url.search + url.hash);
        }

        document.addEventListener("click", handleClick);
        return () => {
            document.removeEventListener("click", handleClick);
        };
    }, [navigate]);

    return null;
}
