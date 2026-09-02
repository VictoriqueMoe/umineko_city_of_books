import { Input } from "../../../../components/Input/Input";
import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { isEnabled } from "../../../../domain/siteSettings";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function MobileAppSection({ form }: SectionProps) {
    const { settings, updateField, toggleField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Mobile App</h2>
            <div className={styles.fieldGroup}>
                <ToggleSwitch
                    label="Enable Push Notifications"
                    description="Send native push notifications to the mobile app when a recipient is offline (requires FCM_CREDENTIALS_FILE on the server)"
                    enabled={isEnabled(settings.push_enabled)}
                    onChange={v => toggleField("push_enabled", v)}
                />
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Latest App Version</span>
                    <Input
                        value={settings.app_latest_version ?? ""}
                        onChange={e => updateField("app_latest_version", e.target.value)}
                        fullWidth
                        placeholder="1.0.0"
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>App Download URL</span>
                    <Input
                        value={settings.app_download_url ?? ""}
                        onChange={e => updateField("app_download_url", e.target.value)}
                        fullWidth
                        placeholder="https://github.com/VictoriqueMoe/umineko_city_of_books/releases/latest"
                    />
                </div>
            </div>
        </div>
    );
}
