import { useEffect, useState } from "react";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { isNativeApp } from "../../platform/capabilities";
import { applyOtaUpdate, hasOtaUpdate, subscribeOtaReady } from "../../platform/appUpdate";
import styles from "./StaleVersionBanner.module.css";

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
        window.location.reload();
    }

    if (native) {
        if (!otaReady) {
            return null;
        }

        return (
            <div className={styles.banner} role="alert">
                <span className={styles.text}>A new version is available. Tap to update now.</span>
                <button type="button" onClick={handleApply} className={styles.button}>
                    Update now
                </button>
            </div>
        );
    }

    if (bundleVersion === "dev" || !siteInfo.version || siteInfo.version === "dev") {
        return null;
    }

    if (siteInfo.version === bundleVersion) {
        return null;
    }

    return (
        <div className={styles.banner} role="alert">
            <span className={styles.text}>A new version of the site is available. Please reload to update.</span>
            <button type="button" onClick={handleReload} className={styles.button}>
                Reload now
            </button>
        </div>
    );
}
