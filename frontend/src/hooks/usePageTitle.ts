import { useEffect } from "react";
import { useNotifications } from "./useNotifications";
import { useSiteInfo } from "./useSiteInfo";

const FIRST_STRONG_ISOLATE = String.fromCharCode(0x2068);
const POP_DIRECTIONAL_ISOLATE = String.fromCharCode(0x2069);

export function usePageTitle(title?: string) {
    const { unreadCount } = useNotifications();
    const { site_name } = useSiteInfo();
    useEffect(() => {
        const isolated = title ? `${FIRST_STRONG_ISOLATE}${title}${POP_DIRECTIONAL_ISOLATE}` : "";
        const full = isolated ? `${isolated} | ${site_name}` : site_name;

        document.title = unreadCount > 0 ? `(${unreadCount}) ${full}` : full;
    }, [title, unreadCount, site_name]);
}
