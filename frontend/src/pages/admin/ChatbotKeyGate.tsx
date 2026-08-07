import { Button } from "../../components/Button/Button";
import styles from "./ChatbotKeyGate.module.css";

const NO_KEY_MESSAGE =
    "No OpenAI API key is saved yet. Everything below stays locked until a key is saved and the provider answers with its model list.";
const CHECKING_MESSAGE = "Checking the saved OpenAI API key against the provider...";
const EMPTY_LIST_MESSAGE = "OpenAI answered, but listed no models for this key, so everything below stays locked.";
const LOCKED_SUFFIX = "Everything below stays locked until the model list can be read.";

interface ChatbotKeyGateProps {
    apiKeySaved: boolean;
    checking: boolean;
    reason?: string;
    onRetry: () => void;
}

export function ChatbotKeyGate({ apiKeySaved, checking, reason, onRetry }: ChatbotKeyGateProps) {
    if (!apiKeySaved) {
        return (
            <div className={styles.gate}>
                <span className={styles.message}>{NO_KEY_MESSAGE}</span>
            </div>
        );
    }

    if (checking) {
        return (
            <div className={styles.gate}>
                <span className={styles.message}>{CHECKING_MESSAGE}</span>
            </div>
        );
    }

    return (
        <div className={styles.gate}>
            <span className={styles.message}>{reason ? `${reason} ${LOCKED_SUFFIX}` : EMPTY_LIST_MESSAGE}</span>
            <div className={styles.actions}>
                <Button variant="secondary" size="small" onClick={onRetry}>
                    Try again
                </Button>
            </div>
        </div>
    );
}
