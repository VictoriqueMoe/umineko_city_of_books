import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import Hyperbeam from "@hyperbeam/web";
import { RoomAudioRenderer, RoomContext } from "@livekit/components-react";
import { Button } from "../../Button/Button";
import { forceMuteWatchPartyVoiceParticipant } from "../../../api/endpoints";
import type { SiteRole } from "../../../utils/permissions";
import { VoiceParticipantList } from "../Voice/VoiceParticipants";
import type { ActiveWatchPartySession } from "./useWatchParty";
import { ScreenShareView } from "./ScreenShareView";
import { useAudioPlaybackGuard } from "./useAudioPlaybackGuard";
import { useSessionMedia, type ScreenShareMode } from "./useSessionMedia";
import { WatchPartyChat } from "./WatchPartyChat";
import { WatchPartyParticipants } from "./WatchPartyParticipants";
import styles from "./WatchParty.module.css";

type HyperbeamHandle = Awaited<ReturnType<typeof Hyperbeam>>;

interface WatchPartyModalProps {
    isOpen: boolean;
    onClose: () => void;
    active: ActiveWatchPartySession;
    viewerUserId: string;
    viewerRole: SiteRole | undefined;
    isStarter: boolean;
    viewerIsStaff: boolean;
    voiceEnabled: boolean;
    onLeave: () => Promise<void>;
    onEnd: () => Promise<void>;
    onTransferControl: (userId: string) => Promise<void>;
    onKick: (userId: string) => Promise<void>;
    onIdentify: (identifier: string) => Promise<void>;
    onSendMessage: (body: string) => Promise<void>;
}

