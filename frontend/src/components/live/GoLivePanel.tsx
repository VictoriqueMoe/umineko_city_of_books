import { useEffect, useState } from "react";
import {
    getMyStream,
    getStreamCredentials,
    resetStreamCredentials,
    startStream,
    stopStream,
    type LiveStream,
    type StreamCredentials,
    type StreamDefaultMode,
    type StreamOwner,
} from "../../api/endpoints";
import { useNotifications } from "../../hooks/useNotifications";
import type { WSMessage } from "../../types/api";
import { Button } from "../Button/Button";
import { Input } from "../Input/Input";
import styles from "./GoLivePanel.module.css";

const STREAM_RESOLUTIONS = [
    { label: "720p", pixels: 1280 * 720 },
    { label: "1080p", pixels: 1920 * 1080 },
    { label: "1440p", pixels: 2560 * 1440 },
    { label: "4K", pixels: 3840 * 2160 },
];

const FPS_OPTIONS = [30, 60];

interface GoLivePanelProps {
    onChanged?: () => void;
}

export function GoLivePanel({ onChanged }: GoLivePanelProps) {
    const { addWSListener } = useNotifications();
    const [owner, setOwner] = useState<StreamOwner | null>(null);
    const [creds, setCreds] = useState<StreamCredentials | null>(null);
    const [credsError, setCredsError] = useState(false);
    const [title, setTitle] = useState("");
    const [defaultMode, setDefaultMode] = useState<StreamDefaultMode>("webrtc");
    const [resIdx, setResIdx] = useState(1);
    const [calcFps, setCalcFps] = useState(60);
    const [busy, setBusy] = useState(false);
    const [resetting, setResetting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [copied, setCopied] = useState<string | null>(null);
    const [setupOpen, setSetupOpen] = useState(false);

    useEffect(() => {
        getMyStream()
            .then(setOwner)
            .catch(() => {});
        getStreamCredentials()
            .then(setCreds)
            .catch(() => setCredsError(true));
    }, []);

    const ownerStreamId = owner?.stream.id;

    useEffect(() => {
        if (!ownerStreamId) {
            return;
        }
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "stream_live") {
                const data = msg.data as LiveStream;
                if (data.id === ownerStreamId) {
                    setOwner(prev => (prev ? { ...prev, stream: { ...prev.stream, status: "live" } } : prev));
                }
                return;
            }
            if (msg.type === "stream_offline") {
                const data = msg.data as { streamId: string };
                if (data.streamId === ownerStreamId) {
                    setOwner(null);
                    setTitle("");
                    onChanged?.();
                }
            }
        });
    }, [ownerStreamId, addWSListener, onChanged]);

    async function handleStart() {
        const trimmed = title.trim();
        if (!trimmed) {
            return;
        }

        setBusy(true);
        setError(null);

        try {
            const result = await startStream(trimmed, defaultMode);
            setOwner(result);
            setCreds({ whipUrl: result.whipUrl, streamKey: result.streamKey });
            onChanged?.();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Could not start the stream.");
        } finally {
            setBusy(false);
        }
    }

    async function handleStop() {
        if (!owner) {
            return;
        }

        setBusy(true);
        setError(null);

        try {
            await stopStream(owner.stream.id);
            setOwner(null);
            setTitle("");
            onChanged?.();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Could not stop the stream.");
        } finally {
            setBusy(false);
        }
    }

    async function handleReset() {
        if (owner) {
            return;
        }
        if (
            !window.confirm("Reset your stream key? You'll need to paste the new key into OBS before your next stream.")
        ) {
            return;
        }

        setResetting(true);
        setError(null);

        try {
            const next = await resetStreamCredentials();
            setCreds(next);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Could not reset your stream key.");
        } finally {
            setResetting(false);
        }
    }

    function copy(label: string, value: string) {
        navigator.clipboard
            .writeText(value)
            .then(() => setCopied(label))
            .catch(() => {});
    }

    const pixelsPerSecond = STREAM_RESOLUTIONS[resIdx].pixels * calcFps;
    const kbpsAt = (bpp: number) => Math.round((pixelsPerSecond * bpp) / 1000 / 500) * 500;
    const lowKbps = kbpsAt(0.07);
    const highKbps = kbpsAt(0.12);
    const typicalKbps = kbpsAt(0.095);

    return (
        <div className={styles.panel}>
            {owner ? (
                owner.stream.status === "live" ? (
                    <>
                        <h2 className={styles.heading}>You're live</h2>
                        <p className={styles.hint}>{owner.stream.title} is live. Stop here or close OBS to end it.</p>
                        <div className={styles.actions}>
                            <Button variant="danger" onClick={() => handleStop()} disabled={busy}>
                                {busy ? "Stopping..." : "Stop streaming"}
                            </Button>
                        </div>
                    </>
                ) : (
                    <>
                        <h2 className={styles.heading}>Going live...</h2>
                        <p className={styles.hint}>
                            Now press <strong>Start Streaming</strong> in OBS to appear, this takes a few seconds. If
                            OBS was already streaming, stop and start it again so it connects to this stream.
                        </p>
                        <div className={styles.actions}>
                            <Button variant="danger" onClick={() => handleStop()} disabled={busy}>
                                {busy ? "Cancelling..." : "Cancel"}
                            </Button>
                        </div>
                    </>
                )
            ) : (
                <>
                    <h2 className={styles.heading}>Go live</h2>
                    <p className={styles.hint}>
                        Give your stream a title and press Go live, then hit Start Streaming in OBS.
                    </p>
                    <Input
                        type="text"
                        placeholder="Stream title"
                        value={title}
                        onChange={e => setTitle(e.target.value)}
                        maxLength={120}
                        fullWidth
                    />
                    <div className={styles.field}>
                        <span className={styles.fieldLabel}>Default playback for viewers</span>
                        <div className={styles.actions}>
                            <Button
                                size="small"
                                variant={defaultMode === "webrtc" ? "primary" : "secondary"}
                                onClick={() => setDefaultMode("webrtc")}
                            >
                                Low latency
                            </Button>
                            <Button
                                size="small"
                                variant={defaultMode === "hls" ? "primary" : "secondary"}
                                onClick={() => setDefaultMode("hls")}
                            >
                                Smooth
                            </Button>
                        </div>
                        <span className={styles.resetHint}>
                            Low latency is best for chatting and slower games; Smooth (a few seconds behind) avoids
                            freezes on fast, twitchy content. Viewers can switch either way.
                        </span>
                    </div>
                    <div className={styles.actions}>
                        <Button variant="primary" onClick={() => handleStart()} disabled={busy || !title.trim()}>
                            {busy ? "Starting..." : "Go live"}
                        </Button>
                    </div>
                </>
            )}

            {error && <p className={styles.error}>{error}</p>}

            {creds && (
                <div className={styles.disclosure}>
                    <button
                        type="button"
                        className={styles.disclosureToggle}
                        onClick={() => setSetupOpen(open => !open)}
                        aria-expanded={setupOpen}
                    >
                        <span className={styles.disclosureText}>
                            <span className={styles.disclosureTitle}>OBS streaming setup</span>
                            <span className={styles.disclosureSub}>
                                Server, key, encoder settings, and a bitrate calculator
                            </span>
                        </span>
                        <span className={`${styles.chevron} ${setupOpen ? styles.chevronOpen : ""}`}>{"▾"}</span>
                    </button>

                    {setupOpen && (
                        <div className={styles.disclosureBody}>
                            <section className={styles.subSection}>
                                <h4 className={styles.subHeading}>
                                    <span className={styles.subNum}>1</span> Connect OBS
                                </h4>
                                <ol className={styles.steps}>
                                    <li>
                                        OBS 30+, <strong>Settings {"→"} Stream</strong>, Service = <strong>WHIP</strong>
                                        .
                                    </li>
                                    <li>Paste the server and key below, click OK.</li>
                                </ol>

                                <div className={styles.field}>
                                    <span className={styles.fieldLabel}>WHIP server</span>
                                    <div className={styles.copyRow}>
                                        <code className={styles.code}>{creds.whipUrl}</code>
                                        <Button
                                            size="small"
                                            variant="secondary"
                                            onClick={() => copy("url", creds.whipUrl)}
                                        >
                                            {copied === "url" ? "Copied" : "Copy"}
                                        </Button>
                                    </div>
                                </div>

                                <div className={styles.field}>
                                    <span className={styles.fieldLabel}>Stream key (bearer token)</span>
                                    <div className={styles.copyRow}>
                                        <code className={styles.code}>{creds.streamKey}</code>
                                        <Button
                                            size="small"
                                            variant="secondary"
                                            onClick={() => copy("key", creds.streamKey)}
                                        >
                                            {copied === "key" ? "Copied" : "Copy"}
                                        </Button>
                                    </div>
                                </div>

                                <div className={styles.resetRow}>
                                    <Button
                                        size="small"
                                        variant="ghost"
                                        onClick={() => handleReset()}
                                        disabled={resetting || !!owner}
                                    >
                                        {resetting ? "Resetting..." : "Reset stream key"}
                                    </Button>
                                    <span className={styles.resetHint}>Use this if your key leaks.</span>
                                </div>
                            </section>

                            <section className={styles.subSection}>
                                <h4 className={styles.subHeading}>
                                    <span className={styles.subNum}>2</span> Encoder settings
                                </h4>
                                <table className={styles.encTable}>
                                    <tbody>
                                        <tr>
                                            <th>Video encoder</th>
                                            <td>
                                                H.264 <span className={styles.encNote}>x264 or NVENC</span>
                                            </td>
                                        </tr>
                                        <tr>
                                            <th>B-Frames</th>
                                            <td>0</td>
                                        </tr>
                                        <tr>
                                            <th>Rate control</th>
                                            <td>CBR</td>
                                        </tr>
                                        <tr>
                                            <th>Keyframe interval</th>
                                            <td>2s</td>
                                        </tr>
                                        <tr>
                                            <th>Audio encoder</th>
                                            <td>Opus</td>
                                        </tr>
                                    </tbody>
                                </table>
                                <p className={styles.hint}>
                                    H.264 only, AV1 and HEVC will not connect. B-Frames must be 0 or WebRTC ingest
                                    stutters.
                                </p>
                            </section>

                            <section className={styles.subSection}>
                                <h4 className={styles.subHeading}>
                                    <span className={styles.subNum}>3</span> Bitrate calculator
                                </h4>
                                <div className={styles.calcCard}>
                                    <div className={styles.calcGroup}>
                                        <span className={styles.calcLabel}>Resolution</span>
                                        <div className={styles.pills}>
                                            {STREAM_RESOLUTIONS.map((r, i) => (
                                                <button
                                                    key={r.label}
                                                    type="button"
                                                    className={i === resIdx ? styles.pillActive : styles.pill}
                                                    onClick={() => setResIdx(i)}
                                                >
                                                    {r.label}
                                                </button>
                                            ))}
                                        </div>
                                    </div>
                                    <div className={styles.calcGroup}>
                                        <span className={styles.calcLabel}>Framerate</span>
                                        <div className={styles.pills}>
                                            {FPS_OPTIONS.map(f => (
                                                <button
                                                    key={f}
                                                    type="button"
                                                    className={f === calcFps ? styles.pillActive : styles.pill}
                                                    onClick={() => setCalcFps(f)}
                                                >
                                                    {f} fps
                                                </button>
                                            ))}
                                        </div>
                                    </div>
                                    <div className={styles.calcResult}>
                                        <span className={styles.calcResultMain}>
                                            {typicalKbps.toLocaleString()}
                                            <span className={styles.calcUnit}> Kbps</span>
                                        </span>
                                        <span className={styles.calcResultSub}>
                                            {lowKbps.toLocaleString()} to {highKbps.toLocaleString()} range, set as CBR
                                        </span>
                                    </div>
                                </div>
                            </section>

                            <p className={styles.tip}>
                                <strong>Low latency</strong> is your stream untouched. <strong>Smooth</strong> is
                                re-encoded and a few seconds behind, but never freezes. Viewers choose.
                            </p>
                        </div>
                    )}
                </div>
            )}
            {credsError && <p className={styles.hint}>Could not load your stream key. Reload the page to try again.</p>}
        </div>
    );
}
