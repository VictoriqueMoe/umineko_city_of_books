import type { KeyboardEvent, TextareaHTMLAttributes } from "react";
import { isSubmitShortcut, submitFormLikeEnter } from "../../utils/submitShortcut";
import styles from "./TextArea.module.css";

interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
    onSubmitShortcut?: () => void;
}

export function TextArea({ className, onKeyDown, onSubmitShortcut, ...rest }: TextAreaProps) {
    const classes = [styles.textarea, className].filter(Boolean).join(" ");

    function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
        onKeyDown?.(e);
        if (e.defaultPrevented || !isSubmitShortcut(e.nativeEvent)) {
            return;
        }

        if (onSubmitShortcut) {
            e.preventDefault();
            onSubmitShortcut();
            return;
        }

        const form = e.currentTarget.form;
        if (form) {
            e.preventDefault();
            submitFormLikeEnter(form);
        }
    }

    return <textarea dir="auto" className={classes} onKeyDown={handleKeyDown} {...rest} />;
}
