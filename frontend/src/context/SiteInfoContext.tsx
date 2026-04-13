import { type PropsWithChildren, useEffect, useState } from "react";
import type { SiteInfo } from "../api/endpoints";
import { getSiteInfo } from "../api/endpoints";
import { SiteInfoContext } from "./siteInfoContextValue";

export function SiteInfoProvider({ children }: PropsWithChildren) {
    const [siteInfo, setSiteInfo] = useState<SiteInfo | null>(null);

    useEffect(() => {
        getSiteInfo()
            .then(setSiteInfo)
            .catch(() => {});

        function handleRefresh() {
            getSiteInfo()
                .then(setSiteInfo)
                .catch(() => {});
        }
        function handleVisibility() {
            if (document.visibilityState === "visible") {
                handleRefresh();
            }
        }
        const pollId = setInterval(handleRefresh, 5 * 60 * 1000);

        window.addEventListener("site-info-refresh", handleRefresh);
        document.addEventListener("visibilitychange", handleVisibility);
        return () => {
            clearInterval(pollId);
            window.removeEventListener("site-info-refresh", handleRefresh);
            document.removeEventListener("visibilitychange", handleVisibility);
        };
    }, []);

    if (!siteInfo) {
        return null;
    }

    return <SiteInfoContext.Provider value={siteInfo}>{children}</SiteInfoContext.Provider>;
}
