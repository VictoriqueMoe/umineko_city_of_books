import { useContext } from "react";
import { SiteInfoContext } from "../context/siteInfoContextValue";

export function useSiteInfo() {
    const ctx = useContext(SiteInfoContext);
    if (!ctx) {
        throw new Error("useSiteInfo must be used within SiteInfoProvider");
    }
    return ctx;
}
