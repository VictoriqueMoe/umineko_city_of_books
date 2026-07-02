import { useEffect, useState } from "react";
import {
    fetchOverlayConnectorSEF,
    getOverlayConnection,
    type OverlayConnection,
    resetOverlayToken,
    testOverlay,
} from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import settings from "./SettingsPage.module.css";
import styles from "./StreamOverlaySection.module.css";

export function StreamOverlaySection() {
    const [conn, setConn] = useState<OverlayConnection | null>(null);
    const [error, setError] = useState("");
    const [success, setSuccess] = useState("");
    const [downloading, setDownloading] = useState(false);
    const [resetting, setResetting] = useState(false);
    const [testing, setTesting] = useState(false);
    const [copied, setCopied] = useState(false);
    const [setupOpen, setSetupOpen] = useState(false);

    useEffect(() => {
        getOverlayConnection()
            .then(setConn)
            .catch(() => setError("Could not load your overlay connection."));
    }, []);

    async function handleDownload() {
        setDownloading(true);
        setError("");
        try {
            const sef = await fetchOverlayConnectorSEF();
            const blob = new Blob([sef], { type: "text/plain" });
            const url = URL.createObjectURL(blob);
            const link = document.createElement("a");
            link.href = url;
            link.download = "overlay-connector.sef";
            document.body.appendChild(link);
            link.click();
            link.remove();
            URL.revokeObjectURL(url);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Could not download the connector.");
        } finally {
            setDownloading(false);
        }
    }

    async function handleReset() {
        const confirmed = window.confirm(
            "Reset your overlay token? Your current connector file will stop working and you'll need to download and re-import the new one.",
        );
        if (!confirmed) {
            return;
        }

        setResetting(true);
        setError("");
        setSuccess("");
        try {
            const next = await resetOverlayToken();
            setConn(next);
            setSuccess("Token reset. Download the new connector below.");
        } catch (err) {
            setError(err instanceof Error ? err.message : "Could not reset your token.");
        } finally {
            setResetting(false);
        }
    }

    async function handleTest() {
        setTesting(true);
        setError("");
        setSuccess("");
        try {
            await testOverlay();
            setSuccess("Test overlay sent. Check your SAMMI overlay.");
            setConn(prev => (prev ? { ...prev, connected: true } : prev));
        } catch (err) {
            setError(
                err instanceof Error
                    ? err.message
                    : "Could not send a test overlay. Make sure SAMMI is open and connected.",
            );
            setConn(prev => (prev ? { ...prev, connected: false } : prev));
        } finally {
            setTesting(false);
        }
    }

    function copyToken() {
        if (!conn) {
            return;
        }
        navigator.clipboard
            .writeText(conn.token)
            .then(() => setCopied(true))
            .catch(() => {});
    }

    return (
        <div className={settings.section}>
            <h3 className={settings.sectionTitle}>Stream Overlay</h3>
            <p className={styles.intro}>
                Fire overlay popups on your stream from site events (likes, follows, comments, theory votes) using
                SAMMI. Download the connector, import it into SAMMI, and your events appear live on stream.
            </p>

            {error && <div className={settings.error}>{error}</div>}
            {success && <div className={settings.success}>{success}</div>}

            {conn && (
                <>
                    <div className={styles.statusRow}>
                        <span className={conn.connected ? styles.statusOn : styles.statusOff}>
                            {conn.connected ? "SAMMI connected" : "SAMMI not connected"}
                        </span>
                    </div>

                    <div className={styles.actions}>
                        <Button variant="primary" onClick={() => handleDownload()} disabled={downloading}>
                            {downloading ? "Preparing..." : "Download SAMMI connector (.sef)"}
                        </Button>
                        <Button variant="secondary" onClick={() => handleTest()} disabled={testing}>
                            {testing ? "Sending..." : "Send test overlay"}
                        </Button>
                    </div>

                    <label className={settings.label}>
                        Connection token
                        <div className={styles.copyRow}>
                            <code className={styles.code}>{conn.token}</code>
                            <Button size="small" variant="secondary" onClick={() => copyToken()}>
                                {copied ? "Copied" : "Copy"}
                            </Button>
                        </div>
                    </label>

                    <div className={styles.resetRow}>
                        <Button size="small" variant="ghost" onClick={() => handleReset()} disabled={resetting}>
                            {resetting ? "Resetting..." : "Reset token"}
                        </Button>
                        <span className={settings.mutedText}>
                            Use this if your token leaks. You'll need to re-download the connector.
                        </span>
                    </div>

                    <button
                        type="button"
                        className={styles.disclosureToggle}
                        onClick={() => setSetupOpen(open => !open)}
                        aria-expanded={setupOpen}
                    >
                        <span>SAMMI setup guide</span>
                        <span>{setupOpen ? "▾" : "▸"}</span>
                    </button>
                    {setupOpen && (
                        <ol className={styles.steps}>
                            <li>
                                In <strong>SAMMI Core {"→"} Settings</strong>, enable the Bridge / Deck websocket
                                server.
                            </li>
                            <li>
                                Download the connector above and import the <code>.sef</code> into SAMMI (Insert {"→"}{" "}
                                Extension).
                            </li>
                            <li>
                                Add a button that runs the <strong>Overlay: Connect</strong> command on deck load (or
                                SAMMI start).
                            </li>
                            <li>
                                Add a button with an <strong>Extension Trigger</strong> named <code>overlay_event</code>
                                . Use <strong>Trigger Pull Data</strong> to read the payload and branch on its{" "}
                                <code>type</code> field (<code>post_liked</code>, <code>new_follower</code>,{" "}
                                <code>post_commented</code>, ...). Show your overlay from there.
                            </li>
                            <li>
                                Click <strong>Send test overlay</strong> above to confirm it fires.
                            </li>
                        </ol>
                    )}
                </>
            )}
        </div>
    );
}
