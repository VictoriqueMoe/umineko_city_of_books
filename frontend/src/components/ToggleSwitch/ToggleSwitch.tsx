import styles from "./ToggleSwitch.module.css";

interface ToggleSwitchProps {
    enabled: boolean;
    onChange: (enabled: boolean) => void;
    label: string;
    description?: string;
}

export function ToggleSwitch({ enabled, onChange, label, description }: ToggleSwitchProps) {
    return (
        <button
            type="button"
            className={styles.row}
            onClick={() => onChange(!enabled)}
            role="switch"
            aria-checked={enabled}
            aria-label={label}
        >
            <div className={styles.info}>
                <span className={styles.label}>{label}</span>
                {description && <span className={styles.desc}>{description}</span>}
            </div>
            <span className={`${styles.toggle}${enabled ? ` ${styles.on}` : ""}`}>
                <span className={styles.knob} />
            </span>
        </button>
    );
}
