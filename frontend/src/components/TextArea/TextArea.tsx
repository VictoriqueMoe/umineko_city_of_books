import type { TextareaHTMLAttributes } from "react";
import styles from "./TextArea.module.css";

type TextAreaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

export function TextArea({ className, ...rest }: TextAreaProps) {
    const classes = [styles.textarea, className].filter(Boolean).join(" ");

    return <textarea className={classes} {...rest} />;
}
