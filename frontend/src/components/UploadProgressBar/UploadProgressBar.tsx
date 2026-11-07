import { uploadStatus } from "../../domain/uploadProgress";
import styles from "./UploadProgressBar.module.css";

type UploadProgressBarProps = {
    percent: number;
};

export function UploadProgressBar({ percent }: UploadProgressBarProps) {
    const processing = percent >= 100;

    return (
        <div className={styles.wrapper}>
            <div
                className={styles.track}
                role="progressbar"
                aria-label="Upload progress"
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={processing ? undefined : percent}
                aria-valuetext={uploadStatus(percent)}
            >
                <div
                    className={processing ? `${styles.fill} ${styles.processing}` : styles.fill}
                    style={{ "--percent": `${percent}%` } as React.CSSProperties}
                />
            </div>
            <span className={styles.status} aria-hidden="true">
                {processing ? "Processing" : `${percent}%`}
            </span>
        </div>
    );
}
