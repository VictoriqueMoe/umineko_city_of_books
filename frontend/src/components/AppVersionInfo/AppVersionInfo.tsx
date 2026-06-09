import { useEffect, useState } from "react";
import { App } from "@capacitor/app";
import { CapacitorUpdater } from "@capgo/capacitor-updater";
import { isNativeApp } from "../../utils/authToken";
import styles from "./AppVersionInfo.module.css";

export function AppVersionInfo() {
    const [info, setInfo] = useState<string | null>(null);

    useEffect(() => {
        if (!isNativeApp()) {
            return;
        }

        async function load() {
            const parts: string[] = [];

            try {
                const appInfo = await App.getInfo();
                parts.push(`app ${appInfo.version} (${appInfo.build})`);
            } catch {}

            try {
                const current = await CapacitorUpdater.current();
                const version = current.bundle.version || "dev";
                const short = version.length > 12 ? version.slice(0, 12) : version;
                parts.push(`bundle ${short}`);
            } catch {
                parts.push("bundle dev");
            }

            if (parts.length > 0) {
                setInfo(parts.join(" · "));
            }
        }

        load().catch(() => {});
    }, []);

    if (!info) {
        return null;
    }

    return <p className={styles.version}>{info}</p>;
}
