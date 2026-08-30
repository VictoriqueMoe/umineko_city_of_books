import { type PropsWithChildren, useEffect, useRef } from "react";
import { useSiteInfoQuery } from "../hooks/queries/site";
import { SiteInfoContext } from "./siteInfoContextValue";

const MIN_REFETCH_INTERVAL_MS = 2000;

export function SiteInfoProvider({ children }: PropsWithChildren) {
    const { siteInfo, refresh, dataUpdatedAt } = useSiteInfoQuery();

    const dataUpdatedAtRef = useRef(dataUpdatedAt);
    useEffect(() => {
        dataUpdatedAtRef.current = dataUpdatedAt;
    }, [dataUpdatedAt]);

    useEffect(() => {
        function handleVisibility() {
            if (document.visibilityState !== "visible") {
                return;
            }
            if (Date.now() - dataUpdatedAtRef.current < MIN_REFETCH_INTERVAL_MS) {
                return;
            }

            refresh().catch(() => {});
        }
        document.addEventListener("visibilitychange", handleVisibility);
        return () => {
            document.removeEventListener("visibilitychange", handleVisibility);
        };
    }, [refresh]);

    if (!siteInfo) {
        return null;
    }

    return <SiteInfoContext.Provider value={siteInfo}>{children}</SiteInfoContext.Provider>;
}
