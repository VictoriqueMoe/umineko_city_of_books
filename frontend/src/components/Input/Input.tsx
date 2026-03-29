import type {InputHTMLAttributes} from "react";
import styles from "./Input.module.css";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
    fullWidth?: boolean;
}

export function Input({ fullWidth, className, ...rest }: InputProps) {
    const classes = [styles.input, fullWidth ? styles.fullWidth : "", className].filter(Boolean).join(" ");

    return <input className={classes} {...rest} />;
}
