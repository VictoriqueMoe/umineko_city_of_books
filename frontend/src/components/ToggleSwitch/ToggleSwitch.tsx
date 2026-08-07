import styles from "./ToggleSwitch.module.css";

interface ToggleSwitchProps {
    enabled: boolean;
    onChange: (enabled: boolean) => void;
    label: string;
    description?: string;
    ariaLabel?: string;
    disabled?: boolean;
}

export function ToggleSwitch({ enabled, onChange, label, description, ariaLabel, disabled }: ToggleSwitchProps) {
    return (
        <button
            type="button"
            className={`${styles.row}${disabled ? ` ${styles.disabled}` : ""}`}
            onClick={() => onChange(!enabled)}
            role="switch"
            aria-checked={enabled}
            aria-label={ariaLabel ?? label}
            disabled={disabled}
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
