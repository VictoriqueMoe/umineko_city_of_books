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

function hexToRgba(hex: string, alpha: number): string {
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

function darkenForText(hex: string): string {
    const r = parseInt(hex.slice(1, 3), 16) / 255;
    const g = parseInt(hex.slice(3, 5), 16) / 255;
    const b = parseInt(hex.slice(5, 7), 16) / 255;
    const max = Math.max(r, g, b);
    const min = Math.min(r, g, b);
    const l = (max + min) / 2;
    if (l <= 0.42) {
        return hex;
    }
    const scale = 0.42 / l;
    const nr = Math.round(r * scale * 255);
    const ng = Math.round(g * scale * 255);
    const nb = Math.round(b * scale * 255);
    const toHex = (v: number) => v.toString(16).padStart(2, "0");
    return `#${toHex(nr)}${toHex(ng)}${toHex(nb)}`;
}

export function RolePill({ role, userId }: RolePillProps) {
    const siteInfo = useSiteInfo();
    const config = roleConfig[role];

    const userVanityRoleIds = (userId && siteInfo.vanity_role_assignments?.[userId]) ?? [];
    const allVanityRoles = siteInfo.vanity_roles ?? [];
    const vanityRoles = [];
    for (const vr of allVanityRoles) {
        if (userVanityRoleIds.includes(vr.id)) {
            vanityRoles.push(vr);
        }
    }
    vanityRoles.sort((a, b) => a.sort_order - b.sort_order);

    return (
        <>
            {config && <span className={`${styles.pill} ${styles[config.className]}`}>{config.label}</span>}
            {vanityRoles.map(vr => {
                if (vr.id === "system_witch_hunter") {
                    return (
                        <span key={vr.id} className={`${styles.pill} ${styles.witchHunter}`}>
                            <span className={styles.witchHunterLabel}>{vr.label}</span>
                        </span>
                    );
                }
                return (
                    <span
                        key={vr.id}
                        className={styles.pill}
                        style={{
                            backgroundColor: hexToRgba(vr.color, 0.18),
                            color: darkenForText(vr.color),
                            border: `1px solid ${hexToRgba(vr.color, 0.55)}`,
                        }}
                    >
                        {vr.label}
                    </span>
                );
            })}
        </>
    );
}
