import styles from "./TypingIndicator.module.css";

interface TypingIndicatorProps {
    names: string[];
}

export function TypingIndicator({ names }: TypingIndicatorProps) {
    if (names.length === 0) {
        return null;
    }

    let text: string;
    if (names.length === 1) {
        text = `${names[0]} is typing...`;
    } else if (names.length === 2) {
        text = `${names[0]} and ${names[1]} are typing...`;
    } else if (names.length === 3) {
        text = `${names[0]}, ${names[1]} and ${names[2]} are typing...`;
    } else {
        text = "Multiple people are typing...";
    }

    return (
        <div className={styles.indicator}>
            <span className={styles.dots} aria-hidden="true">
                <span />
                <span />
                <span />
            </span>
            <span className={styles.text}>{text}</span>
        </div>
    );
}
