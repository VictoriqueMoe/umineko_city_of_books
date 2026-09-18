import {
    type PointerEvent as ReactPointerEvent,
    type ReactElement,
    type ReactNode,
    type RefObject,
    useCallback,
    useEffect,
    useId,
    useRef,
    useState,
} from "react";
import { useChatViewport } from "../../../hooks/useChatViewport";
import styles from "./WatchPartyMobile.module.css";

export interface WatchPartyMobileViewProps {
    title: string;
    hasControl: boolean;
    canEnd: boolean;
    busy: boolean;
    copied: boolean;
    watcherCount: number;
    voiceCount: number;
    onCopyInvite: () => void;
    onHide: () => void;
    onLeave: () => void;
    onEnd: () => void;
    stageRef: RefObject<HTMLDivElement | null>;
    stage: ReactNode;
    chat: ReactNode;
    voice: ReactNode;
    people: ReactNode;
}

type PaneId = "chat" | "voice" | "people";

interface PaneTab {
    id: PaneId;
    label: string;
    pane: ReactNode;
}

interface ActionItem {
    id: string;
    label: string;
    onSelect: () => void;
    disabled: boolean;
    destructive: boolean;
}

const MIN_STAGE_HEIGHT = 96;
const MAX_STAGE_FRACTION = 0.7;
const KEYBOARD_INSET = 120;

