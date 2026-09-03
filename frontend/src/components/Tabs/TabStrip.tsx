import { useRef } from "react";
import { tabId, tabPanelId } from "./tabIds";

export interface TabDefinition<T extends string> {
    id: T;
    label: string;
}

interface TabStripProps<T extends string> {
    tabs: TabDefinition<T>[];
    active: T;
    onSelect: (id: T) => void;
    idPrefix: string;
    className?: string;
    tabClassName?: string;
    activeTabClassName?: string;
    ariaLabel: string;
}

export function TabStrip<T extends string>({
    tabs,
    active,
    onSelect,
    idPrefix,
    className,
    tabClassName,
    activeTabClassName,
    ariaLabel,
}: TabStripProps<T>) {
    const stripRef = useRef<HTMLDivElement>(null);

    function focusTab(index: number) {
        const buttons = stripRef.current?.querySelectorAll<HTMLButtonElement>('[role="tab"]');
        buttons?.[index]?.focus();
    }

    function handleKeyDown(e: React.KeyboardEvent<HTMLDivElement>) {
        const current = tabs.findIndex(t => t.id === active);
        if (current < 0) {
            return;
        }

        let next = -1;
        if (e.key === "ArrowRight") {
            next = (current + 1) % tabs.length;
        }
        if (e.key === "ArrowLeft") {
            next = (current - 1 + tabs.length) % tabs.length;
        }
        if (e.key === "Home") {
            next = 0;
        }
        if (e.key === "End") {
            next = tabs.length - 1;
        }

        if (next < 0) {
            return;
        }

        e.preventDefault();
        onSelect(tabs[next].id);
        focusTab(next);
    }

    return (
        <div ref={stripRef} role="tablist" aria-label={ariaLabel} className={className} onKeyDown={handleKeyDown}>
            {tabs.map(tab => {
                const selected = tab.id === active;
                return (
                    <button
                        key={tab.id}
                        type="button"
                        role="tab"
                        id={tabId(idPrefix, tab.id)}
                        aria-selected={selected}
                        aria-controls={tabPanelId(idPrefix, tab.id)}
                        tabIndex={selected ? 0 : -1}
                        className={
                            selected && activeTabClassName ? `${tabClassName} ${activeTabClassName}` : tabClassName
                        }
                        onClick={() => onSelect(tab.id)}
                    >
                        {tab.label}
                    </button>
                );
            })}
        </div>
    );
}
