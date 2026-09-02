import { Input } from "../../../../components/Input/Input";
import { Select } from "../../../../components/Select/Select";
import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { isEnabled } from "../../../../domain/siteSettings";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function FeatureTogglesSection({ form }: SectionProps) {
    const { settings, updateField, toggleField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Feature Toggles</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Registration</span>
                    <Select
                        value={settings.registration_type ?? "open"}
                        onChange={e => updateField("registration_type", e.target.value)}
                    >
                        <option value="open">Open (anyone can register)</option>
                        <option value="invite">Invite Only</option>
                        <option value="closed">Closed (no registration)</option>
                    </Select>
                </div>
                <ToggleSwitch
                    label="Maintenance Mode"
                    description="Put the site into maintenance mode"
                    enabled={isEnabled(settings.maintenance_mode)}
                    onChange={v => toggleField("maintenance_mode", v)}
                />
                {isEnabled(settings.maintenance_mode) && (
                    <>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Maintenance Title</span>
                            <Input
                                value={settings.maintenance_title ?? ""}
                                onChange={e => updateField("maintenance_title", e.target.value)}
                                fullWidth
                                placeholder="The game board is being prepared"
                            />
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Maintenance Message</span>
                            <Input
                                value={settings.maintenance_message ?? ""}
                                onChange={e => updateField("maintenance_message", e.target.value)}
                                fullWidth
                                placeholder="Without love, it cannot be seen. Please check back shortly."
                            />
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
