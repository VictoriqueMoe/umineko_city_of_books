import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import Hyperbeam from "@hyperbeam/web";
import { Button } from "../../Button/Button";
import type { SiteRole } from "../../../utils/permissions";
import type { ActiveWatchPartySession } from "./useWatchParty";
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
    const { session, embedURL, messages, hasControl } = active;

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
                    void identifyRef.current(handle.userId);
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
                    <WatchPartyChat messages={messages} viewerUserId={viewerUserId} onSend={onSendMessage} />
                </div>
                <footer className={styles.footer}>
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
