interface ToggleSwitchProps {
    enabled: boolean;
    onChange: (enabled: boolean) => void;
    label: string;
    description?: string;
}

export function ToggleSwitch({ enabled, onChange, label, description }: ToggleSwitchProps) {
    return (
        <button
            className="toggle-switch-row"
            onClick={() => onChange(!enabled)}
            role="switch"
            aria-checked={enabled}
            aria-label={label}
        >
            <div className="toggle-switch-info">
                <span className="toggle-switch-label">{label}</span>
                {description && <span className="toggle-switch-desc">{description}</span>}
            </div>
            <span className={`toggle-switch${enabled ? " on" : ""}`}>
                <span className="toggle-switch-knob" />
            </span>
        </button>
    );
}
