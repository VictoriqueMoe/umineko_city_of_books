import { Input } from "../../../../components/Input/Input";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function LimitsSection({ form }: SectionProps) {
    const { updateField, getNumber } = form;

    return (
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
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>New Account Restriction (hours)</span>
                    <Input
                        type="number"
                        value={getNumber("new_account_hours")}
                        onChange={e => updateField("new_account_hours", e.target.value)}
                    />
                    <span className={styles.fieldHint}>
                        For this long after signing up, a new member cannot post attachments or links, and their posts
                        do not render link previews. Set to 0 to disable.
                    </span>
                </div>
            </div>
        </div>
    );
}
