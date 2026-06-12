import { useEffect, useState } from "react";
import { getMyStream, startStream, stopStream, type StreamOwner } from "../../api/endpoints";
import { Button } from "../Button/Button";
import { Input } from "../Input/Input";
import styles from "./GoLivePanel.module.css";

interface GoLivePanelProps {
    onChanged?: () => void;
}

export function GoLivePanel({ onChanged }: GoLivePanelProps) {
    const [owner, setOwner] = useState<StreamOwner | null>(null);
    const [title, setTitle] = useState("");
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [copied, setCopied] = useState<string | null>(null);

    useEffect(() => {
        getMyStream()
            .then(setOwner)
            .catch(() => {});
    }, []);

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

    function copy(label: string, value: string) {
        navigator.clipboard
            .writeText(value)
            .then(() => setCopied(label))
            .catch(() => {});
    }

    if (owner) {
        return (
            <div className={styles.panel}>
                <h2 className={styles.heading}>You're set up to go live</h2>
                <p className={styles.hint}>Broadcast into your stream from OBS over WHIP, about a minute to set up:</p>
                <ol className={styles.steps}>
                    <li>
                        Open <strong>OBS Studio</strong> (version 30 or newer). Streamlabs works too, but its WHIP has
                        audio glitches, so OBS is recommended.
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
                        Click <strong>OK</strong>, then <strong>Start Streaming</strong>.
                    </li>
                    <li>You'll appear on the Live page within a few seconds.</li>
                </ol>
                <p className={styles.tip}>
                    Whatever resolution, bitrate, and framerate you set in OBS is exactly what your viewers get, the
                    site never re-encodes your stream.
                </p>

                <div className={styles.field}>
                    <span className={styles.fieldLabel}>WHIP server (Server)</span>
                    <div className={styles.copyRow}>
                        <code className={styles.code}>{owner.whipUrl}</code>
                        <Button size="small" variant="secondary" onClick={() => copy("url", owner.whipUrl)}>
                            {copied === "url" ? "Copied" : "Copy"}
                        </Button>
                    </div>
                </div>

                <div className={styles.field}>
                    <span className={styles.fieldLabel}>Stream key (Bearer Token)</span>
                    <div className={styles.copyRow}>
                        <code className={styles.code}>{owner.streamKey}</code>
                        <Button size="small" variant="secondary" onClick={() => copy("key", owner.streamKey)}>
                            {copied === "key" ? "Copied" : "Copy"}
                        </Button>
                    </div>
                </div>

                <div className={styles.actions}>
                    <Button variant="danger" onClick={() => handleStop()} disabled={busy}>
                        {busy ? "Stopping..." : "Stop streaming"}
                    </Button>
                </div>

                {error && <p className={styles.error}>{error}</p>}
            </div>
        );
    }

    return (
        <div className={styles.panel}>
            <h2 className={styles.heading}>Go live</h2>
            <p className={styles.hint}>Give your stream a title, then broadcast into it from OBS over WHIP.</p>
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
            {error && <p className={styles.error}>{error}</p>}
        </div>
    );
}
