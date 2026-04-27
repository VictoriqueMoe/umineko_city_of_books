import { useBlockedUsers } from "../../api/queries/user";
import { useUnblockUser } from "../../api/mutations/misc";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { useAuth } from "../../hooks/useAuth";
import styles from "./SettingsPage.module.css";

export function BlockedUsersSection() {
    const { user } = useAuth();
    const { blocked: users, loading } = useBlockedUsers(user?.id ?? "");
    const unblockMutation = useUnblockUser();

    async function handleUnblock(id: string) {
        try {
            await unblockMutation.mutateAsync(id);
        } catch {
            return;
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
