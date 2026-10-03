import { Input } from "../../../../components/Input/Input";
import { Select } from "../../../../components/Select/Select";
import { ToggleSwitch } from "../../../../components/ToggleSwitch/ToggleSwitch";
import { isEnabled } from "../../../../domain/siteSettings";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

type StorageBackend = "local" | "s3";
const STORAGE_BACKEND_LOCAL: StorageBackend = "local";
const STORAGE_BACKEND_S3: StorageBackend = "s3";

export function StorageSection({ form }: SectionProps) {
    const { settings, updateField, toggleField } = form;
    const backend = settings.storage_backend ?? STORAGE_BACKEND_LOCAL;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>File Storage</h2>
            <div className={styles.fieldGroup}>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Active Storage</span>
                    <Select value={backend} onChange={e => updateField("storage_backend", e.target.value)}>
                        <option value={STORAGE_BACKEND_LOCAL}>Local disk</option>
                        <option value={STORAGE_BACKEND_S3}>S3 compatible</option>
                    </Select>
                    <span className={styles.fieldHint}>
                        Where new uploads are written. Switching does not move anything: every file remembers the
                        storage, bucket and path it was written to and keeps being served from there, so existing local
                        files stay on disk after you switch to S3.
                    </span>
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>S3 Endpoint</span>
                    <Input
                        value={settings.s3_endpoint ?? ""}
                        onChange={e => updateField("s3_endpoint", e.target.value)}
                        fullWidth
                        placeholder="https://<account>.r2.cloudflarestorage.com"
                    />
                    <span className={styles.fieldHint}>
                        Leave empty for Amazon S3. Set it for any S3 compatible provider such as Cloudflare R2,
                        Backblaze B2 or MinIO.
                    </span>
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>S3 Region</span>
                    <Input
                        value={settings.s3_region ?? ""}
                        onChange={e => updateField("s3_region", e.target.value)}
                        fullWidth
                        placeholder="auto"
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>S3 Bucket</span>
                    <Input
                        value={settings.s3_bucket ?? ""}
                        onChange={e => updateField("s3_bucket", e.target.value)}
                        fullWidth
                        placeholder="city-of-books-uploads"
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>S3 Key Prefix</span>
                    <Input
                        value={settings.s3_prefix ?? ""}
                        onChange={e => updateField("s3_prefix", e.target.value)}
                        fullWidth
                        placeholder="uploads"
                    />
                    <span className={styles.fieldHint}>
                        Optional folder inside the bucket. Use a bucket or prefix dedicated to this site: the nightly
                        orphan cleanup deletes unreferenced uploads it finds there.
                    </span>
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>S3 Access Key ID</span>
                    <Input
                        value={settings.s3_access_key_id ?? ""}
                        onChange={e => updateField("s3_access_key_id", e.target.value)}
                        fullWidth
                    />
                </div>
                <div className={styles.field}>
                    <span className={styles.fieldLabel}>S3 Secret Access Key</span>
                    <Input
                        type="password"
                        value={settings.s3_secret_access_key ?? ""}
                        onChange={e => updateField("s3_secret_access_key", e.target.value)}
                        fullWidth
                    />
                </div>
                <ToggleSwitch
                    label="S3 Force Path Style"
                    description="Address the bucket as endpoint/bucket instead of bucket.endpoint. MinIO and some self-hosted providers need this."
                    enabled={isEnabled(settings.s3_force_path_style)}
                    onChange={v => toggleField("s3_force_path_style", v)}
                />
                <span className={styles.fieldHint}>
                    S3 is enabled as soon as region, bucket, access key ID and secret are all set, and changes take
                    effect immediately with no restart. Saving checks the bucket first and refuses settings it cannot
                    reach. Changing the bucket or prefix only affects new uploads; older files keep loading from where
                    they were written, so the keys must still be able to read the old bucket. While any upload lives in
                    S3, the endpoint cannot be changed and the connection cannot be cleared.
                </span>
            </div>
        </div>
    );
}
