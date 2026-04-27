import { Link } from "react-router";
import { useSecretList } from "../../api/queries/secret";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { parseServerDate } from "../../utils/time";
import styles from "./SecretsListPage.module.css";

function formatSolveDate(dateStr: string): string {
    const d = parseServerDate(dateStr);
    if (!d) {
        return "";
    }
    return d.toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

export function SecretsListPage() {
    usePageTitle("Secrets");
    const { user } = useAuth();
    const { data, loading } = useSecretList();
    const secrets = data?.secrets ?? [];
    const solvers = data?.solvers_leaderboard ?? [];

    return (
        <div className={styles.page}>
            <h1 className={styles.heading}>Secrets</h1>
            <p className={styles.intro}>
                Quiet things are scattered across this site. Some of them are waiting to be found. Pick a hunt and see
                how close the others are.
            </p>

            {loading && <div className="loading">Consulting the game board...</div>}

            {!loading && secrets.length === 0 && <div className={styles.empty}>No secrets are awake yet.</div>}

            {!loading && secrets.length > 0 && (
                <div className={styles.list}>
                    {secrets.map(s => (
                        <Link key={s.id} to={`/secrets/${s.id}`} className={styles.card}>
                            <div className={styles.cardHeader}>
                                <h2 className={styles.cardTitle}>{s.title}</h2>
                                <span
                                    className={`${styles.status} ${s.solved ? styles.statusSolved : styles.statusOpen}`}
                                >
                                    {s.solved ? "Solved" : "Open"}
                                </span>
                            </div>
                            <p className={styles.description}>{s.description}</p>
                            <div className={styles.meta}>
                                {user && s.viewer_progress > 0 && (
                                    <span className={styles.progress}>
                                        {s.viewer_progress} / {s.total_pieces} pieces
                                    </span>
                                )}
                                {s.solved && s.solver && (
                                    <span className={styles.solverRow}>
                                        Solved by <ProfileLink user={s.solver} size="small" clickable={false} />
                                    </span>
                                )}
                                <span>
                                    {s.comment_count} comment{s.comment_count === 1 ? "" : "s"}
                                </span>
                            </div>
                        </Link>
                    ))}
                </div>
            )}

            {!loading && (
                <>
                    <h2 className={styles.boardHeading}>Solvers</h2>
                    <div className={styles.board}>
                        {solvers.length === 0 && (
                            <div className={styles.boardEmpty}>No one has solved a secret yet.</div>
                        )}
                        {solvers.map((s, i) => (
                            <div key={s.user.id} className={styles.boardRow}>
                                <span className={`${styles.boardRank}${i < 3 ? ` ${styles.boardRankTop}` : ""}`}>
                                    {i + 1}
                                </span>
                                <ProfileLink user={s.user} size="small" />
                                <span className={styles.boardCount}>{s.solved_count} solved</span>
                                <span className={styles.boardDate}>{formatSolveDate(s.last_solved_at)}</span>
                            </div>
                        ))}
                    </div>
                </>
            )}
        </div>
    );
}
