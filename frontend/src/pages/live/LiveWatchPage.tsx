import { useRef, useState } from "react";
import { Link, useParams } from "react-router";
import { RoomAudioRenderer, RoomContext, StartAudio } from "@livekit/components-react";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useIsMobile } from "../../hooks/useIsMobile";
import { useLiveStream } from "../../hooks/useLiveStream";
import { useStreamChatPopout } from "../../hooks/useStreamPopout";
import { useStreamThumbnailCapture } from "../../hooks/useStreamThumbnailCapture";
import { VolumeSlider } from "../../components/VolumeSlider/VolumeSlider";
import { StreamChatPanel } from "./StreamChatPanel";
import { StreamStage, StreamUptime, StreamViewers } from "./streamParts";
import { MobileLiveView } from "./MobileLiveView";
import { HLSVideoPlayer } from "../../components/live/HLSVideoPlayer";
import styles from "./live.module.css";

export function LiveWatchPage() {
    const { streamID } = useParams<{ streamID: string }>();
    const isMobile = useIsMobile();

    const { stream, loading, plan, room, error, showOwnPreview, setShowOwnPreview, setMode } = useLiveStream(streamID);
    const { isLive, mode, isOwnStream } = plan;

    usePageTitle(stream ? stream.title : "Live");

    const [volume, setVolume] = useState(1);
    const stageRef = useRef<HTMLDivElement>(null);

    const thumbnails = useStreamThumbnailCapture({
        streamId: streamID,
        isOwnStream,
        isLive,
        room,
        stageRef,
    });

    const chat = useStreamChatPopout(streamID);

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

    if (loading) {
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
                mode={mode}
                onModeChange={setMode}
                isOwnStream={isOwnStream}
                showOwnPreview={showOwnPreview}
                onToggleOwnPreview={setShowOwnPreview}
                thumbnailError={thumbnails.lastError}
            />
        );
    }

    return (
        <div className={styles.watchLayout}>
            <div className={styles.watchMain}>
                <div className={styles.stage} ref={stageRef}>
                    {!isLive ? (
                        <div className={styles.offline}>{error ? error : "This stream is offline."}</div>
                    ) : isOwnStream && !showOwnPreview ? (
                        <div className={styles.offline}>
                            <p>
                                This is your own stream, so the preview is hidden to avoid downloading your own video.
                            </p>
                            <button type="button" className={styles.modeBtn} onClick={() => setShowOwnPreview(true)}>
                                Show preview (muted)
                            </button>
                        </div>
                    ) : mode === "hls" && stream.hlsUrl ? (
                        <HLSVideoPlayer src={stream.hlsUrl} className={styles.video} muted={isOwnStream} />
                    ) : room ? (
                        <RoomContext.Provider value={room}>
                            <StreamStage />
                            <RoomAudioRenderer volume={isOwnStream ? 0 : volume} />
                            {!isOwnStream && (
                                <>
                                    <StartAudio label="Click to enable sound" className={styles.startAudio} />
                                    <VolumeSlider
                                        value={volume}
                                        onChange={setVolume}
                                        ariaLabel="Stream volume"
                                        className={styles.volumeControl}
                                    />
                                </>
                            )}
                        </RoomContext.Provider>
                    ) : (
                        <div className={styles.offline}>{error ? error : "Connecting..."}</div>
                    )}
                    {isLive && isOwnStream && showOwnPreview && (
                        <button type="button" className={styles.previewToggle} onClick={() => setShowOwnPreview(false)}>
                            Hide preview
                        </button>
                    )}
                    {isLive && stream.hlsUrl && (
                        <div className={styles.modeToggle}>
                            <button
                                type="button"
                                className={mode === "webrtc" ? styles.modeBtnActive : styles.modeBtn}
                                onClick={() => setMode("webrtc")}
                                title="Sub-second latency, may stutter on a weak connection"
                            >
                                Low latency
                            </button>
                            <button
                                type="button"
                                className={mode === "hls" ? styles.modeBtnActive : styles.modeBtn}
                                onClick={() => setMode("hls")}
                                title="A few seconds behind, but smooth"
                            >
                                Smooth
                            </button>
                        </div>
                    )}
                    {isLive && <StreamUptime startedAt={stream.startedAt} />}
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
                    <h1 dir="auto" className={styles.watchTitle}>
                        {stream.title}
                    </h1>
                    <Link to={`/user/${stream.streamerUsername}`} className={styles.watchStreamer}>
                        {stream.streamerAvatarUrl && (
                            <img src={stream.streamerAvatarUrl} alt="" className={styles.cardAvatar} />
                        )}
                        <span dir="auto">{name}</span>
                    </Link>
                    <Link to="/live" className={styles.backLink}>
                        {"←"} All live streams
                    </Link>
                    {thumbnails.lastError && (
                        <p role="status" className={styles.thumbnailWarning}>
                            {thumbnails.lastError}
                        </p>
                    )}
                    {chat.poppedOut && (
                        <button type="button" className={styles.chatRestoreBtn} onClick={chat.bringBack}>
                            Chat is in its own window. Bring it back
                        </button>
                    )}
                </div>

                {isLive && room && (
                    <RoomContext.Provider value={room}>
                        <StreamViewers />
                    </RoomContext.Provider>
                )}
            </div>

            {!chat.poppedOut && (
                <aside className={styles.watchSidebar}>
                    <StreamChatPanel streamId={stream.id} isLive={isLive} onPopOut={chat.popOut} />
                </aside>
            )}
        </div>
    );
}
