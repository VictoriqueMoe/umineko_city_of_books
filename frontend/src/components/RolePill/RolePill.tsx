import { useSiteInfo } from "../../hooks/useSiteInfo";
import styles from "./RolePill.module.css";

interface RolePillProps {
    role: string;
    userId?: string;
}

const roleConfig: Record<string, { label: string; className: string }> = {
    super_admin: { label: "Reality Author", className: "superAdmin" },
    admin: { label: "Voyager Witch", className: "admin" },
    moderator: { label: "Witch", className: "moderator" },
};

export function RolePill({ role, userId }: RolePillProps) {
    const siteInfo = useSiteInfo();
    const config = roleConfig[role];
    const isTopDetective = userId && siteInfo.top_detective_ids?.includes(userId);
    const isTopGM = userId && siteInfo.top_gm_ids?.includes(userId);

    return (
        <>
            {config && <span className={`${styles.pill} ${styles[config.className]}`}>{config.label}</span>}
            {isTopDetective && (
                <span className={`${styles.pill} ${styles.topDetective}`} title="Ranked #1 in mysteries">
                    True Detective
                </span>
            )}
            {isTopGM && (
                <span className={`${styles.pill} ${styles.topGM}`} title="Top ranked Game Master">
                    Game Master
                </span>
            )}
        </>
    );
}
