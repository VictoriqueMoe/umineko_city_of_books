import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Room, RoomEvent, Track } from "livekit-client";
import {
    RoomAudioRenderer,
    RoomContext,
    StartAudio,
    VideoTrack,
    useParticipants,
    useTracks,
} from "@livekit/components-react";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useNotifications } from "../../hooks/useNotifications";
import { getStream, getStreamViewerToken, uploadStreamThumbnail, type LiveStream } from "../../api/endpoints";
import type { WSMessage } from "../../types/api";
import { useAuth } from "../../hooks/useAuth";
import { VolumeSlider } from "../../components/VolumeSlider/VolumeSlider";
import { StreamChatPanel } from "./StreamChatPanel";
import styles from "./live.module.css";

export function LiveWatchPage() {
    const { streamID } = useParams<{ streamID: string }>();
    const qc = useQueryClient();
    const { addWSListener } = useNotifications();
    const { user } = useAuth();

    const streamQuery = useQuery({
        queryKey: ["streams", "detail", streamID],
        queryFn: () => getStream(streamID as string),
        enabled: !!streamID,
    });

    const stream = streamQuery.data;
    usePageTitle(stream ? stream.title : "Live");

    const [room, setRoom] = useState<Room | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [volume, setVolume] = useState(1);
    const roomRef = useRef<Room | null>(null);
    const stageRef = useRef<HTMLDivElement>(null);

    const isLive = stream?.status === "live";

    function toggleFullscreen() {
        const el = stageRef.current;
        if (!el) {
            return;
        }
        if (document.fullscreenElement) {
            document.exitFullscreen().catch(() => {});
            return;
        }
        el.requestFullscreen().catch(() => {});
    }

    useEffect(() => {
        document.body.dataset.chatPage = "true";
        return () => {
            delete document.body.dataset.chatPage;
        };
    }, []);

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "stream_offline") {
                const data = msg.data as { streamId: string };
                if (data.streamId === streamID) {
                    qc.invalidateQueries({ queryKey: ["streams", "detail", streamID] });
                }
                return;
            }

            if (msg.type === "stream_live") {
                const data = msg.data as LiveStream;
                if (data.id === streamID) {
                    qc.invalidateQueries({ queryKey: ["streams", "detail", streamID] });
                }
            }
        });
    }, [addWSListener, qc, streamID]);

    useEffect(() => {
        if (!streamID || !isLive) {
            return;
        }

        let aborted = false;
        const lkRoom = new Room();
        roomRef.current = lkRoom;

        lkRoom.on(RoomEvent.Connected, () => {
            if (!aborted) {
                setRoom(lkRoom);
            }
        });
        lkRoom.on(RoomEvent.Disconnected, () => {
            setRoom(prev => (prev === lkRoom ? null : prev));
        });

        getStreamViewerToken(streamID)
            .then(({ token, url }) => {
                if (aborted) {
                    return undefined;
                }

                return lkRoom.connect(url, token);
            })
            .catch(() => {
                if (!aborted) {
                    setError("Could not connect to this stream.");
                }
            });

        return () => {
            aborted = true;
            if (roomRef.current === lkRoom) {
                roomRef.current = null;
            }
            lkRoom.disconnect().catch(() => {});
        };
    }, [streamID, isLive]);

    useEffect(() => {
        if (!user || !isLive || !room || !streamID) {
            return;
        }

        let stopped = false;

        const capture = () => {
            if (stopped || document.visibilityState !== "visible") {
                return;
            }

            const video = stageRef.current?.querySelector<HTMLVideoElement>("video");
            if (!video || video.videoWidth === 0) {
                return;
            }

            const width = 480;
            const height = Math.round((video.videoHeight / video.videoWidth) * width) || 270;
            const canvas = document.createElement("canvas");
            canvas.width = width;
            canvas.height = height;
            const context = canvas.getContext("2d");
            if (!context) {
                return;
            }

            context.drawImage(video, 0, 0, width, height);
            canvas.toBlob(
                blob => {
                    if (blob && !stopped) {
                        uploadStreamThumbnail(streamID, blob).catch(() => {});
                    }
                },
                "image/webp",
                0.7,
            );
        };

        const initial = window.setTimeout(capture, 8000);
        const interval = window.setInterval(capture, 50000);

        return () => {
            stopped = true;
            window.clearTimeout(initial);
            window.clearInterval(interval);
        };
    }, [user, isLive, room, streamID]);

    if (streamQuery.isLoading) {
        return <div className="loading">Loading stream...</div>;
    }

    if (!stream) {
        return (
            <div className={styles.page}>
                <div className="empty-state">Stream not found.</div>
                <Link to="/live">Back to live streams</Link>
            </div>
        );
    }

    const name = stream.streamerDisplayName || stream.streamerUsername;

    return (
        <div className={styles.watchLayout}>
            <div className={styles.watchMain}>
                <div className={styles.stage} ref={stageRef}>
                    {isLive && room ? (
                        <RoomContext.Provider value={room}>
                            <StreamStage />
                            <RoomAudioRenderer volume={volume} />
                            <StartAudio label="Click to enable sound" className={styles.startAudio} />
                            <VolumeSlider
                                value={volume}
                                onChange={setVolume}
                                ariaLabel="Stream volume"
                                className={styles.volumeControl}
                            />
                        </RoomContext.Provider>
                    ) : (
                        <div className={styles.offline}>
                            {error ? error : isLive ? "Connecting..." : "This stream is offline."}
                        </div>
                    )}
                    <button
                        type="button"
                        className={styles.fullscreenBtn}
                        onClick={toggleFullscreen}
                        aria-label="Toggle fullscreen"
                        title="Fullscreen"
                    >
                        {"⛶"}
                    </button>
                </div>

                <div className={styles.watchMeta}>
                    <h1 className={styles.watchTitle}>{stream.title}</h1>
                    <Link to={`/user/${stream.streamerUsername}`} className={styles.watchStreamer}>
                        {stream.streamerAvatarUrl && (
                            <img src={stream.streamerAvatarUrl} alt="" className={styles.cardAvatar} />
                        )}
                        <span>{name}</span>
                    </Link>
                    <Link to="/live" className={styles.backLink}>
                        {"←"} All live streams
                    </Link>
                </div>

                {isLive && room && (
                    <RoomContext.Provider value={room}>
                        <StreamViewers />
                    </RoomContext.Provider>
                )}
            </div>

            <aside className={styles.watchSidebar}>
                <StreamChatPanel streamId={stream.id} isLive={isLive} />
            </aside>
        </div>
    );
}

function StreamStage() {
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

interface ViewerMeta {
    userId?: string;
    username?: string;
    avatarUrl?: string;
}

function StreamViewers() {
    const participants = useParticipants();
    const viewers = participants.filter(p => p.identity.startsWith("viewer_"));

    const named = new Map<string, { name: string; avatar?: string }>();
    let guests = 0;
    for (let i = 0; i < viewers.length; i++) {
        const p = viewers[i];
        let meta: ViewerMeta | null = null;
        if (p.metadata) {
            try {
                meta = JSON.parse(p.metadata) as ViewerMeta;
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
