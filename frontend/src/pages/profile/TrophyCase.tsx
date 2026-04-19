import { useMemo, useState } from "react";
import { useSiteInfo } from "../../hooks/useSiteInfo";
import { useAuth } from "../../hooks/useAuth";
import { useTheme } from "../../hooks/useTheme";
import { EpitaphPanel } from "../../features/easterEgg";
import styles from "./TrophyCase.module.css";

interface TrophyCaseProps {
    profileUserId: string;
    profileSecrets?: string[];
}

export function TrophyCase({ profileUserId, profileSecrets }: TrophyCaseProps) {
    const siteInfo = useSiteInfo();
    const { user } = useAuth();
    const { hasSecret } = useTheme();
    const [epitaphOpen, setEpitaphOpen] = useState(false);

    const isOwner = !!user && user.id === profileUserId;

    const trophies = useMemo(() => {
        const fetched = new Set(profileSecrets ?? []);
        const listed = siteInfo.listed_secrets ?? [];
        const result = [];
        for (const s of listed) {
            const solved = fetched.has(s.id) || (isOwner && hasSecret(s.id));
            if (!solved) {
                continue;
            }
            const role = s.vanity_role_id ? siteInfo.vanity_roles?.find(v => v.id === s.vanity_role_id) : undefined;
            result.push({
                id: s.id,
                title: s.title,
                description: s.description,
                icon: s.icon || "\u2605",
                color: role?.color ?? "#d4a84b",
            });
        }
        return result;
    }, [profileSecrets, siteInfo, isOwner, hasSecret]);

    if (trophies.length === 0) {
        return null;
    }

    function handleClick(id: string) {
        if (!isOwner) {
            return;
        }
        if (id === "witchHunter") {
            setEpitaphOpen(true);
        }
    }

    return (
        <>
            <div className={styles.section}>
                <h2 className={styles.heading}>Achievements</h2>
                <div className={styles.grid}>
                    {trophies.map(t => {
                        const commonProps = {
                            className: styles.trophy,
                            style: {
                                borderColor: t.color,
                                boxShadow: `0 0 16px ${t.color}55`,
                            },
                            title: t.description,
                        };
                        const inner = (
                            <>
                                <span className={styles.icon} style={{ color: t.color }}>
                                    {t.icon}
                                </span>
                                <span className={styles.title}>{t.title}</span>
                            </>
                        );
                        if (isOwner) {
                            return (
                                <button key={t.id} type="button" {...commonProps} onClick={() => handleClick(t.id)}>
                                    {inner}
                                </button>
                            );
                        }
                        return (
                            <div key={t.id} {...commonProps}>
                                {inner}
                            </div>
                        );
                    })}
                </div>
            </div>
            {isOwner && <EpitaphPanel isOpen={epitaphOpen} onClose={() => setEpitaphOpen(false)} />}
        </>
    );
}
