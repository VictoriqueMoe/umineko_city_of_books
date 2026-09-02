import { Input } from "../../../../components/Input/Input";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function GeneralSection({ form }: SectionProps) {
    const { settings, updateField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>General</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Site Name</span>
                    <Input
                        value={settings.site_name ?? ""}
                        onChange={e => updateField("site_name", e.target.value)}
                        fullWidth
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Site Description</span>
                    <Input
                        value={settings.site_description ?? ""}
                        onChange={e => updateField("site_description", e.target.value)}
                        fullWidth
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Announcement Banner</span>
                    <Input
                        value={settings.announcement_banner ?? ""}
                        onChange={e => updateField("announcement_banner", e.target.value)}
                        fullWidth
                    />
                </div>
            </div>
        </div>
    );
}
