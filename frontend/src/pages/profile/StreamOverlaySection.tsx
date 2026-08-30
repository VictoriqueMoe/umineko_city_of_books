import { useState } from "react";
import { Button } from "../../components/Button/Button";
import { useStreamOverlay } from "../../hooks/useStreamOverlay";
import settings from "./SettingsPage.module.css";
import styles from "./StreamOverlaySection.module.css";

export function StreamOverlaySection() {
    const overlay = useStreamOverlay();
    const [setupOpen, setSetupOpen] = useState(false);

    const connection = overlay.connection;

    return (
        <div className={settings.section}>
            <h3 className={settings.sectionTitle}>Stream Overlay</h3>
            <p className={styles.intro}>
                Fire overlay popups on your stream from site events (likes, follows, comments, theory votes) using
                SAMMI. Download the connector, import it into SAMMI, and your events appear live on stream.
            </p>

            {overlay.loading && <p className={settings.mutedText}>Loading your overlay connection...</p>}

            {overlay.loadError !== "" && (
                <>
                    <div className={settings.error} role="alert">
                        {overlay.loadError}
                    </div>
                    <p className={settings.mutedText}>
                        The overlay controls stay hidden until this loads. Reload the page to try again.
                    </p>
                </>
            )}

            {overlay.actionError !== "" && (
                <div className={settings.error} role="alert">
                    {overlay.actionError}
                </div>
            )}
            {overlay.success !== "" && <div className={settings.success}>{overlay.success}</div>}

            {connection && (
                <>
                    <div className={styles.statusRow}>
                        <span className={connection.connected ? styles.statusOn : styles.statusOff}>
                            {connection.connected ? "SAMMI connected" : "SAMMI not connected"}
                        </span>
                    </div>

                    <div className={styles.actions}>
                        <Button variant="primary" onClick={overlay.downloadConnector} disabled={overlay.downloading}>
                            {overlay.downloading ? "Preparing..." : "Download SAMMI connector (.sef)"}
                        </Button>
                        <Button variant="secondary" onClick={overlay.sendTestOverlay} disabled={overlay.testing}>
                            {overlay.testing ? "Sending..." : "Send test overlay"}
                        </Button>
                    </div>

                    <label className={settings.label}>
                        Connection token
                        <div className={styles.copyRow}>
                            <code className={styles.code}>{connection.token}</code>
                            <Button size="small" variant="secondary" onClick={overlay.copyToken}>
                                {overlay.copied ? "Copied" : "Copy"}
                            </Button>
                        </div>
                    </label>

                    <div className={styles.resetRow}>
                        <Button size="small" variant="ghost" onClick={overlay.resetToken} disabled={overlay.resetting}>
                            {overlay.resetting ? "Resetting..." : "Reset token"}
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
