import { type PropsWithChildren, useEffect, useRef } from "react";
import { useSiteInfoQuery } from "../api/queries/auth";
import { SiteInfoContext } from "./siteInfoContextValue";

const MIN_REFETCH_INTERVAL_MS = 2000;

export function SiteInfoProvider({ children }: PropsWithChildren) {
    const { siteInfo, refresh, dataUpdatedAt } = useSiteInfoQuery();

    const dataUpdatedAtRef = useRef(dataUpdatedAt);
    useEffect(() => {
        dataUpdatedAtRef.current = dataUpdatedAt;
    }, [dataUpdatedAt]);

    useEffect(() => {
        function handleRefresh() {
            if (Date.now() - dataUpdatedAtRef.current < MIN_REFETCH_INTERVAL_MS) {
                return;
            }
            refresh();
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
