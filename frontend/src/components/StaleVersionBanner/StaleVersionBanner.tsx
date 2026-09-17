import { useEffect, useState } from "react";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { isNativeApp } from "../../platform/capabilities";
import { applyOtaUpdate, hasOtaUpdate, subscribeOtaReady } from "../../platform/appUpdate";
import { reloadPage } from "../../platform/pageReload";
import { Banner, BannerButton } from "../Banner/Banner";

export function StaleVersionBanner() {
    const siteInfo = useSiteInfo();
    const bundleVersion = __APP_VERSION__;
    const native = isNativeApp();
    const [otaReady, setOtaReady] = useState(() => native && hasOtaUpdate());

    useEffect(() => {
        if (!native) {
            return;
        }

        return subscribeOtaReady(() => {
            setOtaReady(true);
        });
    }, [native]);

    function handleApply() {
        applyOtaUpdate().catch(() => {});
    }

    function handleReload() {
        reloadPage();
    }

    if (native) {
        if (!otaReady) {
            return null;
        }

        return (
            <Banner colour="red" role="alert" actions={<BannerButton onClick={handleApply}>Update now</BannerButton>}>
                A new version is available. Tap to update now.
            </Banner>
        );
    }

    if (bundleVersion === "dev" || !siteInfo.version || siteInfo.version === "dev") {
        return null;
    }

    if (siteInfo.version === bundleVersion) {
        return null;
    }

    return (
        <Banner colour="red" role="alert" actions={<BannerButton onClick={handleReload}>Reload now</BannerButton>}>
            A new version of the site is available. Please reload to update.
        </Banner>
    );
}
