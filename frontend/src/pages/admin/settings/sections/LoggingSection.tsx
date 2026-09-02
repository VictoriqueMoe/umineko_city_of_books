import { Input } from "../../../../components/Input/Input";
import { Select } from "../../../../components/Select/Select";
import type { SectionProps } from "./types";
import styles from "../AdminSettings.module.css";

export function LoggingSection({ form }: SectionProps) {
    const { settings, updateField } = form;

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Logging & Observability</h2>
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
    );
}
