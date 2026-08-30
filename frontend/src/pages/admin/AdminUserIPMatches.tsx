import { useNavigate } from "react-router";
import { useUserIPMatches } from "../../hooks/queries/admin";
import { Button } from "../../components/Button/Button";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { formatDate } from "../../utils/time";
import styles from "./AdminUserDetail.module.css";

interface AdminUserIPMatchesProps {
    userId: string;
}

export function AdminUserIPMatches({ userId }: AdminUserIPMatchesProps) {
    const navigate = useNavigate();
    const ipMatches = useUserIPMatches(userId, true);

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Other Accounts On This IP</h2>
            {ipMatches.loading ? (
                <span className={styles.emptyState}>Loading...</span>
            ) : ipMatches.failed ? (
                <span className={styles.bannedBadge}>Could not load accounts for this IP address.</span>
            ) : ipMatches.users.length === 0 ? (
                <span className={styles.emptyState}>No other accounts share this IP address.</span>
            ) : (
                <div className={styles.matchList}>
                    {ipMatches.users.map(match => (
                        <div key={match.id} className={styles.matchRow}>
                            <ProfileLink
                                user={{
                                    id: match.id,
                                    username: match.username,
                                    display_name: match.display_name,
                                    avatar_url: match.avatar_url,
                                    role: match.role,
                                }}
                                size="small"
                            />
                            <span className={styles.matchMeta}>
                                {match.banned && <span className={styles.bannedBadge}>Banned</span>}
                                {match.locked && <span className={styles.bannedBadge}>Locked</span>}
                                <span>Joined {formatDate(match.created_at)}</span>
                                <Button
                                    variant="ghost"
                                    size="small"
                                    onClick={() => navigate(`/admin/users/${match.id}`)}
                                >
                                    Manage
                                </Button>
                            </span>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
