import {type PropsWithChildren, useEffect, useState} from "react";
import type {SiteInfo} from "../api/endpoints";
import {getSiteInfo} from "../api/endpoints";
import {SiteInfoContext} from "./siteInfoContextValue";

export function SiteInfoProvider({ children }: PropsWithChildren) {
    const [siteInfo, setSiteInfo] = useState<SiteInfo | null>(null);

    useEffect(() => {
        getSiteInfo()
            .then(setSiteInfo)
            .catch(() => {});
    }, []);

    if (!siteInfo) {
        return null;
    }

    return <SiteInfoContext.Provider value={siteInfo}>{children}</SiteInfoContext.Provider>;
}
