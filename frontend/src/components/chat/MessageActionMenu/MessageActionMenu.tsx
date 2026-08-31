import { useEffect, useLayoutEffect, useRef, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { createPortal } from "react-dom";
import type { MessageAction, MessageActionId } from "../../../domain/chat/messageActions";
import { placeMenu, type MenuPoint } from "./placeMenu";
import styles from "./MessageActionMenu.module.css";

interface MessageActionMenuProps {
    items: MessageAction[];
    origin: MenuPoint;
    onSelect: (id: MessageActionId, origin: MenuPoint) => void;
    onDismiss: (restoreFocus: boolean) => void;
}

function itemsOf(menu: HTMLElement): HTMLButtonElement[] {
    return Array.from(menu.querySelectorAll<HTMLButtonElement>('[role="menuitem"]'));
}

function step(index: number, delta: number, count: number): number {
    if (index < 0) {
        return delta > 0 ? 0 : count - 1;
    }

    return (index + delta + count) % count;
}

export function MessageActionMenu({ items, origin, onSelect, onDismiss }: MessageActionMenuProps) {
    const menuRef = useRef<HTMLDivElement | null>(null);
    const placedRef = useRef<MenuPoint>(origin);

    useLayoutEffect(() => {
        const menu = menuRef.current;
        if (!menu) {
            return;
        }

        const rect = menu.getBoundingClientRect();
        const rightToLeft = window.getComputedStyle(menu).direction === "rtl";
        const placed = placeMenu(
            origin,
            { width: rect.width, height: rect.height },
            { width: window.innerWidth, height: window.innerHeight },
            rightToLeft,
        );

        placedRef.current = placed;
        menu.style.left = `${placed.x}px`;
        menu.style.top = `${placed.y}px`;
        menu.style.visibility = "visible";

        const first = itemsOf(menu)[0];
        if (first) {
            first.focus();
        }
    }, [origin]);

    useEffect(() => {
        function handleOutside(event: Event) {
            const menu = menuRef.current;
            if (menu && !menu.contains(event.target as Node)) {
                onDismiss(false);
            }
        }

        function handleKeyDown(event: KeyboardEvent) {
            if (event.key === "Escape") {
                event.preventDefault();
                onDismiss(true);
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
    }, [onDismiss]);

    function handleMenuKeyDown(event: ReactKeyboardEvent<HTMLDivElement>) {
        const menu = menuRef.current;
        if (!menu) {
            return;
        }

        const focusable = itemsOf(menu);
        if (focusable.length === 0) {
            return;
        }

        const index = focusable.indexOf(document.activeElement as HTMLButtonElement);

        if (event.key === "ArrowDown") {
            event.preventDefault();
            focusable[step(index, 1, focusable.length)].focus();
            return;
        }
        if (event.key === "ArrowUp") {
            event.preventDefault();
            focusable[step(index, -1, focusable.length)].focus();
            return;
        }
        if (event.key === "Home") {
            event.preventDefault();
            focusable[0].focus();
            return;
        }
        if (event.key === "End") {
            event.preventDefault();
            focusable[focusable.length - 1].focus();
            return;
        }
        if (event.key === "Tab") {
            event.preventDefault();
            onDismiss(true);
        }
    }

    return createPortal(
        <div
            ref={menuRef}
            className={styles.menu}
            role="menu"
            aria-label="Message actions"
            onKeyDown={handleMenuKeyDown}
            onContextMenu={event => event.preventDefault()}
        >
            {items.map(item => (
                <button
                    key={item.id}
                    type="button"
                    role="menuitem"
                    className={`${styles.item} ${item.destructive ? styles.itemDestructive : ""}`}
                    aria-haspopup={item.opensPicker ? "dialog" : undefined}
                    onClick={() => onSelect(item.id, placedRef.current)}
                >
                    <span className={styles.itemIcon} aria-hidden="true">
                        {item.icon}
                    </span>
                    <span className={styles.itemLabel}>{item.label}</span>
                </button>
            ))}
        </div>,
        document.body,
    );
}
