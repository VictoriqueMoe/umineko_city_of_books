import type { ReactNode } from "react";
import styles from "./TypingIndicator.module.css";

interface TypingIndicatorProps {
    names: string[];
}

export function TypingIndicator({ names }: TypingIndicatorProps) {
    if (names.length === 0) {
        return null;
    }

    let text: ReactNode;
    if (names.length === 1) {
        text = (
            <>
                <bdi>{names[0]}</bdi>
                {" is typing..."}
            </>
        );
    } else if (names.length === 2) {
        text = (
            <>
                <bdi>{names[0]}</bdi>
                {" and "}
                <bdi>{names[1]}</bdi>
                {" are typing..."}
            </>
        );
    } else if (names.length === 3) {
        text = (
            <>
                <bdi>{names[0]}</bdi>
                {", "}
                <bdi>{names[1]}</bdi>
                {" and "}
                <bdi>{names[2]}</bdi>
                {" are typing..."}
            </>
        );
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
