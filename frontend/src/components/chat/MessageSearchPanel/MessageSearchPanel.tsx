import { SearchTab } from "../RoomInfoPanel/tabs/SearchTab";
import styles from "./MessageSearchPanel.module.css";

interface MessageSearchPanelProps {
    roomId: string;
    isOpen: boolean;
    onClose: () => void;
    onJump: (messageId: string, createdAt?: string) => void;
}

export function MessageSearchPanel({ roomId, isOpen, onClose, onJump }: MessageSearchPanelProps) {
    if (!isOpen) {
        return null;
    }

    return (
        <div className={styles.overlay} onClick={onClose}>
            <aside
                className={styles.drawer}
                onClick={e => e.stopPropagation()}
                role="dialog"
                aria-label="Search messages"
            >
                <header className={styles.header}>
                    <span className={styles.title}>Search messages</span>
                    <button type="button" className={styles.closeBtn} onClick={onClose} aria-label="Close">
                        {"✕"}
                    </button>
                </header>
                <SearchTab roomId={roomId} isActive={isOpen} onJump={onJump} onJumped={onClose} />
            </aside>
        </div>
    );
}
