import { useEffect, useState } from "react";
import { Track } from "livekit-client";
import { VideoTrack, useParticipants, useTracks } from "@livekit/components-react";
import { UPTIME_TICK_MS, streamUptimeLabel } from "../../domain/live/uptime";
import { countViewers } from "../../domain/live/viewers";
import { useViewerRoster } from "../../hooks/useViewerRoster";
import styles from "./live.module.css";

export function ViewerCountReporter({ onChange }: { onChange: (count: number) => void }) {
    const participants = useParticipants();
    const count = countViewers(participants);

    useEffect(() => {
        onChange(count);
    }, [count, onChange]);

    return null;
}

export function StreamUptime({ startedAt }: { startedAt?: string }) {
    const [now, setNow] = useState(() => Date.now());

    useEffect(() => {
        const id = window.setInterval(() => {
            setNow(Date.now());
        }, UPTIME_TICK_MS);
        return () => {
            window.clearInterval(id);
        };
    }, []);

    const label = streamUptimeLabel(startedAt, now);
    if (label === null) {
        return null;
    }

    return (
        <span className={styles.uptime} title="Live for">
            <span className={styles.uptimeDot} />
            {label}
        </span>
    );
}

export function StreamStage() {
    const tracks = useTracks([Track.Source.Camera, Track.Source.ScreenShare, Track.Source.Unknown]);
    const participants = useParticipants();

    const video = tracks.find(t => t.publication?.kind === Track.Kind.Video) ?? null;
    const viewerCount = countViewers(participants);

    return (
        <>
            <div className={styles.viewerCount}>
                {"\u{1F441}"} {viewerCount}
            </div>
            {video ? (
                <VideoTrack trackRef={video} className={styles.video} />
            ) : (
                <div className={styles.offline}>Waiting for the stream to start...</div>
            )}
        </>
    );
}

export function StreamViewers() {
    const participants = useParticipants();
    const roster = useViewerRoster(participants);

    return (
        <div className={styles.viewers}>
            <span className={styles.viewersCount}>
                {"\u{1F441}"} {roster.total} watching
            </span>
            <div className={styles.viewersList}>
                {roster.named.map(viewer => (
                    <span key={viewer.userId} className={styles.viewerChip} title={viewer.name}>
                        {viewer.avatar ? (
                            <img src={viewer.avatar} alt="" className={styles.viewerAvatar} />
                        ) : (
                            <span className={styles.viewerAvatar} />
                        )}
                        <span dir="auto" className={styles.viewerName}>
                            {viewer.name}
                        </span>
                    </span>
                ))}
                {roster.guests > 0 && (
                    <span className={`${styles.viewerChip} ${styles.guestChip}`}>
                        {roster.guests} guest{roster.guests === 1 ? "" : "s"}
                    </span>
                )}
            </div>
        </div>
    );
}
