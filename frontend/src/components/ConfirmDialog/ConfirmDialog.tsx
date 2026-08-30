import { type ReactNode, useEffect, useId, useRef } from "react";
import { Button } from "../Button/Button";
import { Modal } from "../Modal/Modal";
import styles from "./ConfirmDialog.module.css";

interface ConfirmDialogProps {
    open: boolean;
    title: string;
    body: ReactNode;
    confirmLabel?: string;
    cancelLabel?: string;
    destructive?: boolean;
    busy?: boolean;
    onConfirm: () => void;
    onCancel: () => void;
}

const focusableSelector = 'a[href], button, input, select, textarea, [tabindex]:not([tabindex="-1"])';

function focusableWithin(dialog: HTMLElement): HTMLElement[] {
    return Array.from(dialog.querySelectorAll<HTMLElement>(focusableSelector)).filter(
        el => !el.hasAttribute("disabled") && el.getAttribute("aria-hidden") !== "true",
    );
}

function trapTab(dialog: HTMLElement, e: KeyboardEvent) {
    if (e.key !== "Tab") {
        return;
    }

    const focusable = focusableWithin(dialog);
    if (focusable.length === 0) {
        e.preventDefault();
        return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const active = document.activeElement;
    const outside = !dialog.contains(active);

    if (e.shiftKey && (outside || active === first)) {
        e.preventDefault();
        last.focus();
        return;
    }

    if (!e.shiftKey && (outside || active === last)) {
        e.preventDefault();
        first.focus();
    }
}

export function ConfirmDialog({
    open,
    title,
    body,
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    destructive = false,
    busy = false,
    onConfirm,
    onCancel,
}: ConfirmDialogProps) {
    const actionsRef = useRef<HTMLDivElement>(null);
    const confirmId = useId();
    const cancelId = useId();

    useEffect(() => {
        if (!open) {
            return;
        }

        const dialog = actionsRef.current?.closest<HTMLElement>('[role="dialog"]');
        if (!dialog) {
            return;
        }

        const onKeyDown = (e: KeyboardEvent) => trapTab(dialog, e);

        dialog.addEventListener("keydown", onKeyDown);

        return () => {
            dialog.removeEventListener("keydown", onKeyDown);
        };
    }, [open]);

    useEffect(() => {
        if (!open) {
            return;
        }

        if (busy) {
            actionsRef.current?.focus();
            return;
        }

        document.getElementById(destructive ? cancelId : confirmId)?.focus();
    }, [open, busy, destructive, cancelId, confirmId]);

    function handleCancel() {
        if (busy) {
            return;
        }

        onCancel();
    }

    function handleConfirm() {
        if (busy) {
            return;
        }

        onConfirm();
    }

    return (
        <Modal isOpen={open} onClose={handleCancel} title={title}>
            <div className={styles.body} dir="auto">
                {body}
            </div>
            <div ref={actionsRef} className={styles.actions} aria-busy={busy} tabIndex={-1}>
                <Button id={cancelId} type="button" variant="secondary" onClick={handleCancel} disabled={busy}>
                    {cancelLabel}
                </Button>
                <Button
                    id={confirmId}
                    type="button"
                    variant={destructive ? "danger" : "primary"}
                    onClick={handleConfirm}
                    disabled={busy}
                >
                    {confirmLabel}
                </Button>
            </div>
        </Modal>
    );
}
