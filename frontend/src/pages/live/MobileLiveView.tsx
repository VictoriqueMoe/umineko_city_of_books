import { type PointerEvent as ReactPointerEvent, type RefObject, useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { RoomAudioRenderer, RoomContext, StartAudio } from "@livekit/components-react";
import type { Room } from "livekit-client";
import { VolumeSlider } from "../../components/VolumeSlider/VolumeSlider";
import { useChatViewport } from "../../hooks/useChatViewport";
import { type LiveStream } from "../../api/endpoints";
import { StreamChatPanel } from "./StreamChatPanel";
import { StreamStage, StreamUptime, StreamViewers, ViewerCountReporter } from "./streamParts";
import styles from "./live.module.css";

interface MobileLiveViewProps {
    stream: LiveStream;
    room: Room | null;
    isLive: boolean;
    error: string | null;
    volume: number;
    onVolumeChange: (value: number) => void;
    stageRef: RefObject<HTMLDivElement | null>;
    onToggleFullscreen: () => void;
}

const noop = () => {};
const MIN_STAGE_HEIGHT = 96;

export function MobileLiveView({
    stream,
    room,
    isLive,
    error,
    volume,
    onVolumeChange,
    stageRef,
    onToggleFullscreen,
}: MobileLiveViewProps) {
    const [tab, setTab] = useState<"chat" | "viewers">("chat");
    const [keyboardOpen, setKeyboardOpen] = useState(false);
    const [viewerCount, setViewerCount] = useState(0);
    const [stageHeight, setStageHeight] = useState<number | null>(null);
    const dragRef = useRef<{ startY: number; startHeight: number } | null>(null);
    useChatViewport({ scrollToBottom: noop });

    useEffect(() => {
        const vv = window.visualViewport;
        if (!vv) {
            return;
        }
        const onResize = () => {
            setKeyboardOpen(window.innerHeight - vv.height - vv.offsetTop > 120);
        };
        onResize();
        vv.addEventListener("resize", onResize);
        return () => {
            vv.removeEventListener("resize", onResize);
        };
    }, []);

    function handleDragStart(e: ReactPointerEvent<HTMLDivElement>) {
        const stage = stageRef.current;
        const startHeight = stageHeight ?? (stage ? stage.getBoundingClientRect().height : 0);
        dragRef.current = { startY: e.clientY, startHeight };
        e.currentTarget.setPointerCapture(e.pointerId);
    }

    function handleDragMove(e: ReactPointerEvent<HTMLDivElement>) {
        const drag = dragRef.current;
        if (!drag) {
            return;
        }
        const max = Math.round(window.innerHeight * 0.7);
        const next = Math.max(MIN_STAGE_HEIGHT, Math.min(max, drag.startHeight + (e.clientY - drag.startY)));
        setStageHeight(next);
    }

    function handleDragEnd(e: ReactPointerEvent<HTMLDivElement>) {
        dragRef.current = null;
        if (e.currentTarget.hasPointerCapture(e.pointerId)) {
            e.currentTarget.releasePointerCapture(e.pointerId);
        }
    }

    const name = stream.streamerDisplayName || stream.streamerUsername;
    const stageStyle =
        stageHeight !== null ? { height: `${stageHeight}px`, aspectRatio: "auto", flexShrink: 0 } : undefined;

    return (
        <div className={`${styles.mobileShell} ${keyboardOpen ? styles.mobileShellKeyboard : ""}`}>
            <div className={styles.mobileStage} ref={stageRef} style={stageStyle}>
                {isLive && room ? (
                    <RoomContext.Provider value={room}>
                        <StreamStage />
                        <ViewerCountReporter onChange={setViewerCount} />
                        <RoomAudioRenderer volume={volume} />
                        <StartAudio label="Click to enable sound" className={styles.startAudio} />
                        <VolumeSlider
                            value={volume}
                            onChange={onVolumeChange}
                            ariaLabel="Stream volume"
                            className={styles.volumeControl}
                        />
                    </RoomContext.Provider>
                ) : (
                    <div className={styles.offline}>
                        {error ? error : isLive ? "Connecting..." : "This stream is offline."}
                    </div>
                )}
                {isLive && <StreamUptime startedAt={stream.startedAt} />}
                <button
                    type="button"
                    className={styles.fullscreenBtn}
                    onClick={onToggleFullscreen}
                    aria-label="Toggle fullscreen"
                    title="Fullscreen"
                >
                    {"⛶"}
                </button>
            </div>

            <div
                className={styles.mobileDragHandle}
                onPointerDown={handleDragStart}
                onPointerMove={handleDragMove}
                onPointerUp={handleDragEnd}
                onPointerCancel={handleDragEnd}
                role="separator"
                aria-label="Drag to resize the video"
            >
                <span className={styles.mobileDragGrip} />
            </div>

            <div className={styles.mobileMeta}>
                <Link to="/live" className={styles.mobileBackBtn} aria-label="Back to live streams" title="Back">
                    {"←"}
                </Link>
                <div className={styles.mobileMetaText}>
                    <span className={styles.mobileTitle} title={stream.title}>
                        {stream.title}
                    </span>
                    <Link to={`/user/${stream.streamerUsername}`} className={styles.mobileStreamer}>
                        {stream.streamerAvatarUrl && (
                            <img src={stream.streamerAvatarUrl} alt="" className={styles.mobileStreamerAvatar} />
                        )}
                        <span>{name}</span>
                    </Link>
                </div>
                {isLive && (
                    <span className={styles.mobileViewerCount} title="Watching now">
                        {"\u{1F441}"} {viewerCount}
                    </span>
                )}
            </div>

            <div className={styles.mobileTabs}>
                <button
                    type="button"
                    className={`${styles.mobileTab} ${tab === "chat" ? styles.mobileTabActive : ""}`}
                    onClick={() => setTab("chat")}
                >
                    Chat
                </button>
                <button
                    type="button"
                    className={`${styles.mobileTab} ${tab === "viewers" ? styles.mobileTabActive : ""}`}
                    onClick={() => setTab("viewers")}
                >
                    Viewers{isLive ? ` (${viewerCount})` : ""}
                </button>
            </div>

            <div className={styles.mobileBody}>
                <div className={tab === "chat" ? styles.mobilePane : styles.mobilePaneHidden}>
                    <StreamChatPanel streamId={stream.id} isLive={isLive} />
                </div>
                <div className={tab === "viewers" ? styles.mobilePane : styles.mobilePaneHidden}>
                    {isLive && room ? (
                        <RoomContext.Provider value={room}>
                            <div className={styles.mobileViewers}>
                                <StreamViewers />
                            </div>
                        </RoomContext.Provider>
                    ) : (
                        <div className={styles.offline}>No viewers while the stream is offline.</div>
                    )}
                </div>
            </div>
        </div>
    );
}
