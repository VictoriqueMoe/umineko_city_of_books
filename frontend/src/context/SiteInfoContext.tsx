import { type PropsWithChildren, useEffect } from "react";
import { useSiteInfoQuery } from "../api/queries/auth";
import { SiteInfoContext } from "./siteInfoContextValue";

export function SiteInfoProvider({ children }: PropsWithChildren) {
    const { siteInfo, refresh } = useSiteInfoQuery();

    useEffect(() => {
        function handleRefresh() {
            void refresh();
        }
        function handleVisibility() {
            if (document.visibilityState === "visible") {
                handleRefresh();
            }
        }
        window.addEventListener("site-info-refresh", handleRefresh);
        document.addEventListener("visibilitychange", handleVisibility);
        return () => {
            window.removeEventListener("site-info-refresh", handleRefresh);
            document.removeEventListener("visibilitychange", handleVisibility);
        };
    }, [refresh]);

    if (!siteInfo) {
        return null;
    }

    return <SiteInfoContext.Provider value={siteInfo}>{children}</SiteInfoContext.Provider>;
}