export function WatchPartyMobileView({
    title,
    hasControl,
    canEnd,
    busy,
    copied,
    watcherCount,
    voiceCount,
    onCopyInvite,
    onHide,
    onLeave,
    onEnd,
    stageRef,
    stage,
    chat,
    voice,
    people,
}: WatchPartyMobileViewProps): ReactElement {
    const [tab, setTab] = useState<PaneId>("chat");
    const [menuOpen, setMenuOpen] = useState(false);
    const [keyboardOpen, setKeyboardOpen] = useState(false);
    const [stageHeight, setStageHeight] = useState<number | null>(null);
    const dragRef = useRef<{ startY: number; startHeight: number } | null>(null);
    const menuWrapRef = useRef<HTMLDivElement | null>(null);
    const menuButtonRef = useRef<HTMLButtonElement | null>(null);
    const baseId = useId();

    const holdScroll = useCallback(() => {}, []);
    useChatViewport({ scrollToBottom: holdScroll });

    useEffect(() => {
        const vv = window.visualViewport;
        if (!vv) {
            return;
        }

        const onResize = () => {
            setKeyboardOpen(window.innerHeight - vv.height - vv.offsetTop > KEYBOARD_INSET);
        };

        onResize();
        vv.addEventListener("resize", onResize);
        return () => {
            vv.removeEventListener("resize", onResize);
        };
    }, []);

    useEffect(() => {
        if (!menuOpen) {
            return;
        }

        function handleOutside(event: Event) {
            const wrap = menuWrapRef.current;
            if (wrap && !wrap.contains(event.target as Node)) {
                setMenuOpen(false);
            }
        }

        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") {
                event.preventDefault();
                setMenuOpen(false);
                menuButtonRef.current?.focus();
            }
        }

        document.addEventListener("mousedown", handleOutside);
        document.addEventListener("touchstart", handleOutside);
        document.addEventListener("keydown", handleKeyDown);

        return () => {
            document.removeEventListener("mousedown", handleOutside);
            document.removeEventListener("touchstart", handleOutside);
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [menuOpen]);

    function handleDragStart(event: ReactPointerEvent<HTMLDivElement>) {
        const element = stageRef.current;
        const startHeight = stageHeight ?? (element ? element.getBoundingClientRect().height : 0);

        dragRef.current = { startY: event.clientY, startHeight };
        event.currentTarget.setPointerCapture(event.pointerId);
    }

    function handleDragMove(event: ReactPointerEvent<HTMLDivElement>) {
        const drag = dragRef.current;
        if (!drag) {
            return;
        }

        const max = Math.round(window.innerHeight * MAX_STAGE_FRACTION);
        const next = Math.max(MIN_STAGE_HEIGHT, Math.min(max, drag.startHeight + (event.clientY - drag.startY)));

        setStageHeight(next);
    }

    function handleDragEnd(event: ReactPointerEvent<HTMLDivElement>) {
        dragRef.current = null;

        if (event.currentTarget.hasPointerCapture(event.pointerId)) {
            event.currentTarget.releasePointerCapture(event.pointerId);
        }
    }

    function choose(action: () => void) {
        setMenuOpen(false);
        action();
    }

    const tabs: PaneTab[] = [
        { id: "chat", label: "Chat", pane: chat },
        { id: "voice", label: voiceCount > 0 ? `Voice (${voiceCount})` : "Voice", pane: voice },
        { id: "people", label: `People (${watcherCount})`, pane: people },
    ];

    const actions: ActionItem[] = [
        {
            id: "copy",
            label: copied ? "Link copied" : "Copy invite",
            onSelect: onCopyInvite,
            disabled: false,
            destructive: false,
        },
        { id: "hide", label: "Hide", onSelect: onHide, disabled: false, destructive: false },
        { id: "leave", label: "Leave", onSelect: onLeave, disabled: busy, destructive: false },
        ...(canEnd
            ? [{ id: "end", label: "End for everyone", onSelect: onEnd, disabled: busy, destructive: true }]
            : []),
    ];

    const stageStyle =
        stageHeight !== null ? { height: `${stageHeight}px`, aspectRatio: "auto", flexShrink: 0 } : undefined;
    const menuButtonId = `${baseId}-actions`;

    return (
        <div className={`${styles.shell} ${keyboardOpen ? styles.shellKeyboard : ""}`}>
            <div className={styles.stage} ref={stageRef} style={stageStyle}>
                {stage}
            </div>

            <div
                className={styles.dragHandle}
                onPointerDown={handleDragStart}
                onPointerMove={handleDragMove}
                onPointerUp={handleDragEnd}
                onPointerCancel={handleDragEnd}
                role="separator"
                aria-label="Drag to resize the video"
            >
                <span className={styles.dragGrip} />
            </div>

            <div className={styles.meta}>
                <span dir="auto" className={styles.title} title={title}>
                    {title}
                </span>
                {hasControl && (
                    <span className={styles.controlBadge} title="You have control of the VM">
                        Controller
                    </span>
                )}
                <span className={styles.watchers} title="Watching now" aria-label={`${watcherCount} watching`}>
                    <span aria-hidden="true">{"\u{1F441}"}</span>
                    {watcherCount}
                </span>
                {copied && (
                    <span role="status" className={styles.copiedPill}>
                        Link copied
                    </span>
                )}
                <div className={styles.menuWrap} ref={menuWrapRef}>
                    <button
                        type="button"
                        id={menuButtonId}
                        ref={menuButtonRef}
                        className={styles.menuBtn}
                        aria-label="Party actions"
                        aria-haspopup="menu"
                        aria-expanded={menuOpen}
                        onClick={() => setMenuOpen(open => !open)}
                    >
                        <span aria-hidden="true">{"⋯"}</span>
                    </button>
                    {menuOpen && (
                        <div className={styles.menu} role="menu" aria-labelledby={menuButtonId}>
                            {actions.map(action => (
                                <button
                                    key={action.id}
                                    type="button"
                                    role="menuitem"
                                    className={`${styles.menuItem} ${action.destructive ? styles.menuItemDestructive : ""}`}
                                    disabled={action.disabled}
                                    onClick={() => choose(action.onSelect)}
                                >
                                    {action.label}
                                </button>
                            ))}
                        </div>
                    )}
                </div>
            </div>

            <div className={styles.tabs} role="tablist" aria-label="Watch party panels">
                {tabs.map(entry => (
                    <button
                        key={entry.id}
                        type="button"
                        role="tab"
                        id={`${baseId}-tab-${entry.id}`}
                        aria-controls={`${baseId}-pane-${entry.id}`}
                        aria-selected={tab === entry.id}
                        className={`${styles.tab} ${tab === entry.id ? styles.tabActive : ""}`}
                        onClick={() => setTab(entry.id)}
                    >
                        {entry.label}
                    </button>
                ))}
            </div>

            <div className={styles.body}>
                {tabs.map(entry => (
                    <div
                        key={entry.id}
                        role="tabpanel"
                        id={`${baseId}-pane-${entry.id}`}
                        aria-labelledby={`${baseId}-tab-${entry.id}`}
                        hidden={tab !== entry.id}
                        className={tab === entry.id ? styles.pane : styles.paneHidden}
                    >
                        {entry.pane}
                    </div>
                ))}
            </div>
        </div>
    );
}
