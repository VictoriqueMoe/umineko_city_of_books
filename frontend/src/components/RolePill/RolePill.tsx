import styles from "./RolePill.module.css";

interface RolePillProps {
    role: string;
}

const roleConfig: Record<string, { label: string; className: string }> = {
    admin: { label: "Admin", className: "admin" },
};

export function RolePill({ role }: RolePillProps) {
    const config = roleConfig[role];
    if (!config) {
        return null;
    }
    return <span className={`${styles.pill} ${styles[config.className]}`}>{config.label}</span>;
}
