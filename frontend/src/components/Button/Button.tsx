import type {ButtonHTMLAttributes} from "react";
import styles from "./Button.module.css";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: "primary" | "secondary" | "danger" | "ghost";
    size?: "small" | "medium";
}

export function Button({ variant = "secondary", size = "medium", className, children, ...rest }: ButtonProps) {
    const classes = [styles.button, styles[variant], styles[size], className].filter(Boolean).join(" ");

    return (
        <button className={classes} {...rest}>
            {children}
        </button>
    );
}
