import { Input } from "../../../../components/Input/Input";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function WebPushSection({ form }: SectionProps) {
    const { settings, updateField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Web Push</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>VAPID Public Key</span>
                    <Input
                        value={settings.web_push_vapid_key ?? ""}
                        onChange={e => updateField("web_push_vapid_key", e.target.value)}
                        fullWidth
                        placeholder="BEl62iUYgUivxIkv69yViEuiBIa..."
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Firebase Web API Key</span>
                    <Input
                        value={settings.web_push_firebase_api_key ?? ""}
                        onChange={e => updateField("web_push_firebase_api_key", e.target.value)}
                        fullWidth
                        placeholder="AIzaSy..."
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Firebase Project ID</span>
                    <Input
                        value={settings.web_push_firebase_project_id ?? ""}
                        onChange={e => updateField("web_push_firebase_project_id", e.target.value)}
                        fullWidth
                        placeholder="city-of-books"
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Firebase Messaging Sender ID</span>
                    <Input
                        value={settings.web_push_firebase_sender_id ?? ""}
                        onChange={e => updateField("web_push_firebase_sender_id", e.target.value)}
                        fullWidth
                        placeholder="1234567890"
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Firebase Web App ID</span>
                    <Input
                        value={settings.web_push_firebase_app_id ?? ""}
                        onChange={e => updateField("web_push_firebase_app_id", e.target.value)}
                        fullWidth
                        placeholder="1:1234567890:web:abcdef"
                    />
                </div>
            </div>
        </div>
    );
}
