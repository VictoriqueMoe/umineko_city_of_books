import { useEffect, useState } from "react";
import { Capacitor } from "@capacitor/core";
import { App } from "@capacitor/app";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import styles from "./NativeUpdateBanner.module.css";

function isOlder(installed: string, latest: string): boolean {
    const a = installed.split(".").map(Number);
    const b = latest.split(".").map(Number);
    const len = Math.max(a.length, b.length);

    for (let i = 0; i < len; i++) {
        const x = a[i] ?? 0;
        const y = b[i] ?? 0;
        if (x !== y) {
            return x < y;
        }
    }

    return false;
}

export function NativeUpdateBanner() {
    const siteInfo = useSiteInfo();
    const [installed, setInstalled] = useState<string | null>(null);

    useEffect(() => {
        if (!Capacitor.isNativePlatform()) {
            return;
        }

        App.getInfo()
            .then(info => setInstalled(info.version))
            .catch(() => {});
    }, []);

    const latest = siteInfo.app_latest_version;
    const url = siteInfo.app_download_url;

    if (!installed || !latest || !url) {
        return null;
    }

    if (!isOlder(installed, latest)) {
        return null;
    }

    function handleDownload() {
        window.open(url, "_blank");
    }

    return (
        <div className={styles.banner} role="alert">
            <span className={styles.text}>A new version of the app is available.</span>
            <button type="button" onClick={handleDownload} className={styles.button}>
                Download update
            </button>
        </div>
    );
}
