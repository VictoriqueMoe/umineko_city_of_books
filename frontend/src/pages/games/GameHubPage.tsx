import { Link, useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useGameScoreboard, useLiveGameRooms } from "../../api/queries/gameRoom";
import { gameTypeFor } from "../../games/registry";
import { Button } from "../../components/Button/Button";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import styles from "./GamesPages.module.css";

export function GameHubPage() {
    const { type } = useParams<{ type: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const def = type ? gameTypeFor(type) : undefined;
    usePageTitle(def ? def.label : "Games");
    const { data: scoreboard, loading: scoreboardLoading } = useGameScoreboard(def?.type);
    const { rooms: liveRooms, loading: liveLoading } = useLiveGameRooms(def?.type);
    const loading = scoreboardLoading || liveLoading;

    if (!def) {
        return (
            <div className={styles.page}>
                <h2 className={styles.heading}>Unknown game</h2>
                <p className={styles.empty}>That game type does not exist.</p>
                <div className={styles.actions}>
                    <Button onClick={() => navigate("/games")}>Back to Games</Button>
                </div>
            </div>
        );
    }

    const scoreboardRows = scoreboard?.rows ?? [];

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>{def.label}</h2>
            <p className={styles.subline}>{def.tagline}</p>

            {def.howToPlay && def.howToPlay.length > 0 && (
                <InfoPanel title={`How to play ${def.label.toLowerCase()}`}>
                    {def.howToPlay.map((line, i) => (
                        <p key={i}>{line}</p>
                    ))}
                </InfoPanel>
            )}

            <div className={styles.actions}>
                {user ? (
                    <Link to={def.newPath}>
                        <Button variant="primary">Start a new {def.label.toLowerCase()} game</Button>
                    </Link>
                ) : (
                    <Link to="/login">
                        <Button variant="primary">Sign in to play</Button>
                    </Link>
                )}
                <Link to="/games/live">
                    <Button variant="ghost">Live games</Button>
                </Link>
                <Link to="/games/past">
                    <Button variant="ghost">Past games</Button>
                </Link>
            </div>

            <h3 className={styles.sectionTitle}>Live now</h3>
            {loading ? (
                <p className={styles.empty}>Loading...</p>
            ) : liveRooms.length === 0 ? (
                <p className={styles.empty}>No {def.label.toLowerCase()} games in progress right now.</p>
            ) : (
                <div className={styles.gameList}>
                    {liveRooms.slice(0, 5).map(r => {
                        const white = r.players.find(p => p.slot === 0);
                        const black = r.players.find(p => p.slot === 1);
                        return (
                            <Link key={r.id} to={def.detailPath(r.id)} className={styles.gameRow}>
                                <div className={styles.gameRowContent}>
                                    <span className={styles.opponentLine}>
                                        {white?.display_name ?? "?"} vs {black?.display_name ?? "?"}
                                    </span>
                                    <span className={styles.subline}>{r.watcher_count} watching</span>
                                </div>
                                <span className={`${styles.statusBadge} ${styles.statusActive}`}>live</span>
                            </Link>
                        );
                    })}
                </div>
            )}

            <h3 className={styles.sectionTitle}>Scoreboard</h3>
            {scoreboardRows.length === 0 ? (
                <p className={styles.empty}>No completed games yet. Be the first to finish a match.</p>
            ) : (
                <table className={styles.scoreboardTable}>
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>Player</th>
                            <th>W</th>
                            <th>L</th>
                            <th>D</th>
                            <th>Games</th>
                            <th>Win rate</th>
                        </tr>
                    </thead>
                    <tbody>
                        {scoreboardRows.map((row, i) => (
                            <tr key={row.user.id}>
                                <td>{i + 1}</td>
                                <td>
                                    <ProfileLink user={row.user} />
                                </td>
                                <td>{row.wins}</td>
                                <td>{row.losses}</td>
                                <td>{row.draws}</td>
                                <td>{row.games_played}</td>
                                <td>{(row.win_rate * 100).toFixed(1)}%</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </div>
    );
}
