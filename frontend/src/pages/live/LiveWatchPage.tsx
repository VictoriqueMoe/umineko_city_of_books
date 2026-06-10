import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Room, RoomEvent, Track } from "livekit-client";
import { RoomAudioRenderer, RoomContext, VideoTrack, useParticipants, useTracks } from "@livekit/components-react";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useNotifications } from "../../hooks/useNotifications";
import { getStream, getStreamViewerToken, type LiveStream } from "../../api/endpoints";
import type { WSMessage } from "../../types/api";
import styles from "./live.module.css";

export function LiveWatchPage() {
    const { streamID } = useParams<{ streamID: string }>();
    const qc = useQueryClient();
    const { addWSListener } = useNotifications();

    const streamQuery = useQuery({
        queryKey: ["streams", "detail", streamID],
        queryFn: () => getStream(streamID as string),
        enabled: !!streamID,
    });

    const stream = streamQuery.data;
    usePageTitle(stream ? stream.title : "Live");

    const [room, setRoom] = useState<Room | null>(null);
    const [error, setError] = useState<string | null>(null);
    const roomRef = useRef<Room | null>(null);

    const isLive = stream?.status === "live";

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
        <div className={styles.watchPage}>
            <div className={styles.stage}>
                {isLive && room ? (
                    <RoomContext.Provider value={room}>
                        <StreamStage />
                        <RoomAudioRenderer />
                    </RoomContext.Provider>
                ) : (
                    <div className={styles.offline}>
                        {error ? error : isLive ? "Connecting..." : "This stream is offline."}
                    </div>
                )}
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
