import { Input } from "../../../../components/Input/Input";
import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { isEnabled } from "../../../../domain/siteSettings";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function TurnstileSection({ form }: SectionProps) {
    const { settings, updateField, toggleField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Turnstile (Cloudflare)</h2>
            <div className={styles.fieldGroup}>
                <ToggleSwitch
                    label="Enable Turnstile"
                    description="Require Cloudflare Turnstile verification on login and registration"
                    enabled={isEnabled(settings.turnstile_enabled)}
                    onChange={v => toggleField("turnstile_enabled", v)}
                />
                {isEnabled(settings.turnstile_enabled) && (
                    <>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Site Key</span>
                            <Input
                                value={settings.turnstile_site_key ?? ""}
                                onChange={e => updateField("turnstile_site_key", e.target.value)}
                                fullWidth
                                placeholder="0x..."
                            />
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Secret Key</span>
                            <Input
                                type="password"
                                value={settings.turnstile_secret_key ?? ""}
                                onChange={e => updateField("turnstile_secret_key", e.target.value)}
                                fullWidth
                                placeholder="0x..."
                            />
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}
