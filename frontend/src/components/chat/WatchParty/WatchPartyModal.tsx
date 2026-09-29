import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { RoomAudioRenderer, RoomContext, useParticipants } from "@livekit/components-react";
import { Button } from "../../Button/Button";
import type { SiteRole } from "../../../types/api";
import { siteUrl } from "../../../platform/siteOrigin";
import { errorMessage } from "../../../utils/errorMessage";
import { VoiceParticipantList } from "../Voice/VoiceParticipants";
import type { ActiveWatchPartySession } from "../../../hooks/useWatchParty";
import { ScreenShareView } from "./ScreenShareView";
import { WatchPartyMobileView } from "./WatchPartyMobileView";
import { useAudioPlaybackGuard } from "./useAudioPlaybackGuard";
import { useHyperbeamEmbed } from "../../../hooks/useHyperbeamEmbed";
import { useIsMobile } from "../../../hooks/useIsMobile";
import { useForceMuteWatchPartyVoiceParticipant } from "../../../hooks/mutations/watchParty";
import { FORCE_MUTE_FAILED } from "../../../hooks/mutations/chat";
import { useSessionMedia, type ScreenShareMode } from "../../../hooks/useSessionMedia";
import { RoomChatPanel } from "../RoomChatPanel/RoomChatPanel";
import { WatchPartyParticipants } from "./WatchPartyParticipants";
import styles from "./WatchParty.module.css";

export { FORCE_MUTE_FAILED };

interface VoiceCountReporterProps {
    onChange: (count: number) => void;
}

