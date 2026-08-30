import { useEffect } from "react";
import { useLocation } from "react-router";
import { recordLocation } from "../../platform/lastLocation";

export function LastLocationTracker() {
    const location = useLocation();

    useEffect(() => {
        recordLocation(location.pathname, location.search, location.hash);
    }, [location.pathname, location.search, location.hash]);

    return null;
}
