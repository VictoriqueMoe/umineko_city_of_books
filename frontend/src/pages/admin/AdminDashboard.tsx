import { useAdminStats } from "../../api/queries/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import styles from "./AdminDashboard.module.css";

export function AdminDashboard() {
    usePageTitle("Admin");
    const { stats, loading } = useAdminStats();

    if (loading) {
        return <div className={styles.loading}>Loading statistics...</div>;
    }

    if (!stats) {
        return null;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Dashboard</h1>

            <div className={styles.statCards}>
                <div className={styles.statCard}>
                    <div className={styles.statLabel}>Total Users</div>
                    <div className={styles.statValue}>{stats.total_users.toLocaleString()}</div>
                </div>
                <div className={styles.statCard}>
                    <div className={styles.statLabel}>Total Theories</div>
                    <div className={styles.statValue}>{stats.total_theories.toLocaleString()}</div>
                </div>
                <div className={styles.statCard}>
                    <div className={styles.statLabel}>Total Responses</div>
                    <div className={styles.statValue}>{stats.total_responses.toLocaleString()}</div>
                </div>
                <div className={styles.statCard}>
                    <div className={styles.statLabel}>Total Votes</div>
                    <div className={styles.statValue}>{stats.total_votes.toLocaleString()}</div>
                </div>
                <div className={styles.statCard}>
                    <div className={styles.statLabel}>Total Posts</div>
                    <div className={styles.statValue}>{stats.total_posts.toLocaleString()}</div>
                </div>
                <div className={styles.statCard}>
                    <div className={styles.statLabel}>Total Comments</div>
                    <div className={styles.statValue}>{stats.total_comments.toLocaleString()}</div>
                </div>
            </div>

            {stats.posts_by_corner && Object.keys(stats.posts_by_corner).length > 0 && (
                <>
                    <h2 className={styles.sectionTitle}>Posts by Corner</h2>
                    <div className={styles.statCards}>
                        {Object.entries(stats.posts_by_corner).map(([corner, count]) => (
                            <div key={corner} className={styles.statCard}>
                                <div className={styles.statLabel}>{corner}</div>
                                <div className={styles.statValue}>{count.toLocaleString()}</div>
                            </div>
                        ))}
                    </div>
                </>
            )}

            <h2 className={styles.sectionTitle}>Activity Overview</h2>
            <table className={styles.table}>
                <thead>
                    <tr>
                        <th>Period</th>
                        <th>New Users</th>
                        <th>New Theories</th>
                        <th>New Responses</th>
                        <th>New Posts</th>
                    </tr>
                </thead>
                <tbody>
                    <tr>
                        <td>Last 24 hours</td>
                        <td>{stats.new_users_24h}</td>
                        <td>{stats.new_theories_24h}</td>
                        <td>{stats.new_responses_24h}</td>
                        <td>{stats.new_posts_24h}</td>
                    </tr>
                    <tr>
                        <td>Last 7 days</td>
                        <td>{stats.new_users_7d}</td>
                        <td>{stats.new_theories_7d}</td>
                        <td>{stats.new_responses_7d}</td>
                        <td>{stats.new_posts_7d}</td>
                    </tr>
                    <tr>
                        <td>Last 30 days</td>
                        <td>{stats.new_users_30d}</td>
                        <td>{stats.new_theories_30d}</td>
                        <td>{stats.new_responses_30d}</td>
                        <td>{stats.new_posts_30d}</td>
                    </tr>
                </tbody>
            </table>

            <h2 className={styles.sectionTitle}>Most Active Users</h2>
            <div className={styles.activeUsersCard}>
                {stats.most_active_users.map(u => (
                    <div key={u.id} className={styles.activeUserRow}>
                        <div className={styles.activeUserInfo}>
                            {u.avatar_url ? (
                                <img className={styles.avatar} src={u.avatar_url} alt="" />
                            ) : (
                                <span className={styles.avatarPlaceholder}>{u.display_name[0]}</span>
                            )}
                            <span>{u.display_name}</span>
                        </div>
                        <span className={styles.actionCount}>{u.action_count} actions</span>
                    </div>
                ))}
                {stats.most_active_users.length === 0 && <div className={styles.loading}>No active users yet</div>}
            </div>
        </div>
    );
}
