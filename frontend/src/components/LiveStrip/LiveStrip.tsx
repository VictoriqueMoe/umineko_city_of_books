import { Link } from "react-router";
import { useHomeActivity } from "../../api/queries/sidebar";
import type { HomeCornerActivity } from "../../types/api";
import { ProfileLink } from "../ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/time";
import styles from "./LiveStrip.module.css";

const MAX_MEMBERS = 3;

interface LiveStripProps {
    corner?: string;
}

const CORNER_ORDER = ["umineko", "higurashi", "ciconia", "higanbana", "roseguns", "general"];

const CORNER_SHORT: Record<string, string> = {
    umineko: "Umi",
    higurashi: "Higu",
    ciconia: "Cico",
    higanbana: "Higan",
    roseguns: "RGD",
    general: "General",
};

const CORNER_PATH: Record<string, string> = {
    umineko: "/game-board/umineko",
    higurashi: "/game-board/higurashi",
    ciconia: "/game-board/ciconia",
    higanbana: "/game-board/higanbana",
    roseguns: "/game-board/roseguns",
    general: "/game-board",
};

function sortCorners(a: HomeCornerActivity, b: HomeCornerActivity): number {
    const ai = CORNER_ORDER.indexOf(a.corner);
    const bi = CORNER_ORDER.indexOf(b.corner);
    if (ai === -1 && bi === -1) {
        return a.corner.localeCompare(b.corner);
    }
    if (ai === -1) {
        return 1;
    }
    if (bi === -1) {
        return -1;
    }
    return ai - bi;
}

interface CornerBreakdownProps {
    corners: HomeCornerActivity[];
}

function CornerBreakdown({ corners }: CornerBreakdownProps) {
    if (corners.length === 0) {
        return <span className={styles.empty}>No new posts in the last 24h across the board.</span>;
    }
    const sorted = [...corners].sort(sortCorners);
    return (
        <div className={styles.corners}>
            <span className={styles.label}>24h:</span>
            {sorted.map(c => (
                <Link key={c.corner} to={CORNER_PATH[c.corner] ?? "/game-board"} className={styles.cornerChip}>
                    {CORNER_SHORT[c.corner] ?? c.corner}
                    <span className={styles.cornerCount}>{c.post_count}</span>
                </Link>
            ))}
        </div>
    );
}

interface CornerFocusProps {
    stats: HomeCornerActivity | undefined;
}

function CornerFocus({ stats }: CornerFocusProps) {
    if (!stats || stats.post_count === 0) {
        return <span className={styles.empty}>No posts here in the last 24h. Be the first.</span>;
    }
    const posters = stats.unique_posters;
    return (
        <div className={styles.focus}>
            <span>
                <strong className={styles.count}>{stats.post_count}</strong>{" "}
                <span className={styles.label}>new post{stats.post_count === 1 ? "" : "s"} today</span>
            </span>
            <span>
                <strong className={styles.count}>{posters}</strong>{" "}
                <span className={styles.label}>poster{posters === 1 ? "" : "s"}</span>
            </span>
            {stats.last_post_at && <span className={styles.label}>last {relativeTime(stats.last_post_at)}</span>}
        </div>
    );
}

export function LiveStrip({ corner }: LiveStripProps) {
    const { data } = useHomeActivity();

    if (!data) {
        return null;
    }

    const members = data.recent_members.slice(0, MAX_MEMBERS);
    const showPerCorner = !corner || corner === "general";
    const cornerStats = data.corner_activity.find(c => c.corner === corner);

    return (
        <div className={styles.strip}>
            <div className={styles.cell}>
                <span className={styles.dot} aria-hidden="true" />
                <span className={styles.count}>{data.online_count}</span>
                <span className={styles.label}>online</span>
            </div>

            <div className={styles.cell}>
                {showPerCorner ? (
                    <CornerBreakdown corners={data.corner_activity} />
                ) : (
                    <CornerFocus stats={cornerStats} />
                )}
            </div>

            {members.length > 0 && (
                <div className={styles.cell}>
                    <span className={styles.label}>New:</span>
                    <div className={styles.members}>
                        {members.map(m => (
                            <ProfileLink
                                key={m.id}
                                user={{
                                    id: m.id,
                                    username: m.username,
                                    display_name: m.display_name,
                                    avatar_url: m.avatar_url,
                                }}
                                size="small"
                                showName={false}
                            />
                        ))}
                    </div>
                </div>
            )}

            <div className={styles.more}>
                <Link to="/welcome#live" className={styles.moreLink}>
                    See full activity &rarr;
                </Link>
            </div>
        </div>
    );
}
