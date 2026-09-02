import { Input } from "../../../../components/Input/Input";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function FileSizeLimitsSection({ form }: SectionProps) {
    const { getMB, setMB, getMP, setMP } = form;

    return (
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
                    <span className={styles.fieldLabel}>Max Image Pixels (megapixels)</span>
                    <Input
                        type="number"
                        value={getMP("max_image_pixels")}
                        onChange={e => setMP("max_image_pixels", e.target.value)}
                    />
                    <span className={styles.fieldHint}>
                        Rejects images whose width x height exceeds this, however small the file is. A highly
                        compressible image can be tiny on disk yet need gigabytes of memory to decode.
                    </span>
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
                    <span className={styles.fieldLabel}>Max Audio Size (MB)</span>
                    <Input
                        type="number"
                        value={getMB("max_audio_size")}
                        onChange={e => setMB("max_audio_size", e.target.value)}
                    />
                    <span className={styles.fieldHint}>Applies to MP3, M4A, OGG, WAV and FLAC uploads.</span>
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
    );
}
