import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Room, RoomEvent } from "livekit-client";
import { RoomAudioRenderer, RoomContext, StartAudio } from "@livekit/components-react";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useNotifications } from "../../hooks/useNotifications";
import { useIsMobile } from "../../hooks/useIsMobile";
import { getStream, getStreamViewerToken, uploadStreamThumbnail, type LiveStream } from "../../api/endpoints";
import type { WSMessage } from "../../types/api";
import { useAuth } from "../../hooks/useAuth";
import { VolumeSlider } from "../../components/VolumeSlider/VolumeSlider";
import { StreamChatPanel } from "./StreamChatPanel";
import { StreamStage, StreamViewers } from "./streamParts";
import { MobileLiveView } from "./MobileLiveView";
import styles from "./live.module.css";

export function LiveWatchPage() {
    const { streamID } = useParams<{ streamID: string }>();
    const qc = useQueryClient();
    const { addWSListener } = useNotifications();
    const { user } = useAuth();
    const isMobile = useIsMobile();

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

    if (isMobile) {
        return (
            <MobileLiveView
                stream={stream}
                room={room}
                isLive={isLive}
                error={error}
                volume={volume}
                onVolumeChange={setVolume}
                stageRef={stageRef}
                onToggleFullscreen={toggleFullscreen}
            />
        );
    }

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
