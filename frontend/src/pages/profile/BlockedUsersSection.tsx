import { useCallback, useEffect, useState } from "react";
import { getBlockedUsers, unblockUser, type BlockedUserItem } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import styles from "./SettingsPage.module.css";

export function BlockedUsersSection() {
    const [users, setUsers] = useState<BlockedUserItem[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchBlocked = useCallback(async () => {
        try {
            const data = await getBlockedUsers();
            setUsers(data.users);
        } catch {
            setUsers([]);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchBlocked();
    }, [fetchBlocked]);

    async function handleUnblock(id: string) {
        try {
            await unblockUser(id);
            setUsers(prev => prev.filter(u => u.id !== id));
        } catch {
            // ignore
        }
    }

    return (
        <div className={`${styles.section} ${styles.gridFull}`}>
            <h3 className={styles.sectionTitle}>Blocked Users</h3>
            {loading && <p className={styles.mutedText}>Loading...</p>}
            {!loading && users.length === 0 && <p className={styles.mutedText}>You haven't blocked anyone.</p>}
            {!loading && users.length > 0 && (
                <div className={styles.blockedList}>
                    {users.map(u => (
                        <div key={u.id} className={styles.blockedRow}>
                            <ProfileLink
                                user={{
                                    id: u.id,
                                    username: u.username,
                                    display_name: u.display_name,
                                    avatar_url: u.avatar_url,
                                }}
                                size="small"
                            />
                            <Button variant="ghost" size="small" onClick={() => handleUnblock(u.id)}>
                                Unblock
                            </Button>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
