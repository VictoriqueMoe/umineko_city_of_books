import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { RoomAudioRenderer, RoomContext } from "@livekit/components-react";
import { Button } from "../../Button/Button";
import type { SiteRole } from "../../../types/api";
import { siteUrl } from "../../../platform/siteOrigin";
import { errorMessage } from "../../../utils/errorMessage";
import { VoiceParticipantList } from "../Voice/VoiceParticipants";
import type { ActiveWatchPartySession } from "../../../hooks/useWatchParty";
import { ScreenShareView } from "./ScreenShareView";
import { useAudioPlaybackGuard } from "./useAudioPlaybackGuard";
import { useHyperbeamEmbed } from "../../../hooks/useHyperbeamEmbed";
import { useForceMuteWatchPartyVoiceParticipant } from "../../../hooks/mutations/watchParty";
import { FORCE_MUTE_FAILED } from "../../../hooks/mutations/chat";
import { useSessionMedia, type ScreenShareMode } from "../../../hooks/useSessionMedia";
import { RoomChatPanel } from "../RoomChatPanel/RoomChatPanel";
import { WatchPartyParticipants } from "./WatchPartyParticipants";
import styles from "./WatchParty.module.css";

export { FORCE_MUTE_FAILED };

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
}: WatchPartyModalProps) {
    const [busy, setBusy] = useState(false);
    const [copied, setCopied] = useState(false);
    const [shareMode, setShareMode] = useState<ScreenShareMode>("gaming");
    const { session, embedURL, hasControl } = active;

    const { wrapRef, mountError } = useHyperbeamEmbed({ embedURL, isOpen, hasControl, onIdentify });

    const handleCopyInvite = () => {
        const link = siteUrl(`/rooms/${session.room_id}?party=${session.id}`);
        navigator.clipboard
            .writeText(link)
            .then(() => {
                setCopied(true);
                setTimeout(() => setCopied(false), 2000);
            })
            .catch(() => {});
    };

    const isScreenShare = session.type === "screenshare";
    const canModerate = isStarter || viewerIsStaff;
    const media = useSessionMedia({
        roomId: session.room_id,
        sessionId: session.id,
        type: session.type,
        isStarter,
    });

    useAudioPlaybackGuard(media.room);

    const forceMute = useForceMuteWatchPartyVoiceParticipant(session.room_id);

    const forceMuteVoice = (identity: string, muted: boolean) => {
        forceMute.mutate({ sessionId: session.id, userId: identity, muted });
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

    if (!isOpen) {
        return null;
    }

    const handleLeave = async () => {
        setBusy(true);
        try {
            await onLeave();
            onClose();
        } catch {
        } finally {
            setBusy(false);
        }
    };

    const handleEnd = async () => {
        setBusy(true);
        try {
            await onEnd();
            onClose();
        } catch {
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
                        <span dir="auto" className={styles.headerName}>
                            {session.title || "Untitled party"}
                        </span>
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
                            onClick={handleCopyInvite}
                            title="Copy a link that opens this watch party. People who are not in the room will be asked to join it first."
                        >
                            {copied ? "Link copied" : "Copy invite"}
                        </Button>
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
                    <div className={styles.chatPanel}>
                        <RoomChatPanel
                            roomId={session.id}
                            title="Party chat"
                            canSend
                            loginPrompt="to join the party chat."
                        />
                    </div>
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
                            {forceMute.error && (
                                <span className={styles.voiceError} role="alert">
                                    {errorMessage(forceMute.error, FORCE_MUTE_FAILED)}
                                </span>
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
