import { useEffect } from "react";
import { useNotifications } from "./useNotifications";
import { useSiteInfo } from "./useSiteInfo";

export function usePageTitle(title?: string) {
    const { unreadCount } = useNotifications();
    const { site_name } = useSiteInfo();
    useEffect(() => {
        const full = title ? `${title} | ${site_name}` : site_name;
        document.title = unreadCount > 0 ? `(${unreadCount}) ${full}` : full;
    }, [title, unreadCount, site_name]);
}
