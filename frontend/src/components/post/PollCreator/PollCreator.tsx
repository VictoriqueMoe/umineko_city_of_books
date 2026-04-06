import { Button } from "../../Button/Button";
import styles from "./PollCreator.module.css";

const POLL_DURATIONS = [
    { value: 3600, label: "1 hour" },
    { value: 14400, label: "4 hours" },
    { value: 28800, label: "8 hours" },
    { value: 43200, label: "12 hours" },
    { value: 86400, label: "24 hours" },
    { value: 259200, label: "3 days" },
    { value: 604800, label: "1 week" },
    { value: 1209600, label: "2 weeks" },
];

interface PollCreatorProps {
    options: string[];
    onOptionsChange: (options: string[]) => void;
    duration: number;
    onDurationChange: (seconds: number) => void;
    onRemove: () => void;
}

export function PollCreator({ options, onOptionsChange, duration, onDurationChange, onRemove }: PollCreatorProps) {
    function updateOption(index: number, value: string) {
        const next = [...options];
        next[index] = value;
        onOptionsChange(next);
    }

    function removeOption(index: number) {
        onOptionsChange(options.filter((_, i) => i !== index));
    }

    function addOption() {
        if (options.length < 10) {
            onOptionsChange([...options, ""]);
        }
    }

    return (
        <div className={styles.pollCreator}>
            <div className={styles.header}>
                <span className={styles.title}>Poll</span>
                <Button variant="ghost" size="small" onClick={onRemove}>
                    Remove Poll
                </Button>
            </div>

            <div className={styles.options}>
                {options.map((opt, i) => (
                    <div key={i} className={styles.optionRow}>
                        <input
                            type="text"
                            value={opt}
                            onChange={e => updateOption(i, e.target.value)}
                            placeholder={`Option ${i + 1}`}
                            maxLength={200}
                        />
                        {options.length > 2 && (
                            <button
                                type="button"
                                className={styles.removeBtn}
                                onClick={() => removeOption(i)}
                                aria-label="Remove option"
                            >
                                ×
                            </button>
                        )}
                    </div>
                ))}
            </div>

            <div className={styles.footer}>
                <Button variant="ghost" size="small" onClick={addOption} disabled={options.length >= 10}>
                    + Add Option
                </Button>
                <select
                    className={styles.durationSelect}
                    value={duration}
                    onChange={e => onDurationChange(Number(e.target.value))}
                >
                    {POLL_DURATIONS.map(d => (
                        <option key={d.value} value={d.value}>
                            {d.label}
                        </option>
                    ))}
                </select>
            </div>
        </div>
    );
}
