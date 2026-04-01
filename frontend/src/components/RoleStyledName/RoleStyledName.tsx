import styles from "./RoleStyledName.module.css";

interface RoleStyledNameProps {
    name: string;
    role?: string;
}

const roleClass: Record<string, string> = {
    moderator: "moderator",
    admin: "admin",
    super_admin: "superAdmin",
};

export function RoleStyledName({ name, role }: RoleStyledNameProps) {
    const cls = role ? roleClass[role] : undefined;
    return <span className={`${styles.name}${cls ? ` ${styles[cls]}` : ""}`}>{name}</span>;
}
