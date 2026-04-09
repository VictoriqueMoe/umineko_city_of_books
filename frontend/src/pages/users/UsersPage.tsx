import { useEffect, useMemo, useState } from "react";
import type { PublicUser } from "../../api/endpoints";
import { listUsersPublic } from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Input } from "../../components/Input/Input";
import styles from "./UsersPage.module.css";

const ROLE_GROUPS = [
    { role: "super_admin", label: "Reality Author" },
    { role: "admin", label: "Voyager Witches" },
    { role: "moderator", label: "Witches" },
];

export function UsersPage() {
    usePageTitle("Users");
    const [users, setUsers] = useState<PublicUser[]>([]);
    const [loading, setLoading] = useState(true);
    const [search, setSearch] = useState("");

    useEffect(() => {
        listUsersPublic()
            .then(setUsers)
            .catch(() => {})
            .finally(() => setLoading(false));
    }, []);

    const filtered = useMemo(() => {
        if (!search.trim()) {
            return users;
        }
        const q = search.toLowerCase();
        return users.filter(u => u.display_name.toLowerCase().includes(q) || u.username.toLowerCase().includes(q));
    }, [users, search]);

    const roleUsers = useMemo(() => {
        const map = new Map<string, PublicUser[]>();
        for (const group of ROLE_GROUPS) {
            map.set(
                group.role,
                filtered.filter(u => u.role === group.role),
            );
        }
        return map;
    }, [filtered]);

    const regularUsers = useMemo(() => {
        const roleSet = new Set(ROLE_GROUPS.map(g => g.role));
        return filtered.filter(u => !u.role || !roleSet.has(u.role));
    }, [filtered]);

    const onlineUsers = useMemo(() => regularUsers.filter(u => u.online), [regularUsers]);
    const offlineUsers = useMemo(() => regularUsers.filter(u => !u.online), [regularUsers]);

    if (loading) {
        return <div className="loading">Consulting the game board...</div>;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Players</h1>
            <Input
                type="text"
                placeholder="Search players..."
                value={search}
                onChange={e => setSearch(e.target.value)}
                className={styles.search}
            />

            {ROLE_GROUPS.map(group => (
                <div key={group.role}>
                    <h2 className={styles.groupTitle}>{group.label}</h2>
                    <div className={styles.userList}>
                        {(roleUsers.get(group.role) ?? []).length === 0 && <span className={styles.empty}>None</span>}
                        {(roleUsers.get(group.role) ?? []).map(u => (
                            <div key={u.id} className={styles.userItem}>
                                <ProfileLink user={u} size="medium" online={u.online} />
                            </div>
                        ))}
                    </div>
                    <hr className={styles.divider} />
                </div>
            ))}

            <h2 className={styles.groupTitle}>
                Online <span className={styles.count}>({onlineUsers.length})</span>
            </h2>
            <div className={styles.userList}>
                {onlineUsers.length === 0 && <span className={styles.empty}>No one online</span>}
                {onlineUsers.map(u => (
                    <div key={u.id} className={styles.userItem}>
                        <ProfileLink user={u} size="medium" online />
                    </div>
                ))}
            </div>
            <hr className={styles.divider} />

            <h2 className={styles.groupTitle}>
                Offline <span className={styles.count}>({offlineUsers.length})</span>
            </h2>
            <div className={styles.userList}>
                {offlineUsers.length === 0 && <span className={styles.empty}>No offline users</span>}
                {offlineUsers.map(u => (
                    <div key={u.id} className={styles.userItem}>
                        <ProfileLink user={u} size="medium" />
                    </div>
                ))}
            </div>
        </div>
    );
}
