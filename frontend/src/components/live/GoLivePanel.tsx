import { useState } from "react";
import { FPS_OPTIONS, MAX_BITRATE, MIN_BITRATE, STREAM_RESOLUTIONS } from "../../domain/live/bitrate";
import { LIVE_STATUS } from "../../domain/live/playback";
import { useGoLive } from "../../hooks/useGoLive";
import { Button } from "../Button/Button";
import { Input } from "../Input/Input";
import styles from "./GoLivePanel.module.css";

export function GoLivePanel() {
    const {
        owner,
        ownerError,
        credentials,
        credentialsError,
        smoothAvailable,
        title,
        setTitle,
        defaultMode,
        setDefaultMode,
        bitrate,
        setBitrate,
        calculator,
        titleEdit,
        start,
        stop,
        resetCredentials,
        canStart,
        busy,
        starting,
        stopping,
        resetting,
        error,
        copied,
        copy,
    } = useGoLive();

    const [setupOpen, setSetupOpen] = useState(false);

    return (
        <div className={styles.panel}>
            {owner ? (
                owner.stream.status === LIVE_STATUS ? (
                    <>
                        <h2 className={styles.heading}>You're live</h2>
                        {titleEdit.editing ? (
                            <div className={styles.field}>
                                <Input
                                    type="text"
                                    placeholder="Stream title"
                                    value={titleEdit.draft}
                                    onChange={e => titleEdit.setDraft(e.target.value)}
                                    maxLength={120}
                                    fullWidth
                                />
                                <div className={styles.actions}>
                                    <Button
                                        size="small"
                                        variant="primary"
                                        onClick={titleEdit.save}
                                        disabled={!titleEdit.canSave}
                                    >
                                        {titleEdit.saving ? "Saving..." : "Save title"}
                                    </Button>
                                    <Button
                                        size="small"
                                        variant="secondary"
                                        onClick={titleEdit.cancel}
                                        disabled={titleEdit.saving}
                                    >
                                        Cancel
                                    </Button>
                                </div>
                            </div>
                        ) : (
                            <div className={styles.field}>
                                <p className={styles.hint}>
                                    <strong>
                                        <bdi>{owner.stream.title}</bdi>
                                    </strong>{" "}
                                    is live. Stop here or close OBS to end it.
                                </p>
                                <div className={styles.actions}>
                                    <Button size="small" variant="ghost" onClick={titleEdit.open}>
                                        Edit title
                                    </Button>
                                </div>
                            </div>
                        )}
                        <div className={styles.actions}>
                            <Button variant="danger" onClick={stop} disabled={busy}>
                                {stopping ? "Stopping..." : "Stop streaming"}
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
                            <Button variant="danger" onClick={stop} disabled={busy}>
                                {stopping ? "Cancelling..." : "Cancel"}
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
                    {smoothAvailable && (
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
                    )}
                    {smoothAvailable && (
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Stream bitrate (Kbps)</span>
                            <Input
                                type="number"
                                placeholder="e.g. 6000"
                                value={bitrate}
                                onChange={e => setBitrate(e.target.value)}
                                min={MIN_BITRATE}
                                max={MAX_BITRATE}
                                fullWidth
                            />
                            <span className={styles.resetHint}>
                                Required. Set this to the bitrate you use in OBS, it is what Smooth playback encodes at.
                                Between {MIN_BITRATE.toLocaleString()} and {MAX_BITRATE.toLocaleString()} Kbps. Not
                                sure? Open OBS setup below for a calculator.
                            </span>
                        </div>
                    )}
                    <div className={styles.actions}>
                        <Button variant="primary" onClick={start} disabled={!canStart}>
                            {starting ? "Starting..." : "Go live"}
                        </Button>
                    </div>
                </>
            )}

            {error && <p className={styles.error}>{error}</p>}
            {ownerError && <p className={styles.error}>{ownerError}</p>}

            {credentials && (
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
                                {smoothAvailable
                                    ? "Server, key, encoder settings, and a bitrate calculator"
                                    : "Server, key, and encoder settings"}
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
                                        <code className={styles.code}>{credentials.whipUrl}</code>
                                        <Button
                                            size="small"
                                            variant="secondary"
                                            onClick={() => copy("url", credentials.whipUrl)}
                                        >
                                            {copied === "url" ? "Copied" : "Copy"}
                                        </Button>
                                    </div>
                                </div>

                                <div className={styles.field}>
                                    <span className={styles.fieldLabel}>Stream key (bearer token)</span>
                                    <div className={styles.copyRow}>
                                        <code className={styles.code}>{credentials.streamKey}</code>
                                        <Button
                                            size="small"
                                            variant="secondary"
                                            onClick={() => copy("key", credentials.streamKey)}
                                        >
                                            {copied === "key" ? "Copied" : "Copy"}
                                        </Button>
                                    </div>
                                </div>

                                <div className={styles.resetRow}>
                                    <Button
                                        size="small"
                                        variant="ghost"
                                        onClick={resetCredentials}
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

                            {smoothAvailable && (
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
                                                        className={
                                                            i === calculator.resolutionIndex
                                                                ? styles.pillActive
                                                                : styles.pill
                                                        }
                                                        onClick={() => calculator.setResolutionIndex(i)}
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
                                                        className={
                                                            f === calculator.fps ? styles.pillActive : styles.pill
                                                        }
                                                        onClick={() => calculator.setFps(f)}
                                                    >
                                                        {f} fps
                                                    </button>
                                                ))}
                                            </div>
                                        </div>
                                        <div className={styles.calcResult}>
                                            <span className={styles.calcResultMain}>
                                                {calculator.recommendation.typical.toLocaleString()}
                                                <span className={styles.calcUnit}> Kbps</span>
                                            </span>
                                            <span className={styles.calcResultSub}>
                                                {calculator.recommendation.low.toLocaleString()} to{" "}
                                                {calculator.recommendation.high.toLocaleString()} range, set as CBR
                                            </span>
                                        </div>
                                        <Button size="small" variant="secondary" onClick={calculator.applyTypical}>
                                            Use {calculator.recommendation.typical.toLocaleString()} Kbps
                                        </Button>
                                    </div>
                                </section>
                            )}

                            {smoothAvailable && (
                                <p className={styles.tip}>
                                    <strong>Low latency</strong> is your stream untouched. <strong>Smooth</strong> is
                                    re-encoded and a few seconds behind, but never freezes. Viewers choose.
                                </p>
                            )}
                        </div>
                    )}
                </div>
            )}
            {credentialsError && (
                <p className={styles.hint}>Could not load your stream key. Reload the page to try again.</p>
            )}
        </div>
    );
}
