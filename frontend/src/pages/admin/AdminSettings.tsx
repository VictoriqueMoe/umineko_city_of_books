import { useState } from "react";
import { useAdminSettings } from "../../api/queries/admin";
import { useSendTestEmail, useUpdateAdminSettings } from "../../api/mutations/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Select } from "../../components/Select/Select";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import type { SiteSettings } from "../../types/api";
import styles from "./AdminSettings.module.css";

const BYTES_PER_MB = 1024 * 1024;

type EmailProvider = "smtp" | "cloudflare";
const EMAIL_PROVIDER_SMTP: EmailProvider = "smtp";
const EMAIL_PROVIDER_CLOUDFLARE: EmailProvider = "cloudflare";

export function AdminSettings() {
    usePageTitle("Admin - Settings");
    const { settings: loadedSettings, loading } = useAdminSettings();
    const updateSettingsMutation = useUpdateAdminSettings();
    const sendTestEmailMutation = useSendTestEmail();
    const [draft, setDraft] = useState<SiteSettings>({});
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");
    const [testMessage, setTestMessage] = useState("");
    const [testError, setTestError] = useState("");

    const saving = updateSettingsMutation.isPending;
    const settings: SiteSettings = { ...(loadedSettings ?? {}), ...draft };

    function updateField(key: string, value: string) {
        setDraft(prev => ({ ...prev, [key]: value }));
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
        if (settings.voice_enabled === "true") {
            if (!settings.livekit_url || !settings.livekit_api_key || !settings.livekit_api_secret) {
                return "Voice chat requires LiveKit URL, API key and API secret";
            }
        }
        if (settings.email_provider === EMAIL_PROVIDER_CLOUDFLARE) {
            if (!settings.cloudflare_account_id || !settings.cloudflare_api_token || !settings.cloudflare_email_from) {
                return "Cloudflare email requires account ID, API token and from address";
            }
        }
        return null;
    }

    async function handleSave() {
        const validationError = validateSettings();
        if (validationError) {
            setError(validationError);
            return;
        }

        setError("");
        setSuccess("");
        try {
            await updateSettingsMutation.mutateAsync(settings);
            setSuccess("Settings saved successfully");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save settings");
        }
    }

    async function handleSendTestEmail() {
        setTestMessage("");
        setTestError("");
        try {
            await sendTestEmailMutation.mutateAsync();
            setTestMessage("Test email sent. Check your inbox.");
        } catch (e) {
            setTestError(e instanceof Error ? e.message : "Failed to send test email");
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
                    {settings.maintenance_mode === "true" && (
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
                <h2 className={styles.sectionTitle}>Voice Chat (LiveKit)</h2>
                <div className={styles.fieldGroup}>
                    <ToggleSwitch
                        label="Enable Voice Chat"
                        description="Allow voice calls in chat rooms and DMs (requires a self-hosted LiveKit server)"
                        enabled={settings.voice_enabled === "true"}
                        onChange={v => toggleField("voice_enabled", v)}
                    />
                    {settings.voice_enabled === "true" && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>LiveKit URL</span>
                                <Input
                                    value={settings.livekit_url ?? ""}
                                    onChange={e => updateField("livekit_url", e.target.value)}
                                    fullWidth
                                    placeholder="wss://livekit.example.com"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Key</span>
                                <Input
                                    value={settings.livekit_api_key ?? ""}
                                    onChange={e => updateField("livekit_api_key", e.target.value)}
                                    fullWidth
                                    placeholder="APIxxxxxxxx"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Secret</span>
                                <Input
                                    type="password"
                                    value={settings.livekit_api_secret ?? ""}
                                    onChange={e => updateField("livekit_api_secret", e.target.value)}
                                    fullWidth
                                    placeholder="secret"
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
                <h2 className={styles.sectionTitle}>Email</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Email Provider</span>
                        <Select
                            value={settings.email_provider ?? EMAIL_PROVIDER_SMTP}
                            onChange={e => updateField("email_provider", e.target.value)}
                        >
                            <option value={EMAIL_PROVIDER_SMTP}>SMTP</option>
                            <option value={EMAIL_PROVIDER_CLOUDFLARE}>Cloudflare Email Service</option>
                        </Select>
                    </div>
                    {(settings.email_provider ?? EMAIL_PROVIDER_SMTP) === EMAIL_PROVIDER_SMTP && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Host</span>
                                <Input
                                    value={settings.smtp_host ?? ""}
                                    onChange={e => updateField("smtp_host", e.target.value)}
                                    fullWidth
                                    placeholder="127.0.0.1"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Port</span>
                                <Input
                                    type="number"
                                    value={getNumber("smtp_port")}
                                    onChange={e => updateField("smtp_port", e.target.value)}
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>From Address</span>
                                <Input
                                    value={settings.smtp_from ?? ""}
                                    onChange={e => updateField("smtp_from", e.target.value)}
                                    fullWidth
                                    placeholder="noreply@example.com"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Username</span>
                                <Input
                                    value={settings.smtp_username ?? ""}
                                    onChange={e => updateField("smtp_username", e.target.value)}
                                    fullWidth
                                    placeholder="Leave empty for no auth"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>SMTP Password</span>
                                <Input
                                    type="password"
                                    value={settings.smtp_password ?? ""}
                                    onChange={e => updateField("smtp_password", e.target.value)}
                                    fullWidth
                                    placeholder="Leave empty for no auth"
                                />
                            </div>
                        </>
                    )}
                    {settings.email_provider === EMAIL_PROVIDER_CLOUDFLARE && (
                        <>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>Account ID</span>
                                <Input
                                    value={settings.cloudflare_account_id ?? ""}
                                    onChange={e => updateField("cloudflare_account_id", e.target.value)}
                                    fullWidth
                                    placeholder="Cloudflare account ID"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>API Token</span>
                                <Input
                                    type="password"
                                    value={settings.cloudflare_api_token ?? ""}
                                    onChange={e => updateField("cloudflare_api_token", e.target.value)}
                                    fullWidth
                                    placeholder="Token with email sending permission"
                                />
                            </div>
                            <div className={styles.field}>
                                <span className={styles.fieldLabel}>From Address</span>
                                <Input
                                    value={settings.cloudflare_email_from ?? ""}
                                    onChange={e => updateField("cloudflare_email_from", e.target.value)}
                                    fullWidth
                                    placeholder="noreply@yourdomain.com"
                                />
                            </div>
                        </>
                    )}
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Sends a test email to your own account using the saved settings. Save changes first.
                        </span>
                        <Button
                            variant="secondary"
                            onClick={handleSendTestEmail}
                            disabled={sendTestEmailMutation.isPending}
                        >
                            {sendTestEmailMutation.isPending ? "Sending..." : "Send test email"}
                        </Button>
                        {testMessage && <span className={styles.success}>{testMessage}</span>}
                        {testError && <span className={styles.saveError}>{testError}</span>}
                    </div>
                </div>
            </div>

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Logging & Error Reporting</h2>
                <div className={styles.fieldGroup}>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Log Level</span>
                        <Select
                            value={settings.log_level ?? "info"}
                            onChange={e => updateField("log_level", e.target.value)}
                        >
                            <option value="trace">Trace</option>
                            <option value="debug">Debug</option>
                            <option value="info">Info</option>
                            <option value="warn">Warn</option>
                            <option value="error">Error</option>
                        </Select>
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Sentry DSN</span>
                        <Input
                            value={settings.sentry_dsn ?? ""}
                            onChange={e => updateField("sentry_dsn", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            OTLP endpoint (OpenTelemetry traces, e.g. http://tempo:4318)
                        </span>
                        <Input
                            value={settings.otlp_endpoint ?? ""}
                            onChange={e => updateField("otlp_endpoint", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable tracing"
                        />
                    </div>
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>
                            Pyroscope URL (continuous profiling, e.g. http://pyroscope:4040)
                        </span>
                        <Input
                            value={settings.pyroscope_url ?? ""}
                            onChange={e => updateField("pyroscope_url", e.target.value)}
                            fullWidth
                            placeholder="Leave empty to disable profiling"
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
                            <option value="beatrice">Beatrice</option>
                            <option value="bernkastel">Bernkastel</option>
                            <option value="lambdadelta">Lambdadelta</option>
                            <option value="erika">Erika Furudo</option>
                            <option value="battler">Battler Ushiromiya</option>
                            <option value="virgilia">Virgilia</option>
                            <option value="rika">Rika Furude</option>
                            <option value="mion">Mion Sonozaki</option>
                            <option value="satoko">Satoko Houjou</option>
                            <option value="miyao">Miyao</option>
                            <option value="lingji">Lingji</option>
                            <option value="stanislaw">Stanis&#322;aw</option>
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
