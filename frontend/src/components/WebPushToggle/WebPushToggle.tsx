import { ToggleSwitch } from "../ToggleSwitch/ToggleSwitch";
import { useWebPush } from "../../hooks/useWebPush";
import styles from "./WebPushToggle.module.css";

export function WebPushToggle() {
    const { available, enabled, busy, error, setEnabled } = useWebPush();

    if (!available) {
        return null;
    }

    return (
        <div className={styles.wrapper}>
            <ToggleSwitch
                enabled={enabled}
                onChange={setEnabled}
                disabled={busy}
                label="Push Notifications"
                description="Get notified on this device even when the site is closed. Turn this on separately on each device you use."
            />
            {error ? <span className={styles.error}>{error}</span> : null}
        </div>
    );
}
