import { Button } from "../../../../components/Button/Button";
import { Input } from "../../../../components/Input/Input";
import { Select } from "../../../../components/Select/Select";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

type EmailProvider = "smtp" | "cloudflare";
const EMAIL_PROVIDER_SMTP: EmailProvider = "smtp";
const EMAIL_PROVIDER_CLOUDFLARE: EmailProvider = "cloudflare";

export function EmailSection({ form }: SectionProps) {
    const { settings, updateField, getNumber, emailTest } = form;

    return (
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
                    <Button variant="secondary" onClick={emailTest.send} disabled={emailTest.sending}>
                        {emailTest.sending ? "Sending..." : "Send test email"}
                    </Button>
                    {emailTest.message && <span className={styles.success}>{emailTest.message}</span>}
                    {emailTest.error && <span className={styles.saveError}>{emailTest.error}</span>}
                </div>
            </div>
        </div>
    );
}
