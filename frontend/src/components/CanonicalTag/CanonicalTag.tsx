import { useEffect } from "react";
import { useLocation } from "react-router";

export function CanonicalTag() {
    const { pathname, search } = useLocation();
    useEffect(() => {
        const href = `${window.location.origin}${pathname}${search}`;
        let link = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');
        if (!link) {
            link = document.createElement("link");
            link.rel = "canonical";
            document.head.appendChild(link);
        }
        link.href = href;
    }, [pathname, search]);
    return null;
}
