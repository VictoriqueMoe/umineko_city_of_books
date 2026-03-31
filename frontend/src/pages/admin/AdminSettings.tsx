import { useEffect, useState } from "react";
import { getAdminSettings, updateAdminSettings } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Select } from "../../components/Select/Select";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import type { SiteSettings } from "../../types/api";
import styles from "./AdminSettings.module.css";

const BYTES_PER_MB = 1024 * 1024;

export function AdminSettings() {
    const [settings, setSettings] = useState<SiteSettings>({});
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");

    useEffect(() => {
        getAdminSettings()
            .then(setSettings)
            .catch(e => setError(e.message))
            .finally(() => setLoading(false));
    }, []);

    function updateField(key: string, value: string) {
        setSettings(prev => ({ ...prev, [key]: value }));
        setSuccess("");
    }

    function toggleField(key: string, enabled: boolean) {
        updateField(key, enabled ? "true" : "false");
    }

    function getNumber(key: string): string {
        return settings[key] ?? "0";
    }

    function getMB(key: string): string {
        const bytes = parseInt(settings[key] ?? "0", 10);
        if (isNaN(bytes)) {
            return "0";
        }
        return String(Math.round(bytes / BYTES_PER_MB));
    }

    function setMB(key: string, mb: string) {
        const mbNum = parseFloat(mb);
        if (isNaN(mbNum)) {
            updateField(key, "0");
        } else {
            updateField(key, String(Math.round(mbNum * BYTES_PER_MB)));
        }
    }

    function validateSettings(): string | null {
        const maxBody = parseInt(settings.max_body_size ?? "0", 10);
        const maxImage = parseInt(settings.max_image_size ?? "0", 10);
        const maxVideo = parseInt(settings.max_video_size ?? "0", 10);
        const maxGeneral = parseInt(settings.max_general_size ?? "0", 10);
        const minPassword = parseInt(settings.min_password_length ?? "0", 10);
        const sessionDays = parseInt(settings.session_duration_days ?? "0", 10);
        const maxTheories = parseInt(settings.max_theories_per_day ?? "0", 10);
        const maxResponses = parseInt(settings.max_responses_per_day ?? "0", 10);

        if (maxBody <= 0) {
            return "Max body size must be greater than 0";
        }
        if (maxImage <= 0) {
            return "Max image size must be greater than 0";
        }
        if (maxImage > maxBody) {
            return `Max image size (${Math.round(maxImage / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
        }
        if (maxVideo > maxBody) {
            return `Max video size (${Math.round(maxVideo / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
        }
        if (maxGeneral > maxBody) {
            return `Max general size (${Math.round(maxGeneral / BYTES_PER_MB)} MB) cannot exceed max body size (${Math.round(maxBody / BYTES_PER_MB)} MB)`;
        }
        if (minPassword < 1) {
            return "Minimum password length must be at least 1";
        }
        if (sessionDays < 1) {
            return "Session duration must be at least 1 day";
        }
        if (maxTheories < 0) {
            return "Max theories per day cannot be negative";
        }
        if (maxResponses < 0) {
            return "Max responses per day cannot be negative";
        }
        return null;
    }

    async function handleSave() {
        const validationError = validateSettings();
        if (validationError) {
            setError(validationError);
            return;
        }

        setSaving(true);
        setError("");
        setSuccess("");
        try {
            await updateAdminSettings(settings);
            setSuccess("Settings saved successfully");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save settings");
        } finally {
            setSaving(false);
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading settings...</div>;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Site Settings</h1>

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
                        enabled={settings.maintenance_mode === "true"}
                        onChange={v => toggleField("maintenance_mode", v)}
                    />
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Turnstile (Cloudflare)</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Turnstile"
                        description="Require Cloudflare Turnstile verification on login and registration"
                        enabled={settings.turnstile_enabled === "true"}
                        onChange={v => toggleField("turnstile_enabled", v)}
                    />
                    {settings.turnstile_enabled === "true" && (
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

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Limits</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Theories Per Day</span>
                        <Input
                            type="number"
                            value={getNumber("max_theories_per_day")}
                            onChange={e => updateField("max_theories_per_day", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Responses Per Day</span>
                        <Input
                            type="number"
                            value={getNumber("max_responses_per_day")}
                            onChange={e => updateField("max_responses_per_day", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Min Password Length</span>
                        <Input
                            type="number"
                            value={getNumber("min_password_length")}
                            onChange={e => updateField("min_password_length", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Session Duration (days)</span>
                        <Input
                            type="number"
                            value={getNumber("session_duration_days")}
                            onChange={e => updateField("session_duration_days", e.target.value)}
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>File Size Limits</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Image Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_image_size")}
                            onChange={e => setMB("max_image_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Video Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_video_size")}
                            onChange={e => setMB("max_video_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max General Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_general_size")}
                            onChange={e => setMB("max_general_size", e.target.value)}
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Max Body Size (MB)</span>
                        <Input
                            type="number"
                            value={getMB("max_body_size")}
                            onChange={e => setMB("max_body_size", e.target.value)}
                        />
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Appearance</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Default Theme</span>
                        <Select
                            value={settings.default_theme ?? "featherine"}
                            onChange={e => updateField("default_theme", e.target.value)}
                        >
                            <option value="featherine">Featherine</option>
                            <option value="bernkastel">Bernkastel</option>
                            <option value="lambdadelta">Lambdadelta</option>
                        </Select>
                    </div>
                </div>
            </div>

            <div className={styles.saveRow}>
                <Button variant="primary" onClick={handleSave} disabled={saving}>
                    {saving ? "Saving..." : "Save Settings"}
                </Button>
                {error && <span className={styles.saveError}>{error}</span>}
                {success && <span className={styles.success}>{success}</span>}
            </div>
        </div>
    );
}
