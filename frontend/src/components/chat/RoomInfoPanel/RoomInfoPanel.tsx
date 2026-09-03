import { useEffect } from "react";
import { TabStrip, type TabDefinition } from "../../Tabs/TabStrip";
import { tabId, tabPanelId } from "../../Tabs/tabIds";
import { SearchTab } from "./tabs/SearchTab";
import { PinsTab } from "./tabs/PinsTab";
import { MediaTab } from "./tabs/MediaTab";
import { LinksTab } from "./tabs/LinksTab";
import styles from "./RoomInfoPanel.module.css";

export type RoomInfoTab = "search" | "pins" | "media" | "links";

const TABS: TabDefinition<RoomInfoTab>[] = [
    { id: "search", label: "Search" },
    { id: "media", label: "Media" },
    { id: "pins", label: "Pins" },
    { id: "links", label: "Links" },
];

const ID_PREFIX = "room-info";

interface RoomInfoPanelProps {
    roomId: string;
    tab: RoomInfoTab | null;
    onTabChange: (tab: RoomInfoTab) => void;
    onClose: () => void;
    onJump: (messageId: string, createdAt?: string) => void;
    canUnpin: boolean;
    onLightbox?: (src: string) => void;
}

export function RoomInfoPanel({ roomId, tab, onTabChange, onClose, onJump, canUnpin, onLightbox }: RoomInfoPanelProps) {
    useEffect(() => {
        if (!tab) {
            return;
        }

        function onKeyDown(e: KeyboardEvent) {
            if (e.key === "Escape") {
                onClose();
            }
        }

        document.addEventListener("keydown", onKeyDown);

        return () => {
            document.removeEventListener("keydown", onKeyDown);
        };
    }, [tab, onClose]);

    if (!tab) {
        return null;
    }

    return (
        <div className={styles.overlay} onClick={onClose}>
            <aside
                className={styles.drawer}
                onClick={e => e.stopPropagation()}
                role="dialog"
                aria-label="Room information"
            >
                <header className={styles.header}>
                    <TabStrip
                        tabs={TABS}
                        active={tab}
                        onSelect={onTabChange}
                        idPrefix={ID_PREFIX}
                        ariaLabel="Room information"
                        className={styles.tabStrip}
                        tabClassName={styles.tab}
                        activeTabClassName={styles.tabActive}
                    />
                    <button type="button" className={styles.closeBtn} onClick={onClose} aria-label="Close">
                        {"✕"}
                    </button>
                </header>
                <div
                    role="tabpanel"
                    id={tabPanelId(ID_PREFIX, tab)}
                    aria-labelledby={tabId(ID_PREFIX, tab)}
                    className={styles.panel}
                >
                    {tab === "search" && <SearchTab roomId={roomId} isActive onJump={onJump} onJumped={onClose} />}
                    {tab === "pins" && (
                        <PinsTab
                            roomId={roomId}
                            isActive
                            onJump={onJump}
                            onJumped={onClose}
                            canUnpin={canUnpin}
                            onLightbox={onLightbox}
                        />
                    )}
                    {tab === "media" && (
                        <MediaTab roomId={roomId} isActive onJump={onJump} onJumped={onClose} onLightbox={onLightbox} />
                    )}
                    {tab === "links" && <LinksTab roomId={roomId} isActive onJump={onJump} onJumped={onClose} />}
                </div>
            </aside>
        </div>
    );
}
