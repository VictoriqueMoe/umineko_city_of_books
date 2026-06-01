import { useEffect, useRef, useState } from "react";
import { Button } from "../../Button/Button";
import type { WatchPartySession } from "../../../types/api";
import styles from "./WatchParty.module.css";

interface WatchPartyButtonProps {
    enabled: boolean;
    screenShareEnabled: boolean;
    sessions: WatchPartySession[];
    activeSessionId: string | null;
    viewerUserId: string | null;
    onStart: (opts: { title?: string; type?: "hyperbeam" | "screenshare" }) => Promise<unknown>;
    onJoin: (sessionId: string) => Promise<void>;
    onOpenExisting: (sessionId: string) => void;
}

export function WatchPartyButton({
    enabled,
    screenShareEnabled,
    sessions,
    activeSessionId,
    viewerUserId,
    onStart,
    onJoin,
    onOpenExisting,
}: WatchPartyButtonProps) {
    const [open, setOpen] = useState(false);
    const [titleDraft, setTitleDraft] = useState("");
    const [partyType, setPartyType] = useState<"hyperbeam" | "screenshare">(enabled ? "hyperbeam" : "screenshare");
    const [busy, setBusy] = useState(false);
    const popoverRef = useRef<HTMLDivElement | null>(null);

    useEffect(() => {
        if (!open) {
            return;
        }
        const handleDocClick = (e: MouseEvent) => {
            if (!popoverRef.current) {
                return;
            }
            if (!popoverRef.current.contains(e.target as Node)) {
                setOpen(false);
            }
        };
        document.addEventListener("mousedown", handleDocClick);
        return () => document.removeEventListener("mousedown", handleDocClick);
    }, [open]);

    if (!enabled && !screenShareEnabled) {
        return null;
    }

    const showTypeSelector = enabled && screenShareEnabled;
    const activeCount = sessions.length;
    const buttonLabel = activeCount === 0 ? "+ Watch Party" : `Watch Parties (${activeCount})`;

    const handleStart = async () => {
        setBusy(true);
        try {
            await onStart({ title: titleDraft.trim() || undefined, type: partyType });
            setTitleDraft("");
            setOpen(false);
        } finally {
            setBusy(false);
        }
    };

    const handleJoinExisting = async (sessionId: string) => {
        setBusy(true);
        try {
            await onJoin(sessionId);
            setOpen(false);
        } finally {
            setBusy(false);
        }
    };

    const handleOpenAlreadyJoined = (sessionId: string) => {
        onOpenExisting(sessionId);
        setOpen(false);
    };

    return (
        <div className={styles.buttonAnchor} ref={popoverRef}>
            <Button variant="ghost" size="small" onClick={() => setOpen(prev => !prev)} disabled={busy}>
                {buttonLabel}
            </Button>
            {open && (
                <div className={styles.picker}>
                    <div className={styles.pickerHeader}>Watch parties</div>
                    {sessions.length === 0 && <div className={styles.pickerEmpty}>No active parties yet.</div>}
                    {sessions.map(s => {
                        const isViewerActive = s.id === activeSessionId;
                        const viewerIsParticipant =
                            viewerUserId !== null && s.participants.some(p => p.user.id === viewerUserId);
                        return (
                            <div key={s.id} className={styles.pickerRow}>
                                <div className={styles.pickerRowMain}>
                                    <div className={styles.pickerTitle}>{s.title || "Untitled party"}</div>
                                    <div className={styles.pickerMeta}>
                                        {s.participants.length}{" "}
                                        {s.participants.length === 1 ? "participant" : "participants"}
                                    </div>
                                </div>
                                {isViewerActive ? (
                                    <Button
                                        variant="secondary"
                                        size="small"
                                        onClick={() => handleOpenAlreadyJoined(s.id)}
                                    >
                                        Open
                                    </Button>
                                ) : viewerIsParticipant ? (
                                    <Button
                                        variant="secondary"
                                        size="small"
                                        onClick={() => handleJoinExisting(s.id)}
                                        disabled={busy}
                                    >
                                        Resume
                                    </Button>
                                ) : (
                                    <Button
                                        variant="primary"
                                        size="small"
                                        onClick={() => handleJoinExisting(s.id)}
                                        disabled={busy}
                                    >
                                        Join
                                    </Button>
                                )}
                            </div>
                        );
                    })}
                    <div className={styles.pickerDivider} />
                    {showTypeSelector && (
                        <div className={styles.pickerTypes}>
                            <button
                                type="button"
                                className={`${styles.pickerType} ${partyType === "hyperbeam" ? styles.pickerTypeActive : ""}`}
                                onClick={() => setPartyType("hyperbeam")}
                                disabled={busy}
                            >
                                Virtual browser
                            </button>
                            <button
                                type="button"
                                className={`${styles.pickerType} ${partyType === "screenshare" ? styles.pickerTypeActive : ""}`}
                                onClick={() => setPartyType("screenshare")}
                                disabled={busy}
                            >
                                Screen share
                            </button>
                        </div>
                    )}
                    <div className={styles.pickerStart}>
                        <input
                            className={styles.pickerInput}
                            type="text"
                            placeholder="Title (optional)"
                            value={titleDraft}
                            onChange={e => setTitleDraft(e.target.value)}
                            maxLength={80}
                            disabled={busy}
                        />
                        <Button variant="primary" size="small" onClick={handleStart} disabled={busy}>
                            {busy ? "Starting..." : "Start new"}
                        </Button>
                    </div>
                </div>
            )}
        </div>
    );
}