export function WatchPartyModal({
    isOpen,
    onClose,
    active,
    viewerUserId,
    viewerRole,
    isStarter,
    viewerIsStaff,
    voiceEnabled,
    onLeave,
    onEnd,
    onTransferControl,
    onKick,
    onIdentify,
    onSendMessage,
}: WatchPartyModalProps) {
    const wrapRef = useRef<HTMLDivElement | null>(null);
    const handleRef = useRef<HyperbeamHandle | null>(null);
    const identifyRef = useRef(onIdentify);
    const hasControlRef = useRef(false);
    const [busy, setBusy] = useState(false);
    const [mountError, setMountError] = useState<string | null>(null);
    const [shareMode, setShareMode] = useState<ScreenShareMode>("gaming");
    const { session, embedURL, messages, hasControl } = active;

    const isScreenShare = session.type === "screenshare";
    const canModerate = isStarter || viewerIsStaff;
    const media = useSessionMedia({
        roomId: session.room_id,
        sessionId: session.id,
        type: session.type,
        isStarter,
    });

    useAudioPlaybackGuard(media.room);

    const forceMuteVoice = (identity: string, muted: boolean) => {
        forceMuteWatchPartyVoiceParticipant(session.room_id, session.id, identity, muted).catch(() => {});
    };

    const mediaRef = useRef<HTMLElement | null>(null);
    const [isFullscreen, setIsFullscreen] = useState(false);

    useEffect(() => {
        const onFsChange = () => {
            setIsFullscreen(document.fullscreenElement === mediaRef.current);
        };
        document.addEventListener("fullscreenchange", onFsChange);
        return () => document.removeEventListener("fullscreenchange", onFsChange);
    }, []);

    const toggleFullscreen = () => {
        if (document.fullscreenElement) {
            document.exitFullscreen().catch(() => {});
            return;
        }
        mediaRef.current?.requestFullscreen().catch(() => {});
    };

    useEffect(() => {
        identifyRef.current = onIdentify;
    }, [onIdentify]);

    useEffect(() => {
        hasControlRef.current = hasControl;
    }, [hasControl]);

    useEffect(() => {
        if (!isOpen || !embedURL) {
            return;
        }
        const wrap = wrapRef.current;
        if (!wrap) {
            return;
        }
        const container = document.createElement("div");
        container.style.cssText = "position:absolute;inset:0;width:100%;height:100%;";
        wrap.appendChild(container);
        let cancelled = false;
        let activeHandle: HyperbeamHandle | null = null;
        setMountError(null);
        Hyperbeam(container, embedURL, { delegateKeyboard: true, disableInput: !hasControlRef.current })
            .then(handle => {
                if (cancelled) {
                    handle.destroy();
                    return;
                }
                activeHandle = handle;
                handleRef.current = handle;
                if (handle.userId) {
                    identifyRef.current(handle.userId);
                }
            })
            .catch((err: unknown) => {
                console.error("hyperbeam mount failed", err);
                if (cancelled) {
                    return;
                }
                const msg = err instanceof Error ? err.message : String(err);
                setMountError(msg || "Failed to connect to virtual browser");
            });
        return () => {
            cancelled = true;
            if (activeHandle) {
                activeHandle.destroy();
            }
            handleRef.current = null;
            if (container.parentElement) {
                container.parentElement.removeChild(container);
            }
        };
    }, [embedURL, isOpen]);

    useEffect(() => {
        const handle = handleRef.current;
        if (handle) {
            handle.disableInput = !hasControl;
        }
    }, [hasControl]);

    if (!isOpen) {
        return null;
    }

    const handleLeave = async () => {
        setBusy(true);
        try {
            await onLeave();
            onClose();
        } finally {
            setBusy(false);
        }
    };

    const handleEnd = async () => {
        setBusy(true);
        try {
            await onEnd();
            onClose();
        } finally {
            setBusy(false);
        }
    };

    return createPortal(
        <div className={styles.overlay}>
            <div className={styles.shell}>
                <header className={styles.header}>
                    <div className={styles.headerTitle}>
                        <span className={styles.headerLabel}>Watch party</span>
                        <span className={styles.headerName}>{session.title || "Untitled party"}</span>
                    </div>
                    <div className={styles.headerActions}>
                        {hasControl && (
                            <span className={styles.controlBadge} title="You have control of the VM">
                                Controller
                            </span>
                        )}
                        <Button
                            variant="ghost"
                            size="small"
                            onClick={onClose}
                            title="Hide the watch party window. The party keeps running; reopen it from the + Watch Party menu."
                        >
                            Hide
                        </Button>
                        <Button variant="secondary" size="small" onClick={handleLeave} disabled={busy}>
                            Leave
                        </Button>
                        {(isStarter || viewerIsStaff) && (
                            <Button variant="danger" size="small" onClick={handleEnd} disabled={busy}>
                                End for everyone
                            </Button>
                        )}
                    </div>
                </header>
                <div className={styles.body}>
                    {isScreenShare ? (
                        <section className={styles.iframeWrap} ref={mediaRef}>
                            {media.room ? (
                                <RoomContext.Provider value={media.room}>
                                    <ScreenShareView
                                        placeholder={
                                            isStarter
                                                ? "Click Share screen to start sharing."
                                                : "Waiting for the host to share their screen."
                                        }
                                        onReload={() => {
                                            media.reload().catch(() => {});
                                        }}
                                    />
                                </RoomContext.Provider>
                            ) : (
                                <div className={styles.empty}>Connecting...</div>
                            )}
                            <button
                                type="button"
                                className={styles.fullscreenBtn}
                                onClick={toggleFullscreen}
                                title={isFullscreen ? "Exit fullscreen" : "Fullscreen"}
                            >
                                {isFullscreen ? "Exit fullscreen" : "Fullscreen"}
                            </button>
                        </section>
                    ) : (
                        <section className={styles.iframeWrap} ref={wrapRef}>
                            {!embedURL && <div className={styles.empty}>Loading virtual browser...</div>}
                            {mountError && (
                                <div className={styles.mountError}>
                                    <div className={styles.mountErrorTitle}>Virtual browser failed to connect</div>
                                    <div className={styles.mountErrorBody}>{mountError}</div>
                                    <div className={styles.mountErrorHint}>
                                        The VM may have expired. Try ending this party and starting a fresh one.
                                    </div>
                                </div>
                            )}
                        </section>
                    )}
                    <WatchPartyChat messages={messages} viewerUserId={viewerUserId} onSend={onSendMessage} />
                </div>
                <footer className={styles.footer}>
                    {voiceEnabled && (
                        <div className={styles.voiceStrip}>
                            <div className={styles.voiceControls}>
                                <span className={styles.voiceStripLabel}>{"\u{1F50A}"} Voice</span>
                                {media.inVoice ? (
                                    <Button
                                        variant="secondary"
                                        size="small"
                                        onClick={() => {
                                            media.leaveVoice().catch(() => {});
                                        }}
                                    >
                                        Leave voice
                                    </Button>
                                ) : (
                                    <Button
                                        variant="primary"
                                        size="small"
                                        disabled={media.status === "connecting"}
                                        onClick={() => {
                                            media.joinVoice().catch(() => {});
                                        }}
                                    >
                                        Join voice
                                    </Button>
                                )}
                                {isScreenShare &&
                                    isStarter &&
                                    (media.isSharing ? (
                                        <Button
                                            variant="ghost"
                                            size="small"
                                            onClick={() => {
                                                media.shareScreen(false, shareMode).catch(() => {});
                                            }}
                                        >
                                            Stop sharing
                                        </Button>
                                    ) : (
                                        <div className={styles.shareControls}>
                                            <div
                                                className={styles.shareModeToggle}
                                                role="group"
                                                aria-label="Stream mode"
                                            >
                                                <button
                                                    type="button"
                                                    className={`${styles.shareMode} ${shareMode === "gaming" ? styles.shareModeActive : ""}`}
                                                    onClick={() => setShareMode("gaming")}
                                                    title="Smoother video, 1080p 60fps. Best for games and video."
                                                >
                                                    Gaming
                                                </button>
                                                <button
                                                    type="button"
                                                    className={`${styles.shareMode} ${shareMode === "screenshare" ? styles.shareModeActive : ""}`}
                                                    onClick={() => setShareMode("screenshare")}
                                                    title="Clearer text, 1080p 15fps. Best for documents or code."
                                                >
                                                    Screenshare
                                                </button>
                                            </div>
                                            <Button
                                                variant="ghost"
                                                size="small"
                                                onClick={() => {
                                                    media.shareScreen(true, shareMode).catch(() => {});
                                                }}
                                            >
                                                Share screen
                                            </Button>
                                        </div>
                                    ))}
                            </div>
                            {media.room && (
                                <RoomContext.Provider value={media.room}>
                                    <RoomAudioRenderer />
                                    <VoiceParticipantList canModerate={canModerate} onForceMute={forceMuteVoice} />
                                </RoomContext.Provider>
                            )}
                        </div>
                    )}
                    <WatchPartyParticipants
                        participants={session.participants}
                        viewerUserId={viewerUserId}
                        viewerRole={viewerRole}
                        viewerHasControl={hasControl}
                        ownerUserId={session.started_by}
                        onTransferControl={onTransferControl}
                        onKick={onKick}
                    />
                </footer>
            </div>
        </div>,
        document.body,
    );
}
