import styles from "./MinesweeperCell.module.css";
import React from "react";

interface MinesweeperCellProps {
    revealed: boolean;
    flagged: boolean;
    mine: boolean;
    value: number;
    isHitMine?: boolean;
    isPending?: boolean;
    forceShowMine?: boolean;
    hideContent?: boolean;
    onClick?: () => void;
    onRightClick?: () => void;
    disabled?: boolean;
}

const VALUE_CLASS: Record<number, string> = {
    1: styles.v1,
    2: styles.v2,
    3: styles.v3,
    4: styles.v4,
    5: styles.v5,
    6: styles.v6,
    7: styles.v7,
    8: styles.v8,
};

export function MinesweeperCell({
    revealed,
    flagged,
    mine,
    value,
    isHitMine,
    isPending,
    forceShowMine,
    hideContent,
    onClick,
    onRightClick,
    disabled,
}: MinesweeperCellProps) {
    const showAsMine = mine && (revealed || forceShowMine);
    const classes = [styles.cell];
    if (revealed) {
        classes.push(styles.revealed);
        if (!mine && value > 0 && VALUE_CLASS[value]) {
            classes.push(VALUE_CLASS[value]);
        }
    } else if (flagged) {
        classes.push(styles.flagged);
    }
    if (showAsMine) {
        classes.push(styles.mine);
    }
    if (isHitMine) {
        classes.push(styles.hitMine);
    }
    if (isPending && !revealed) {
        classes.push(styles.pending);
    }

    function handleMouseDown(event: React.MouseEvent) {
        if (disabled) {
            return;
        }
        if (event.button === 0 && onClick) {
            onClick();
            return;
        }
        if (event.button === 2 && onRightClick) {
            event.preventDefault();
            onRightClick();
        }
    }

    function handleContextMenu(event: React.MouseEvent) {
        event.preventDefault();
    }

    let content: React.ReactNode = "";
    if (!hideContent) {
        if (showAsMine) {
            content = "✸";
        } else if (revealed && value > 0) {
            content = value;
        } else if (flagged) {
            content = "⚑";
        }
    }

    return (
        <button
            type="button"
            className={classes.join(" ")}
            onMouseDown={handleMouseDown}
            onContextMenu={handleContextMenu}
            disabled={disabled}
            aria-label={revealed ? `cell value ${value}` : flagged ? "flagged" : "hidden"}
        >
            {content}
        </button>
    );
}