function VoiceCountReporter({ onChange }: VoiceCountReporterProps) {
    const participants = useParticipants();
    const count = participants.length;

    useEffect(() => {
        onChange(count);
    }, [count, onChange]);

    return null;
}

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
    const [voiceCount, setVoiceCount] = useState(0);
    const isMobile = useIsMobile();
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

    const mediaRef = useRef<HTMLDivElement | null>(null);
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

    const title = session.title || "Untitled party";
    const canEnd = isStarter || viewerIsStaff;
    const fullscreenLabel = isFullscreen ? "Exit fullscreen" : "Fullscreen";

    const fullscreenControl = (
        <button
            type="button"
            className={isMobile ? `${styles.fullscreenBtn} ${styles.iconBtn}` : styles.fullscreenBtn}
            onClick={toggleFullscreen}
            title={fullscreenLabel}
            aria-label={fullscreenLabel}
        >
            {isMobile ? "⛶" : fullscreenLabel}
        </button>
    );

    const screenBody = media.room ? (
        <RoomContext.Provider value={media.room}>
            <ScreenShareView
                compact={isMobile}
                placeholder={
                    isStarter ? "Click Share screen to start sharing." : "Waiting for the host to share their screen."
                }
                onReload={() => {
                    media.reload().catch(() => {});
                }}
            />
        </RoomContext.Provider>
    ) : (
        <div className={styles.empty}>Connecting...</div>
    );

    const browserBody = (
        <>
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
        </>
    );

    const voiceToggle = media.inVoice ? (
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
    );

    const shareControls =
        isScreenShare && isStarter ? (
            media.isSharing ? (
                <Button
                    variant="ghost"
                    size="small"
                    onClick={() => {
                        media.shareScreen(false, shareMode);
                    }}
                >
                    Stop sharing
                </Button>
            ) : (
                <div className={styles.shareControls}>
                    <div className={styles.shareModeToggle} role="group" aria-label="Stream mode">
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
                            media.shareScreen(true, shareMode);
                        }}
                    >
                        Share screen
                    </Button>
                </div>
            )
        ) : null;

    const audioSink =
        voiceEnabled && media.room ? (
            <RoomContext.Provider value={media.room}>
                <RoomAudioRenderer />
                <VoiceCountReporter onChange={setVoiceCount} />
            </RoomContext.Provider>
        ) : null;

    const voiceRoster =
        voiceEnabled && media.room ? (
            <RoomContext.Provider value={media.room}>
                <VoiceParticipantList canModerate={canModerate} onForceMute={forceMuteVoice} />
            </RoomContext.Provider>
        ) : null;

    const shareAlert = media.shareError ? (
        <span className={styles.voiceError} role="alert">
            {media.shareError}
        </span>
    ) : null;

    const muteAlert = forceMute.error ? (
        <span className={styles.voiceError} role="alert">
            {errorMessage(forceMute.error, FORCE_MUTE_FAILED)}
        </span>
    ) : null;

    if (isMobile) {
        const stage = (
            <>
                {isScreenShare ? (
                    <div className={styles.mobileMedia}>
                        {screenBody}
                        {fullscreenControl}
                    </div>
                ) : (
                    <div className={styles.mobileMedia} ref={wrapRef}>
                        {browserBody}
                    </div>
                )}
                {audioSink}
            </>
        );

        const chat = (
            <RoomChatPanel
                roomId={session.id}
                title="Party chat"
                canSend
                loginPrompt="to join the party chat."
                flush
                hideHeader
            />
        );

        const voice = (
            <div className={styles.mobileVoice}>
                <div className={styles.mobileVoiceActions}>
                    {voiceEnabled && voiceToggle}
                    {shareControls}
                </div>
                {shareAlert}
                {muteAlert}
                {voiceEnabled ? (
                    <div className={styles.voiceRoster}>{voiceRoster}</div>
                ) : (
                    <div className={styles.mobileNotice}>Voice chat is switched off for this site.</div>
                )}
            </div>
        );

        const people = (
            <WatchPartyParticipants
                layout="list"
                participants={session.participants}
                viewerUserId={viewerUserId}
                viewerRole={viewerRole}
                viewerHasControl={hasControl}
                ownerUserId={session.started_by}
                onTransferControl={onTransferControl}
                onKick={onKick}
            />
        );

        return createPortal(
            <div className={`${styles.overlay} ${styles.overlayMobile}`}>
                <div className={`${styles.shell} ${styles.shellMobile}`}>
                    <WatchPartyMobileView
                        title={title}
                        hasControl={hasControl}
                        canEnd={canEnd}
                        busy={busy}
                        copied={copied}
                        watcherCount={session.participants.length}
                        voiceCount={voiceCount}
                        onCopyInvite={handleCopyInvite}
                        onHide={onClose}
                        onLeave={handleLeave}
                        onEnd={handleEnd}
                        stageRef={mediaRef}
                        stage={stage}
                        chat={chat}
                        voice={voice}
                        people={people}
                    />
                </div>
            </div>,
            document.body,
        );
    }

    return createPortal(
        <div className={styles.overlay}>
            <div className={styles.shell}>
                <header className={styles.header}>
                    <div className={styles.headerTitle}>
                        <span className={styles.headerLabel}>Watch party</span>
                        <span dir="auto" className={styles.headerName}>
                            {title}
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
                        {canEnd && (
                            <Button variant="danger" size="small" onClick={handleEnd} disabled={busy}>
                                End for everyone
                            </Button>
                        )}
                    </div>
                </header>
                <div className={styles.body}>
                    {isScreenShare ? (
                        <div className={styles.iframeWrap} ref={mediaRef}>
                            {screenBody}
                            {fullscreenControl}
                        </div>
                    ) : (
                        <div className={styles.iframeWrap} ref={wrapRef}>
                            {browserBody}
                        </div>
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
                                {voiceToggle}
                                {shareControls}
                            </div>
                            {shareAlert}
                            {audioSink}
                            {voiceRoster}
                            {muteAlert}
                        </div>
                    )}
                    {!voiceEnabled && shareControls && (
                        <div className={styles.voiceStrip}>
                            <div className={styles.voiceControls}>{shareControls}</div>
                            {shareAlert}
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
