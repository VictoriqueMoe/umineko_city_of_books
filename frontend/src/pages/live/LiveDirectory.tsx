import { useEffect, useState } from "react";
import { Link } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { listLiveStreams, type LiveStream, type LiveStreamListResponse } from "../../api/endpoints";
import type { WSMessage } from "../../types/api";
import { GoLivePanel } from "../../components/live/GoLivePanel";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import styles from "./live.module.css";

export function LiveDirectory() {
    usePageTitle("Live");
    const qc = useQueryClient();
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const [showGoLive, setShowGoLive] = useState(false);

    const query = useQuery({
        queryKey: ["streams", "live"],
        queryFn: listLiveStreams,
    });

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "stream_live" || msg.type === "stream_offline") {
                qc.invalidateQueries({ queryKey: ["streams", "live"] });
                return;
            }

            if (msg.type === "stream_viewers") {
                const data = msg.data as { streamId: string; viewerCount: number };
                qc.setQueryData<LiveStreamListResponse>(["streams", "live"], prev => {
                    if (!prev) {
                        return prev;
                    }

                    return {
                        ...prev,
                        streams: prev.streams.map(s =>
                            s.id === data.streamId ? { ...s, viewerCount: data.viewerCount } : s,
                        ),
                    };
                });
            }
        });
    }, [addWSListener, qc]);

    const streams = query.data?.streams ?? [];
    const enabled = query.data?.enabled ?? false;

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

            {user && enabled && showGoLive && (
                <GoLivePanel onChanged={() => qc.invalidateQueries({ queryKey: ["streams", "live"] })} />
            )}

            {enabled && query.isLoading && <div className="loading">Loading streams...</div>}

            {enabled && !query.isLoading && streams.length === 0 && (
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
        <Link to={`/live/${stream.id}`} className={styles.card}>
            <div className={styles.cardThumb}>
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
                    <h3 className={styles.cardTitle}>{stream.title}</h3>
                    <p className={styles.cardStreamer}>{name}</p>
                </div>
            </div>
        </Link>
    );
}
