import { useEffect } from "react";
import { useNotifications } from "./useNotifications";

const BASE_TITLE = "Umineko City of Books";

export function usePageTitle(title?: string) {
    const { unreadCount } = useNotifications();
    useEffect(() => {
        const full = title ? `${title} | ${BASE_TITLE}` : BASE_TITLE;
        document.title = unreadCount > 0 ? `(${unreadCount}) ${full}` : full;
    }, [title, unreadCount]);
}
