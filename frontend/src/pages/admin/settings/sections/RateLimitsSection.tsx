import { Input } from "../../../../components/Input/Input";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

const LIMITS = [
    {
        key: "rate_limit_credentials",
        label: "Login, Register and Password Reset (per minute)",
        hint: "Attempts per IP address. Login also counts per username. Set to 0 to disable.",
    },
    {
        key: "rate_limit_mail",
        label: "Account Emails (per hour)",
        hint: "Forgot password, set email and resend verification, per IP address. Forgot password also counts per username. Set to 0 to disable.",
    },
    {
        key: "rate_limit_chat_send",
        label: "Chat Messages (per 5 seconds)",
        hint: "Per member. Set to 0 to disable.",
    },
    {
        key: "rate_limit_uploads",
        label: "Attachment Uploads (per minute)",
        hint: "Per member. Set to 0 to disable.",
    },
    {
        key: "rate_limit_link_preview",
        label: "Link Preview Lookups (per minute)",
        hint: "Per IP address. Set to 0 to disable.",
    },
    {
        key: "rate_limit_stream_token",
        label: "Stream Viewer Joins (per minute)",
        hint: "Per IP address. Set to 0 to disable.",
    },
    {
        key: "rate_limit_stream_thumbnail",
        label: "Stream Thumbnail Uploads (per minute)",
        hint: "Per IP address. Set to 0 to disable.",
    },
    {
        key: "rate_limit_stream_credentials",
        label: "Stream Key Resets (per minute)",
        hint: "Per IP address. Set to 0 to disable.",
    },
    {
        key: "max_queued_uploads_per_user",
        label: "Media Jobs Queued Per Member",
        hint: "How many of one member's images and videos can be in the processing queue at once. Any more wait their turn. A change applies once that member has nothing queued. Must be at least 1.",
    },
    {
        key: "link_preview_max_fetches",
        label: "Link Previews Fetched At Once",
        hint: "Across the whole site. When every fetch is busy, a link shows no preview card until it is requested again. Must be at least 1.",
    },
];

export function RateLimitsSection({ form }: SectionProps) {
    const { updateField, getNumber } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Rate Limits</h2>
            <div className={styles.fieldGroup}>
                {LIMITS.map(limit => (
                    <div key={limit.key} className={styles.field}>
                        <span className={styles.fieldLabel}>{limit.label}</span>
                        <Input
                            type="number"
                            value={getNumber(limit.key)}
                            onChange={e => updateField(limit.key, e.target.value)}
                        />
                        <span className={styles.fieldHint}>{limit.hint}</span>
                    </div>
                ))}
            </div>
        </div>
    );
}
