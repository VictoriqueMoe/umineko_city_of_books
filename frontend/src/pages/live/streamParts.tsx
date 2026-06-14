import { useEffect } from "react";
import { Track } from "livekit-client";
import { VideoTrack, useParticipants, useTracks } from "@livekit/components-react";
import { absolutizeMedia } from "../../api/client";
import styles from "./live.module.css";

interface ViewerMeta {
    userId?: string;
    username?: string;
    avatarUrl?: string;
}

export function ViewerCountReporter({ onChange }: { onChange: (count: number) => void }) {
    const participants = useParticipants();
    const count = participants.filter(p => p.identity.startsWith("viewer_")).length;

    useEffect(() => {
        onChange(count);
    }, [count, onChange]);

    return null;
}

export function StreamStage() {
    const tracks = useTracks([Track.Source.Camera, Track.Source.ScreenShare, Track.Source.Unknown]);
    const participants = useParticipants();

    const video = tracks.find(t => t.publication?.kind === Track.Kind.Video) ?? null;
    const viewerCount = participants.filter(p => p.identity.startsWith("viewer_")).length;

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
    const viewers = participants.filter(p => p.identity.startsWith("viewer_"));

    const named = new Map<string, { name: string; avatar?: string }>();
    let guests = 0;
    for (let i = 0; i < viewers.length; i++) {
        const p = viewers[i];
        let meta: ViewerMeta | null = null;
        if (p.metadata) {
            try {
                meta = absolutizeMedia(JSON.parse(p.metadata) as ViewerMeta);
            } catch {
                meta = null;
            }
        }
        if (meta?.userId) {
            named.set(meta.userId, { name: p.name || meta.username || "Member", avatar: meta.avatarUrl });
        } else {
            guests += 1;
        }
    }

    const namedList = Array.from(named.entries());

    return (
        <div className={styles.viewers}>
            <span className={styles.viewersCount}>
                {"\u{1F441}"} {viewers.length} watching
            </span>
            <div className={styles.viewersList}>
                {namedList.map(([userId, v]) => (
                    <span key={userId} className={styles.viewerChip} title={v.name}>
                        {v.avatar ? (
                            <img src={v.avatar} alt="" className={styles.viewerAvatar} />
                        ) : (
                            <span className={styles.viewerAvatar} />
                        )}
                        <span className={styles.viewerName}>{v.name}</span>
                    </span>
                ))}
                {guests > 0 && (
                    <span className={`${styles.viewerChip} ${styles.guestChip}`}>
                        {guests} guest{guests === 1 ? "" : "s"}
                    </span>
                )}
            </div>
        </div>
    );
}
