import { useEffect, useState } from "react";
import {
    getMyStream,
    getStreamCredentials,
    resetStreamCredentials,
    startStream,
    stopStream,
    type LiveStream,
    type StreamCredentials,
    type StreamOwner,
} from "../../api/endpoints";
import { useNotifications } from "../../hooks/useNotifications";
import type { WSMessage } from "../../types/api";
import { Button } from "../Button/Button";
import { Input } from "../Input/Input";
import styles from "./GoLivePanel.module.css";

interface GoLivePanelProps {
    onChanged?: () => void;
}

export function GoLivePanel({ onChanged }: GoLivePanelProps) {
    const { addWSListener } = useNotifications();
    const [owner, setOwner] = useState<StreamOwner | null>(null);
    const [creds, setCreds] = useState<StreamCredentials | null>(null);
    const [credsError, setCredsError] = useState(false);
    const [title, setTitle] = useState("");
    const [busy, setBusy] = useState(false);
    const [resetting, setResetting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [copied, setCopied] = useState<string | null>(null);

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
            const result = await startStream(trimmed);
            setOwner(result);
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
                    <div className={styles.actions}>
                        <Button variant="primary" onClick={() => handleStart()} disabled={busy || !title.trim()}>
                            {busy ? "Starting..." : "Go live"}
                        </Button>
                    </div>
                </>
            )}

            {error && <p className={styles.error}>{error}</p>}

            <div className={styles.setup}>
                <h3 className={styles.setupHeading}>OBS setup (one time)</h3>
                {credsError && (
                    <p className={styles.hint}>Could not load your stream key. Reload the page to try again.</p>
                )}
                {creds && (
                    <>
                        <p className={styles.hint}>
                            Set this up once in OBS and you're ready every time, your server and key stay the same until
                            you reset them.
                        </p>
                        <ol className={styles.steps}>
                            <li>
                                Open <strong>OBS Studio</strong> (version 30 or newer). Streamlabs works too, but its
                                WHIP has audio glitches, so OBS is recommended.
                            </li>
                            <li>
                                In OBS, open <strong>Settings {"→"} Stream</strong>.
                            </li>
                            <li>
                                Set <strong>Service</strong> to <strong>WHIP</strong>.
                            </li>
                            <li>
                                Copy the <strong>WHIP server</strong> below into OBS's <strong>Server</strong> box.
                            </li>
                            <li>
                                Copy the <strong>Stream key</strong> below into OBS's <strong>Bearer Token</strong> box.
                            </li>
                            <li>
                                Click <strong>OK</strong>. From now on, to go live: enter a title above, press
                                <strong> Go live</strong>, then press <strong>Start Streaming</strong> in OBS, no
                                reconfiguring.
                            </li>
                        </ol>

                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>WHIP server (Server)</span>
                            <div className={styles.copyRow}>
                                <code className={styles.code}>{creds.whipUrl}</code>
                                <Button size="small" variant="secondary" onClick={() => copy("url", creds.whipUrl)}>
                                    {copied === "url" ? "Copied" : "Copy"}
                                </Button>
                            </div>
                        </div>

                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Stream key (Bearer Token)</span>
                            <div className={styles.copyRow}>
                                <code className={styles.code}>{creds.streamKey}</code>
                                <Button size="small" variant="secondary" onClick={() => copy("key", creds.streamKey)}>
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
                            <span className={styles.resetHint}>
                                {owner
                                    ? "Stop streaming before resetting your key."
                                    : "Use this if your key leaks, then update OBS with the new one."}
                            </span>
                        </div>

                        <p className={styles.tip}>
                            Whatever resolution, bitrate, and framerate you set in OBS is exactly what your viewers get,
                            the site never re-encodes your stream. Live video runs a few seconds behind real time, which
                            is normal.
                        </p>
                        <p className={styles.tip}>
                            To also stream to a non-WebRTC service (such as Twitch or YouTube) at the same time, run a
                            second instance of OBS set up for that service. On Windows you can start one with{" "}
                            <code className={styles.inlineCode}>obs --multi</code>.
                        </p>
                    </>
                )}
            </div>
        </div>
    );
}
