import styles from "./VolumeSlider.module.css";

interface VolumeSliderProps {
    value: number;
    onChange: (value: number) => void;
    ariaLabel?: string;
    className?: string;
}

export function VolumeSlider({ value, onChange, ariaLabel = "Volume", className }: VolumeSliderProps) {
    const classes = className ? `${styles.volume} ${className}` : styles.volume;

    return (
        <label className={classes}>
            <span aria-hidden="true">{value === 0 ? "\u{1F507}" : "\u{1F50A}"}</span>
            <input
                type="range"
                min={0}
                max={1}
                step={0.01}
                value={value}
                onChange={e => onChange(Number(e.target.value))}
                aria-label={ariaLabel}
            />
        </label>
    );
}
