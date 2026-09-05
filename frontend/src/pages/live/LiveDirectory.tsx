import { useState } from "react";
import { Link } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { useLiveDirectory } from "../../hooks/useLiveDirectory";
import type { LiveStream } from "../../types/api";
import { GoLivePanel } from "../../components/live/GoLivePanel";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import styles from "./live.module.css";

export function LiveDirectory() {
    usePageTitle("Live");
    const { user } = useAuth();
    const { streams, enabled, loading } = useLiveDirectory();
    const [showGoLive, setShowGoLive] = useState(false);

    return (
        <div className={styles.page}>
            <div className={styles.pageHeader}>
                <h1 className={styles.pageTitle}>Live</h1>
                {user && enabled && (
                    <button className={styles.goLiveBtn} onClick={() => setShowGoLive(prev => !prev)}>
                        {showGoLive ? "Close" : "Go live"}
                    </button>
                )}
            </div>

            <InfoPanel title="What is Live?">
                <p>
                    Live is where members broadcast from <strong>OBS</strong> straight into the site. Anyone can watch,
                    no account needed. Pick a stream below, or go live yourself and stream a playthrough, a reading, or
                    just hang out.
                </p>
            </InfoPanel>

            {!enabled && <div className="empty-state">Live streaming is currently disabled.</div>}

            {user && enabled && showGoLive && <GoLivePanel />}

            {enabled && loading && <div className="loading">Loading streams...</div>}

            {enabled && !loading && streams.length === 0 && (
                <div className="empty-state">No one is live right now. Be the first!</div>
            )}

            <div className={styles.grid}>
                {streams.map(s => (
                    <StreamCard key={s.id} stream={s} />
                ))}
            </div>
        </div>
    );
}

function StreamCard({ stream }: { stream: LiveStream }) {
    const name = stream.streamerDisplayName || stream.streamerUsername;

    return (
        <Link to={`/${stream.streamerUsername}/live`} className={styles.card}>
            <div className={styles.cardThumb}>
                {stream.thumbnailUrl ? (
                    <img src={stream.thumbnailUrl} alt="" className={styles.thumbImg} />
                ) : stream.streamerAvatarUrl ? (
                    <img src={stream.streamerAvatarUrl} alt="" className={styles.thumbAvatar} />
                ) : (
                    <div className={styles.thumbInitial}>{name.slice(0, 1).toUpperCase()}</div>
                )}
                <span className={styles.liveBadge}>LIVE</span>
                <span className={styles.viewerBadge}>
                    {"\u{1F441}"} {stream.viewerCount}
                </span>
            </div>
            <div className={styles.cardBody}>
                {stream.streamerAvatarUrl && (
                    <img src={stream.streamerAvatarUrl} alt="" className={styles.cardAvatar} />
                )}
                <div className={styles.cardText}>
                    <h3 dir="auto" className={styles.cardTitle}>
                        {stream.title}
                    </h3>
                    <p dir="auto" className={styles.cardStreamer}>
                        {name}
                    </p>
                </div>
            </div>
        </Link>
    );
}
