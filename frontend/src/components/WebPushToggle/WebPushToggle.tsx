import { useState } from "react";
import { ToggleSwitch } from "../ToggleSwitch/ToggleSwitch";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import {
    disableWebPush,
    enableWebPush,
    webPushConfigured,
    webPushEnabled,
    webPushSupported,
} from "../../utils/webPush";
import styles from "./WebPushToggle.module.css";

export function WebPushToggle() {
    const siteInfo = useSiteInfo();
    const [enabled, setEnabled] = useState(() => webPushEnabled());
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState("");

    const config = siteInfo.web_push;

    if (!siteInfo.push_enabled || !webPushConfigured(config) || !webPushSupported()) {
        return null;
    }

    function handleChange(next: boolean) {
        setBusy(true);
        setError("");

        const action = next ? enableWebPush(config) : disableWebPush(config);

        action
            .then(() => setEnabled(next))
            .catch(() =>
                setError(
                    next
                        ? "Could not turn notifications on. Your browser may have blocked them for this site."
                        : "Could not turn notifications off.",
                ),
            )
            .finally(() => setBusy(false));
    }

    return (
        <div className={styles.wrapper}>
            <ToggleSwitch
                enabled={enabled}
                onChange={handleChange}
                disabled={busy}
                label="Push Notifications"
                description="Get notified on this device even when the site is closed. Turn this on separately on each device you use."
            />
            {error ? <span className={styles.error}>{error}</span> : null}
        </div>
    );
}
