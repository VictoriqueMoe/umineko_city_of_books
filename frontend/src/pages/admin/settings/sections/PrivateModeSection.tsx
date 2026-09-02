import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { isEnabled } from "../../../../domain/siteSettings";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function PrivateModeSection({ form }: SectionProps) {
    const { settings, toggleField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Private Mode</h2>
            <div className={styles.fieldGroup}>
                <ToggleSwitch
                    label="Require a login for everything"
                    description="Nobody who is not signed in can see or do anything: no pages, no API, no uploads, no link previews, and search engines are told to index nothing. Signing in, resetting a password and verifying an email keep working."
                    enabled={isEnabled(settings.private_mode)}
                    onChange={v => toggleField("private_mode", v)}
                />
            </div>
        </div>
    );
}
